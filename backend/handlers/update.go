package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"iupdater/models"
	"iupdater/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// resolveDownloadURL 根据存储配置把 DB 里存的 download_url 转换成客户端可用的完整下载链接
// 优先级：CDN BaseURL → redirect 直链 → proxy 绝对路径
// DB 里可能存三种格式：
//  1. https://xxx.com/file.exe  — 完整 URL（metadata 模式）
//  2. /api/download/uuid/file.exe — 代理路径（S3/WebDAV 默认）
//  3. uuid/file.exe — 裸 key（历史数据或某些存储）
func resolveDownloadURL(c *gin.Context, downloadURL string, app models.App) string {
	// 已经是完整 URL，原样返回
	if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
		return downloadURL
	}

	// 提取 fileKey：从 /api/download/uuid/file.exe 或直接 uuid/file.exe
	var fileKey string
	if strings.HasPrefix(downloadURL, "/api/download/") {
		fileKey = strings.TrimPrefix(downloadURL, "/api/download/")
	} else if strings.Contains(downloadURL, "/") {
		fileKey = downloadURL
	} else {
		return downloadURL
	}

	// proxy 模式需要拼绝对 URL 的辅助函数
	proxyURL := func() string {
		scheme := "http"
		if c != nil && (c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https") {
			scheme = "https"
		}
		host := "localhost:8080"
		if c != nil && c.Request.Host != "" {
			host = c.Request.Host
		}
		return fmt.Sprintf("%s://%s/api/download/%s", scheme, host, fileKey)
	}

	storageType, storageConfig, err := ResolveAppStorage(&app)
	if err != nil {
		return downloadURL
	}

	switch storageType {
	case "s3":
		var cfg storage.S3Config
		if err := json.Unmarshal([]byte(storageConfig), &cfg); err != nil {
			return downloadURL
		}
		objectKey := strings.TrimLeft(path.Join(cfg.BasePath, fileKey), "/")

		// 1) CDN BaseURL → 永久 CDN 直链
		if cfg.BaseURL != "" {
			return strings.TrimRight(cfg.BaseURL, "/") + "/" + objectKey
		}

		// 2) redirect 模式 → 预签名 S3 URL（15 分钟有效）
		if cfg.DownloadMode == "redirect" {
			client, err := storage.NewS3Client(cfg)
			if err == nil {
				presignClient := s3.NewPresignClient(client)
				req, err := presignClient.PresignGetObject(context.TODO(),
					&s3.GetObjectInput{
						Bucket: aws.String(cfg.Bucket),
						Key:    aws.String(objectKey),
					},
					func(o *s3.PresignOptions) {
						o.Expires = 15 * time.Minute
					},
				)
				if err == nil {
					return req.URL
				}
				log.Printf("[resolveDownloadURL] presign failed: %v", err)
			}
		}
		// 3) proxy 模式 → 返回绝对代理 URL
		return proxyURL()

	case "webdav":
		var cfg storage.WebDAVConfig
		if err := json.Unmarshal([]byte(storageConfig), &cfg); err != nil {
			return downloadURL
		}
		// redirect 模式 → 直连 WebDAV
		if cfg.DownloadMode != "proxy" {
			remotePath := path.Join(cfg.BasePath, fileKey)
			return strings.TrimRight(cfg.URL, "/") + "/" + strings.TrimLeft(remotePath, "/")
		}
		// proxy 模式 → 返回绝对代理 URL
		return proxyURL()
	}

	return downloadURL
}

// loadAppsMap 批量加载 app_id → App 映射，供列表 API 批量 resolve download_url 使用
func loadAppsMap(appIDs []string) map[string]models.App {
	if len(appIDs) == 0 {
		return nil
	}
	apps := make([]models.App, 0, len(appIDs))
	models.DB.Where("app_id IN ?", appIDs).Find(&apps)
	m := make(map[string]models.App, len(apps))
	for _, a := range apps {
		m[a.AppID] = a
	}
	return m
}

// resolveVersionList 批量把版本列表的 DownloadURL 转成完整直链
func resolveVersionList(c *gin.Context, versions []models.AppVersion) {
	appIDs := make([]string, 0)
	seen := make(map[string]bool)
	for _, v := range versions {
		if !seen[v.AppID] {
			seen[v.AppID] = true
			appIDs = append(appIDs, v.AppID)
		}
	}
	apps := loadAppsMap(appIDs)
	for i := range versions {
		if app, ok := apps[versions[i].AppID]; ok {
			versions[i].DownloadURL = resolveDownloadURL(c, versions[i].DownloadURL, app)
		}
	}
}

// resolveComponentList 批量把组件列表的 DownloadURL 转成完整直链
func resolveComponentList(c *gin.Context, components []models.Component) {
	appIDs := make([]string, 0)
	seen := make(map[string]bool)
	for _, comp := range components {
		if !seen[comp.AppID] {
			seen[comp.AppID] = true
			appIDs = append(appIDs, comp.AppID)
		}
	}
	apps := loadAppsMap(appIDs)
	for i := range components {
		if app, ok := apps[components[i].AppID]; ok {
			components[i].DownloadURL = resolveDownloadURL(c, components[i].DownloadURL, app)
		}
	}
}

type CheckUpdateRequest struct {
	AppID        string `json:"app_id" binding:"required"`
	VersionCode  int64  `json:"version_code" binding:"required"`
	Channel      string `json:"channel"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	AccessKey    string `json:"access_key"`
}

type CheckUpdateResponse struct {
	NeedUpdate    bool            `json:"need_update"`
	IsForceUpdate bool            `json:"is_force_update"`
	LatestVersion VersionInfo     `json:"latest_version"`
	Components    []ComponentInfo `json:"components"`
}

type VersionInfo struct {
	AppID        string `json:"app_id"`
	VersionCode  int64  `json:"version_code"`
	VersionName  string `json:"version_name"`
	Channel      string `json:"channel"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	PackageType  string `json:"package_type"`
	FileSize     int64  `json:"file_size"`
	FileHash     string `json:"file_hash"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
}

type ComponentInfo struct {
	ComponentKey string `json:"component_key"`
	Name         string `json:"name"`
	VersionCode  int64  `json:"version_code"`
	VersionName  string `json:"version_name"`
	FileSize     int64  `json:"file_size"`
	FileHash     string `json:"file_hash"`
	DownloadURL  string `json:"download_url"`
}

func CheckUpdate(c *gin.Context) {
	var req CheckUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证应用访问权限
	var app models.App
	if err := models.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	// 如果启用了校验，验证 AccessKey
	if app.EnableAuth {
		if req.AccessKey == "" || req.AccessKey != app.AccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"})
			return
		}
	}

	channel := req.Channel
	if channel == "" {
		channel = models.ChannelStable
	}

	platform := req.Platform
	if platform == "" {
		platform = "all"
	}

	arch := req.Architecture
	if arch == "" {
		arch = "all"
	}

	var latestVersion models.AppVersion
	result := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		req.AppID, channel, models.PackageTypeFull).
		Where("platform IN ?", []string{platform, "all"}).
		Where("architecture IN ?", []string{arch, "all"}).
		Order("version_code DESC").
		First(&latestVersion)

	resp := CheckUpdateResponse{
		NeedUpdate: false,
		Components: []ComponentInfo{},
	}

	if result.Error == nil && latestVersion.VersionCode > req.VersionCode {
		resp.NeedUpdate = true
		resp.IsForceUpdate = latestVersion.IsForceUpdate
		resp.LatestVersion = VersionInfo{
			AppID:        latestVersion.AppID,
			VersionCode:  latestVersion.VersionCode,
			VersionName:  latestVersion.VersionName,
			Channel:      latestVersion.Channel,
			Platform:     latestVersion.Platform,
			Architecture: latestVersion.Architecture,
			PackageType:  latestVersion.PackageType,
			FileSize:     latestVersion.FileSize,
			FileHash:     latestVersion.FileHash,
			DownloadURL:  resolveDownloadURL(c, latestVersion.DownloadURL, app),
			ReleaseNotes: latestVersion.ReleaseNotes,
		}
	}

	var components []models.Component
	models.DB.Where("app_id = ? AND channel = ? AND status = 1", req.AppID, channel).
		Where("platform IN ?", []string{platform, "all"}).
		Where("architecture IN ?", []string{arch, "all"}).
		Find(&components)

	for _, comp := range components {
		resp.Components = append(resp.Components, ComponentInfo{
			ComponentKey: comp.ComponentKey,
			Name:         comp.Name,
			VersionCode:  comp.VersionCode,
			VersionName:  comp.VersionName,
			FileSize:     comp.FileSize,
			FileHash:     comp.FileHash,
			DownloadURL:  resolveDownloadURL(c, comp.DownloadURL, app),
		})
	}

	c.JSON(http.StatusOK, resp)
}

// GetLatestVersion 通过 GET 获取最新版本信息
// 用法: GET /u/:appid?channel=stable&platform=android&access_key=xxx
func GetLatestVersion(c *gin.Context) {
	appID := c.Param("appid")
	if appID == "" {
		appID = c.Query("appid") // 兼容旧查询参数方式
	}
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appid is required"})
		return
	}

	// 查找应用
	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	// 验证 AccessKey
	if app.EnableAuth {
		accessKey := c.Query("access_key")
		if accessKey == "" || accessKey != app.AccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"})
			return
		}
	}

	// 查询参数
	channel := c.DefaultQuery("channel", models.ChannelStable)
	platform := c.Query("platform")
	arch := c.Query("architecture")

	// 构建查询
	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		appID, channel, models.PackageTypeFull)

	// 显式传了 platform 才过滤，否则匹配所有
	if platform != "" {
		query = query.Where("platform IN ?", []string{platform, "all"})
	}
	if arch != "" {
		query = query.Where("architecture IN ?", []string{arch, "all"})
	}

	// 查询最新版本
	var latestVersion models.AppVersion
	result := query.Order("version_code DESC").First(&latestVersion)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No published version found", "channel": channel, "platform": platform})
		return
	}

	// 查询组件
	compQuery := models.DB.Where("app_id = ? AND channel = ? AND status = 1", appID, channel)
	if platform != "" {
		compQuery = compQuery.Where("platform IN ?", []string{platform, "all"})
	}
	if arch != "" {
		compQuery = compQuery.Where("architecture IN ?", []string{arch, "all"})
	}
	var components []models.Component
	compQuery.Find(&components)

	compInfos := make([]ComponentInfo, 0, len(components))
	for _, comp := range components {
		compInfos = append(compInfos, ComponentInfo{
			ComponentKey: comp.ComponentKey,
			Name:         comp.Name,
			VersionCode:  comp.VersionCode,
			VersionName:  comp.VersionName,
			FileSize:     comp.FileSize,
			FileHash:     comp.FileHash,
			DownloadURL:  resolveDownloadURL(c, comp.DownloadURL, app),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"app_id": appID,
		"latest_version": VersionInfo{
			AppID:        latestVersion.AppID,
			VersionCode:  latestVersion.VersionCode,
			VersionName:  latestVersion.VersionName,
			Channel:      latestVersion.Channel,
			Platform:     latestVersion.Platform,
			Architecture: latestVersion.Architecture,
			PackageType:  latestVersion.PackageType,
			FileSize:     latestVersion.FileSize,
			FileHash:     latestVersion.FileHash,
			DownloadURL:  resolveDownloadURL(c, latestVersion.DownloadURL, app),
			ReleaseNotes: latestVersion.ReleaseNotes,
		},
		"components":      compInfos,
		"is_force_update": latestVersion.IsForceUpdate,
	})
}

// GetLatestLink 直接返回最新版本的下载链接（纯 URL 字符串）
// 用法: GET /u/getlink/:appid?channel=stable&platform=windows&architecture=all&access_key=xxx
// 返回: https://cdn.example.com/path/file.exe  或  http://host/api/download/uuid/file.exe
func GetLatestLink(c *gin.Context) {
	appID := c.Param("appid")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appid is required"})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	if app.EnableAuth {
		accessKey := c.Query("access_key")
		if accessKey == "" || accessKey != app.AccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"})
			return
		}
	}

	channel := c.DefaultQuery("channel", models.ChannelStable)
	platform := c.Query("platform")
	arch := c.Query("architecture")

	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		appID, channel, models.PackageTypeFull)
	if platform != "" {
		query = query.Where("platform IN ?", []string{platform, "all"})
	}
	if arch != "" {
		query = query.Where("architecture IN ?", []string{arch, "all"})
	}

	var latestVersion models.AppVersion
	if err := query.Order("version_code DESC").First(&latestVersion).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No published version found"})
		return
	}

	link := resolveDownloadURL(c, latestVersion.DownloadURL, app)
	c.Redirect(http.StatusFound, link)
}
