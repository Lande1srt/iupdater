package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"iupdater/models"
	"iupdater/storage"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
)

type CreateVersionRequest struct {
	AppID         string `json:"app_id" binding:"required"`
	VersionName   string `json:"version_name" binding:"required"`
	Channel       string `json:"channel"`
	Platform      string `json:"platform" binding:"required"`
	Architecture  string `json:"architecture"`
	PackageType   string `json:"package_type"`
	IsForceUpdate bool   `json:"is_force_update"`
	DownloadURL   string `json:"download_url"`
	FileHash      string `json:"file_hash"`
	FileSize      int64  `json:"file_size"`
	ReleaseNotes  string `json:"release_notes"`
}

// checkVersionDuplicate 检查同 app + channel + version_name + platform + architecture 是否已存在
// 返回已存在的版本（如果存在），或者 nil
func checkVersionDuplicate(appID, channel, versionName, platform, architecture, packageType string) (*models.AppVersion, error) {
	arch := architecture
	if arch == "" {
		arch = "all"
	}
	pkg := packageType
	if pkg == "" {
		pkg = models.PackageTypeFull
	}
	var existing models.AppVersion
	err := models.DB.Where(
		"app_id = ? AND channel = ? AND version_name = ? AND platform = ? AND architecture = ? AND package_type = ?",
		appID, channel, versionName, platform, arch, pkg,
	).First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func GetVersions(c *gin.Context) {
	appID := c.Query("app_id")
	channel := c.Query("channel")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var versions []models.AppVersion
	var total int64

	query := models.DB.Model(&models.AppVersion{})
	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("version_code DESC").Find(&versions)

	resolveVersionList(c, versions)

	c.JSON(http.StatusOK, gin.H{
		"list":      versions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetVersionGroups 按 (app_id, channel, version_name) 归组，组内各平台聚合在一起。
// 前端用来「选择已有版本组」场景，也给列表页做分组表格。
// 每个组返回：version_name / channel / latest_version_code / platforms[] / versions[]
func GetVersionGroups(c *gin.Context) {
	appID := c.Query("app_id")
	channel := c.Query("channel")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	var versions []models.AppVersion
	q := models.DB.Where("app_id = ?", appID)
	if channel != "" {
		q = q.Where("channel = ?", channel)
	}
	if err := q.Order("version_code DESC").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resolveVersionList(c, versions)

	type Group struct {
		VersionName       string              `json:"version_name"`
		Channel           string              `json:"channel"`
		PackageType       string              `json:"package_type"`
		LatestVersionCode int64               `json:"latest_version_code"`
		Platforms         []string            `json:"platforms"`
		Versions          []models.AppVersion `json:"versions"`
	}

	groupsMap := map[string]*Group{}
	// 默认 channel / package_type 顺序稳定的 key
	for _, v := range versions {
		if v.Channel == "" {
			v.Channel = models.ChannelStable
		}
		if v.PackageType == "" {
			v.PackageType = models.PackageTypeFull
		}
		key := v.Channel + "||" + v.PackageType + "||" + v.VersionName
		g, ok := groupsMap[key]
		if !ok {
			g = &Group{
				VersionName:       v.VersionName,
				Channel:           v.Channel,
				PackageType:       v.PackageType,
				LatestVersionCode: v.VersionCode,
				Platforms:         []string{},
				Versions:          []models.AppVersion{},
			}
			groupsMap[key] = g
		}
		g.Versions = append(g.Versions, v)
		if v.VersionCode > g.LatestVersionCode {
			g.LatestVersionCode = v.VersionCode
		}
		plat := v.Platform
		if plat == "" {
			plat = "all"
		}
		g.Platforms = append(g.Platforms, plat)
	}

	// 转成数组 + 按最新 version_code 降序
	groups := make([]Group, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}
	// 简单排序：最新版本号最大的组放最前
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[j].LatestVersionCode > groups[i].LatestVersionCode {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"list":  groups,
		"total": len(groups),
	})
}

func GetVersion(c *gin.Context) {
	id := c.Param("id")

	var version models.AppVersion
	if err := models.DB.First(&version, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", version.AppID).First(&app).Error; err == nil {
		version.DownloadURL = resolveDownloadURL(c, version.DownloadURL, app)
	}

	c.JSON(http.StatusOK, version)
}

func CreateVersion(c *gin.Context) {
	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	channel := req.Channel
	if channel == "" {
		channel = models.ChannelStable
	}
	arch := req.Architecture
	if arch == "" {
		arch = "all"
	}
	pkgType := req.PackageType
	if pkgType == "" {
		pkgType = models.PackageTypeFull
	}

	// 重复检查
	if existing, _ := checkVersionDuplicate(req.AppID, channel, req.VersionName, req.Platform, arch, pkgType); existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":                 fmt.Sprintf("已存在版本 %s 的 %s/%s (%s)", req.VersionName, req.Platform, arch, channel),
			"existing_version_code": existing.VersionCode,
			"existing_id":           existing.ID,
		})
		return
	}

	versionCode, err := utils.GenerateVersionCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	version := models.AppVersion{
		AppID:         req.AppID,
		VersionCode:   versionCode,
		VersionName:   req.VersionName,
		Channel:       channel,
		Platform:      req.Platform,
		Architecture:  arch,
		PackageType:   pkgType,
		IsForceUpdate: req.IsForceUpdate,
		FileSize:      req.FileSize,
		FileHash:      req.FileHash,
		DownloadURL:   req.DownloadURL,
		ReleaseNotes:  req.ReleaseNotes,
		Status:        1,
	}

	if err := models.DB.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	version.DownloadURL = resolveDownloadURL(c, version.DownloadURL, app)
	c.JSON(http.StatusOK, version)
}

func UploadVersionFile(c *gin.Context) {
	appID := c.PostForm("app_id")
	versionName := c.PostForm("version_name")
	channel := c.PostForm("channel")
	platform := c.PostForm("platform")
	architecture := c.PostForm("architecture")
	packageType := c.PostForm("package_type")
	isForceUpdate := c.PostForm("is_force_update") == "true"
	releaseNotes := c.PostForm("release_notes")
	downloadURL := c.PostForm("download_url")

	if appID == "" || versionName == "" || platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, version_name and platform are required"})
		return
	}

	if channel == "" {
		channel = models.ChannelStable
	}
	if architecture == "" {
		architecture = "all"
	}
	if packageType == "" {
		packageType = models.PackageTypeFull
	}

	// 重复检查（在真的上传之前就拦，避免白传）
	if existing, _ := checkVersionDuplicate(appID, channel, versionName, platform, architecture, packageType); existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("已存在版本 %s 的 %s/%s (%s)，请删除后再上传或选择新建版本名称",
				versionName, platform, architecture, channel),
			"existing_version_code": existing.VersionCode,
			"existing_id":           existing.ID,
		})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	var fileHash string
	var fileSize int64
	var finalDownloadURL string

	storageType, storageConfig, err := ResolveAppStorage(&app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage profile not found"})
		return
	}

	if storageType == "metadata" {
		if downloadURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "download_url is required for metadata storage"})
			return
		}
		finalDownloadURL = downloadURL
		fileHash = c.PostForm("file_hash")
		fileSizeStr := c.PostForm("file_size")
		if fileSizeStr != "" {
			fmt.Sscanf(fileSizeStr, "%d", &fileSize)
		} else {
			fileSize = 0
		}
	} else {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File is required for S3/WebDAV storage"})
			return
		}
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}

		fileHash = utils.ComputeBytesHash(fileData)

		log.Printf("[Upload] app_id=%s storage_type=%s storage_config_len=%d file_bytes=%d",
			appID, storageType, len(storageConfig), len(fileData))

		adapter, err := storage.NewStorageAdapter(storageType, storageConfig)
		if err != nil {
			log.Printf("[Upload] NewStorageAdapter FAILED: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to init storage: " + err.Error()})
			return
		}

		var key string
		key, fileSize, err = adapter.Upload(header.Filename, fileData)
		if err != nil {
			log.Printf("[Upload] adapter.Upload FAILED: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file: " + err.Error()})
			return
		}
		log.Printf("[Upload] SUCCESS key=%q size=%d download_url=%q", key, fileSize, adapter.GetURL(key))
		finalDownloadURL = adapter.GetURL(key)
	}

	versionCode, err := utils.GenerateVersionCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	version := models.AppVersion{
		AppID:         appID,
		VersionCode:   versionCode,
		VersionName:   versionName,
		Channel:       channel,
		Platform:      platform,
		Architecture:  architecture,
		PackageType:   packageType,
		IsForceUpdate: isForceUpdate,
		FileSize:      fileSize,
		FileHash:      fileHash,
		DownloadURL:   finalDownloadURL,
		ReleaseNotes:  releaseNotes,
		Status:        1,
	}

	if err := models.DB.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	version.DownloadURL = resolveDownloadURL(c, version.DownloadURL, app)
	c.JSON(http.StatusOK, version)
}

func ProbeFileInfo(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := http.Head(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to probe URL: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Remote server returned status %d", resp.StatusCode)})
		return
	}

	fileSize := resp.ContentLength
	if fileSize < 0 {
		c.JSON(http.StatusOK, gin.H{
			"file_size": 0,
			"message":   "Server did not provide Content-Length header",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_size": fileSize,
	})
}

func UpdateVersionStatus(c *gin.Context) {
	id := c.Param("id")

	var version models.AppVersion
	if err := models.DB.First(&version, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version.Status = req.Status
	if err := models.DB.Save(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", version.AppID).First(&app).Error; err == nil {
		version.DownloadURL = resolveDownloadURL(c, version.DownloadURL, app)
	}
	c.JSON(http.StatusOK, version)
}

func DeleteVersion(c *gin.Context) {
	id := c.Param("id")

	var version models.AppVersion
	if err := models.DB.First(&version, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// ====== 先删后端存储中的文件（metadata 外部链接跳过）======
	app, storageType, storageConfig := resolveAppAndStorage(c, version.AppID)
	if app != nil && storageType != "" && storageType != "metadata" {
		if adapter, err := storage.NewStorageAdapter(storageType, storageConfig); err == nil {
			key := stripDownloadKey(version.DownloadURL)
			if key != "" {
				if err := adapter.Delete(key); err != nil {
					log.Printf("[DeleteVersion] 存储文件删除失败 key=%s: %v (继续删除数据库记录)", key, err)
				} else {
					log.Printf("[DeleteVersion] 存储文件已删除 key=%s", key)
				}
			}
		} else {
			log.Printf("[DeleteVersion] NewStorageAdapter 失败，跳过存储文件删除: %v", err)
		}
	}

	if err := models.DB.Delete(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version deleted successfully"})
}

// resolveAppAndStorage 组合: 查 App 并 ResolveAppStorage
func resolveAppAndStorage(c *gin.Context, appID string) (*models.App, string, string) {
	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		return nil, "", ""
	}
	stype, sconf, _ := ResolveAppStorage(&app)
	return &app, stype, sconf
}

// stripDownloadKey 把数据库里存的 download_url 还原成 storage adapter 需要的裸 key
//
//	"/api/download/uuid/file.exe" → "uuid/file.exe"
//	"uuid/file.exe"               → "uuid/file.exe"  (已裸 key)
//	"https://cdn.example.com/..." → ""               (外部链接，跳过)
//	""                            → ""
func stripDownloadKey(url string) string {
	if url == "" {
		return ""
	}
	// 绝对 URL 跳过（外部链接 / CDN 直连）
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return ""
	}
	// /api/download/xxx → xxx
	const prefix = "/api/download/"
	if strings.HasPrefix(url, prefix) {
		return url[len(prefix):]
	}
	// 其他相对路径形式（历史数据）直接返回
	return url
}
