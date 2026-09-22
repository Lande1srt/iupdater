package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

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

	// 批量把 download_url 转成完整直链
	resolveVersionList(c, versions)

	c.JSON(http.StatusOK, gin.H{
		"list":      versions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetVersion(c *gin.Context) {
	id := c.Param("id")

	var version models.AppVersion
	if err := models.DB.First(&version, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// resolve download_url 为完整直链
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
	downloadURL := c.PostForm("download_url") // 新增：用于元数据模式

	if appID == "" || versionName == "" || platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, version_name and platform are required"})
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

	// 元数据模式：使用外部下载链接，不上传文件
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

		// 可选：从表单读取文件哈希和大小
		fileHash = c.PostForm("file_hash")
		fileSizeStr := c.PostForm("file_size")
		if fileSizeStr != "" {
			fmt.Sscanf(fileSizeStr, "%d", &fileSize)
		} else {
			fileSize = 0
		}
	} else {
		// S3/WebDAV 模式：上传文件
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

	if channel == "" {
		channel = models.ChannelStable
	}
	if architecture == "" {
		architecture = "all"
	}
	if packageType == "" {
		packageType = models.PackageTypeFull
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

	// 使用 HEAD 请求获取文件信息，不下载内容
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

	if err := models.DB.Delete(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version deleted successfully"})
}
