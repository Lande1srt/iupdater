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

// normalizePlatform / normalizeArchitecture 把用户传入的常见别名归一成数据库存储时用的 canonical 值
// 存什么值由上传方（admin UI / agent）决定，这里尽量兼容：
//   platform:   保持原样
//   arch:       x86_64→amd64, aarch64→arm64, i386/x86→386
// archMatchList 生成的 IN 列表包含 canonical 值 + 它的常见别名 + "all"，确保无论 DB 里存什么写法都能命中

// archCanonical: 把常见架构别名统一到 canonical 值（Go runtime / 主流工具链的写法）
var archCanonical = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
	"i386":    "386",
	"x86":     "386",
	"amd64":   "amd64",
	"arm64":   "arm64",
	"386":     "386",
	"all":     "all",
}

func normalizeArchitecture(in string) string {
	if in == "" {
		return in
	}
	if v, ok := archCanonical[strings.ToLower(in)]; ok {
		return v
	}
	return in
}

// matchPlatformArch 构造 WHERE 平台/架构匹配子句。
// 兼容两种存储值：用户上传时可能填 Go runtime 值（amd64/arm64）也可能填通用值（x86_64/aarch64），
// 这里生成 IN 列表把「用户传入的 canonical 值 + 它的别名 + all」都包含进去，保证能命中。
func archMatchList(canonical string) []string {
	out := []string{canonical, "all"}
	if canonical == "amd64" {
		out = append(out, "x86_64")
	} else if canonical == "arm64" {
		out = append(out, "aarch64")
	} else if canonical == "386" {
		out = append(out, "i386", "x86")
	}
	return out
}

func platformMatchList(platform string) []string {
	return []string{platform, "all"}
}

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

	platform := strings.ToLower(req.Platform)
	if platform == "" {
		platform = "all"
	}

	arch := normalizeArchitecture(strings.ToLower(req.Architecture))
	if arch == "" {
		arch = "all"
	}

	var latestVersion models.AppVersion
	result := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		req.AppID, channel, models.PackageTypeFull).
		Where("platform IN ?", platformMatchList(platform)).
		Where("architecture IN ?", archMatchList(arch)).
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
		Where("platform IN ?", platformMatchList(platform)).
		Where("architecture IN ?", archMatchList(arch)).
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

	// 查询参数 —— 先做 alias 归一化（x86_64 → amd64, aarch64 → arm64 等）
	channel := c.DefaultQuery("channel", models.ChannelStable)
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	archRaw := strings.ToLower(strings.TrimSpace(c.Query("architecture")))
	arch := normalizeArchitecture(archRaw)

	// 1) 最新版本 code —— 如果用户传了 platform/arch，就在指定范围内找 MAX；
	//    这样避免跨平台独立自增 version_code 时，MAX 拿到别的平台的版本组
	//    显式 .Table("app_versions") 避免 GORM 在 Select+Scan 非 Find 路径下丢失表名绑定
	maxQuery := models.DB.Table("app_versions").Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		appID, channel, models.PackageTypeFull)
	if platform != "" {
		maxQuery = maxQuery.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		maxQuery = maxQuery.Where("architecture IN ?", archMatchList(arch))
	}
	var latestCode int64
	if err := maxQuery.Select("MAX(version_code)").Scan(&latestCode).Error; err != nil || latestCode == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No published version found", "channel": channel})
		return
	}

	// 2) 在这个版本组里匹配
	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ? AND version_code = ?",
		appID, channel, models.PackageTypeFull, latestCode)
	if platform != "" {
		query = query.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		query = query.Where("architecture IN ?", archMatchList(arch))
	}
	var candidates []models.AppVersion
	if err := query.Order("version_code DESC").Find(&candidates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(candidates) == 0 {
		// 额外诊断：抓一下这个版本组里实际有哪些 platform/arch，方便生产排错
		var inGroup []struct{ Platform, Architecture, PackageType string; Status int }
		models.DB.Where("app_id = ? AND channel = ? AND version_code = ?", appID, channel, latestCode).
			Distinct("platform", "architecture", "package_type", "status").
			Find(&inGroup)
		c.JSON(http.StatusNotFound, gin.H{
			"error":                     "No published version found for the given filters",
			"channel":                   channel,
			"platform":                  platform,
			"architecture":              arch,
			"latest_version_group_code": latestCode,
			"group_actual":              inGroup,
			"hint":                      "版本组内未匹配指定 platform/architecture，请检查参数；architecture 别名可用 x86_64/amd64、aarch64/arm64",
		})
		return
	}

	// 用户未指定 platform/arch 时的智能处理
	if platform == "" && arch == "" {
		if len(candidates) > 1 {
			// 多个平台 —— 报错提示必须指定
			var platforms []string
			seen := map[string]bool{}
			for _, v := range candidates {
				if !seen[v.Platform] {
					seen[v.Platform] = true
					platforms = append(platforms, v.Platform)
				}
			}
			c.JSON(http.StatusConflict, gin.H{
				"error":              "最新版本组存在多平台，请指定 platform 参数",
				"platforms":          platforms,
				"hint":               "使用 ?platform=windows 或 ?platform=linux&architecture=amd64",
				"version_group_code": latestCode,
			})
			return
		}
		// 唯一候选 —— 直接返回
	}

	latestVersion := candidates[0]

	// 查询组件
	compQuery := models.DB.Where("app_id = ? AND channel = ? AND status = 1", appID, channel)
	if platform != "" {
		compQuery = compQuery.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		compQuery = compQuery.Where("architecture IN ?", archMatchList(arch))
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
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	archRaw := strings.ToLower(strings.TrimSpace(c.Query("architecture")))
	arch := normalizeArchitecture(archRaw)

	// 1) 最新版本 code —— 用户传了 platform/arch 时在指定范围内找 MAX
	maxQuery := models.DB.Table("app_versions").Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		appID, channel, models.PackageTypeFull)
	if platform != "" {
		maxQuery = maxQuery.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		maxQuery = maxQuery.Where("architecture IN ?", archMatchList(arch))
	}
	var latestCode int64
	if err := maxQuery.Select("MAX(version_code)").Scan(&latestCode).Error; err != nil || latestCode == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No published version found"})
		return
	}

	// 2) 在这个版本组里匹配
	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ? AND version_code = ?",
		appID, channel, models.PackageTypeFull, latestCode)
	if platform != "" {
		query = query.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		query = query.Where("architecture IN ?", archMatchList(arch))
	}
	var candidates []models.AppVersion
	if err := query.Order("version_code DESC").Find(&candidates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(candidates) == 0 {
		var inGroup []struct{ Platform, Architecture, PackageType string; Status int }
		models.DB.Where("app_id = ? AND channel = ? AND version_code = ?", appID, channel, latestCode).
			Distinct("platform", "architecture", "package_type", "status").
			Find(&inGroup)
		c.JSON(http.StatusNotFound, gin.H{
			"error":                     "No published version found for the given filters",
			"platform":                  platform,
			"architecture":              arch,
			"latest_version_group_code": latestCode,
			"group_actual":              inGroup,
			"hint":                      "请检查 platform/architecture 是否与数据库一致；architecture 别名可用 x86_64/amd64、aarch64/arm64",
		})
		return
	}

	if platform == "" && arch == "" && len(candidates) > 1 {
		var platforms []string
		seen := map[string]bool{}
		for _, v := range candidates {
			if !seen[v.Platform] {
				seen[v.Platform] = true
				platforms = append(platforms, v.Platform)
			}
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":     "最新版本组存在多平台，请指定 platform 查询参数",
			"platforms": platforms,
			"hint":      "GET /u/getlink/:appid?platform=windows 或 /u/getlink/:appid?platform=linux&architecture=amd64",
		})
		return
	}

	latestVersion := candidates[0]

	link := resolveDownloadURL(c, latestVersion.DownloadURL, app)
	c.Redirect(http.StatusFound, link)
}

// ---------- 公开元数据接口（新） ----------

// VersionMeta 公开返回的版本元数据（不暴露 internal 字段）
type VersionMeta struct {
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
	IsForce      bool   `json:"is_force_update"`
	Status       int    `json:"status"`
	CreatedAt    string `json:"created_at"`
}

func toVersionMeta(c *gin.Context, app models.App, v models.AppVersion) VersionMeta {
	return VersionMeta{
		AppID:        v.AppID,
		VersionCode:  v.VersionCode,
		VersionName:  v.VersionName,
		Channel:      v.Channel,
		Platform:     v.Platform,
		Architecture: v.Architecture,
		PackageType:  v.PackageType,
		FileSize:     v.FileSize,
		FileHash:     v.FileHash,
		DownloadURL:  resolveDownloadURL(c, v.DownloadURL, app),
		ReleaseNotes: v.ReleaseNotes,
		IsForce:      v.IsForceUpdate,
		Status:       v.Status,
		CreatedAt:    v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// sharedLatestVersionQuery 抽取 GetMeta / GetLatestVersion / GetLatestLink 共用的两步查询
// 返回 (candidates, latestCode, app, errCode, errBody)
// errCode == 0 表示成功，errCode != 0 时调用方应直接 c.JSON 返回 errBody
func sharedLatestVersionQuery(c *gin.Context) ([]models.AppVersion, int64, models.AppVersion, int, gin.H) {
	appID := c.Param("appid")
	if appID == "" {
		appID = c.Query("appid")
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		return nil, 0, models.AppVersion{}, http.StatusNotFound, gin.H{"error": "App not found"}
	}

	if app.EnableAuth {
		ak := c.Query("access_key")
		if ak == "" || ak != app.AccessKey {
			return nil, 0, models.AppVersion{}, http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"}
		}
	}

	channel := c.DefaultQuery("channel", models.ChannelStable)
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	archRaw := strings.ToLower(strings.TrimSpace(c.Query("architecture")))
	arch := normalizeArchitecture(archRaw)

	// 1) MAX
	maxQuery := models.DB.Table("app_versions").Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		appID, channel, models.PackageTypeFull)
	if platform != "" {
		maxQuery = maxQuery.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		maxQuery = maxQuery.Where("architecture IN ?", archMatchList(arch))
	}
	var latestCode int64
	if err := maxQuery.Select("MAX(version_code)").Scan(&latestCode).Error; err != nil || latestCode == 0 {
		return nil, 0, models.AppVersion{}, http.StatusNotFound, gin.H{"error": "No published version found", "channel": channel}
	}

	// 2) 在这个版本组里匹配
	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ? AND version_code = ?",
		appID, channel, models.PackageTypeFull, latestCode)
	if platform != "" {
		query = query.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		query = query.Where("architecture IN ?", archMatchList(arch))
	}
	var candidates []models.AppVersion
	if err := query.Order("version_code DESC").Find(&candidates).Error; err != nil {
		return nil, 0, models.AppVersion{}, http.StatusInternalServerError, gin.H{"error": err.Error()}
	}

	if len(candidates) == 0 {
		var inGroup []struct{ Platform, Architecture, PackageType string; Status int }
		models.DB.Where("app_id = ? AND channel = ? AND version_code = ?", appID, channel, latestCode).
			Distinct("platform", "architecture", "package_type", "status").
			Find(&inGroup)
		return nil, latestCode, models.AppVersion{}, http.StatusNotFound, gin.H{
			"error":                     "No published version found for the given filters",
			"channel":                   channel,
			"platform":                  platform,
			"architecture":              arch,
			"latest_version_group_code": latestCode,
			"group_actual":              inGroup,
			"hint":                      "请检查 platform/architecture 参数；architecture 别名可用 x86_64/amd64、aarch64/arm64",
		}
	}

	// 无 platform/arch 且候选 > 1 → 409
	if platform == "" && arch == "" && len(candidates) > 1 {
		var platforms []string
		seen := map[string]bool{}
		for _, v := range candidates {
			if !seen[v.Platform] {
				seen[v.Platform] = true
				platforms = append(platforms, v.Platform)
			}
		}
		return nil, latestCode, models.AppVersion{}, http.StatusConflict, gin.H{
			"error":              "最新版本组存在多平台，请指定 platform 参数",
			"platforms":          platforms,
			"hint":               "GET /u/getmeta/:appid?platform=windows 或 /u/getmeta/:appid?platform=linux&architecture=amd64",
			"version_group_code": latestCode,
		}
	}

	return candidates, latestCode, models.AppVersion{}, 0, nil
}

// GetMeta 返回最新版本的元数据 JSON（重写原 /u/:appid）
// 用法: GET /u/getmeta/:appid?channel=stable&platform=windows&architecture=x86_64
func GetMeta(c *gin.Context) {
	appID := c.Param("appid")

	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}
	if app.EnableAuth {
		if ak := c.Query("access_key"); ak == "" || ak != app.AccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"})
			return
		}
	}

	candidates, _, _, errCode, errBody := sharedLatestVersionQuery(c)
	if errCode != 0 {
		c.JSON(errCode, errBody)
		return
	}
	c.JSON(http.StatusOK, toVersionMeta(c, app, candidates[0]))
}

// ListMeta 列出指定应用的所有已发布版本元数据
// 用法: GET /u/listmeta/:appid?channel=stable&platform=windows&architecture=x86_64&limit=50&include_status=0
//   channel / platform / architecture  过滤；都可省
//   order                              "desc"(默认，按 version_code DESC) 或 "asc"
//   limit                              默认 50，<=0 则不限制（上限 500）
//   include_status                     "1" 包含 status != 1 的历史版本（默认只返回 status=1）
func ListMeta(c *gin.Context) {
	appID := c.Param("appid")

	var app models.App
	if err := models.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}
	if app.EnableAuth {
		if ak := c.Query("access_key"); ak == "" || ak != app.AccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing access key"})
			return
		}
	}

	channel := c.DefaultQuery("channel", "")
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	archRaw := strings.ToLower(strings.TrimSpace(c.Query("architecture")))
	arch := normalizeArchitecture(archRaw)
	order := strings.ToLower(c.DefaultQuery("order", "desc"))
	if order != "asc" {
		order = "desc"
	}
	includeStatus := c.Query("include_status") == "1"

	var limit int
	if _, err := fmt.Sscanf(c.DefaultQuery("limit", "50"), "%d", &limit); err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	q := models.DB.Table("app_versions").Where("app_id = ?", appID)
	if channel != "" {
		q = q.Where("channel = ?", channel)
	}
	if !includeStatus {
		q = q.Where("status = 1")
	}
	if platform != "" {
		q = q.Where("platform IN ?", platformMatchList(platform))
	}
	if arch != "" {
		q = q.Where("architecture IN ?", archMatchList(arch))
	}

	var versions []models.AppVersion
	if err := q.Order("version_code " + strings.ToUpper(order)).Limit(limit).Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]VersionMeta, 0, len(versions))
	for _, v := range versions {
		out = append(out, toVersionMeta(c, app, v))
	}

	c.JSON(http.StatusOK, gin.H{
		"app_id":   appID,
		"count":    len(out),
		"channel":  channel,
		"platform": platform,
		"arch":     arch,
		"order":    order,
		"limit":    limit,
		"versions": out,
	})
}

// RedirectToGetMeta 旧 /u/:appid 的 301 永久重定向，兼容历史客户端
func RedirectToGetMeta(c *gin.Context) {
	target := "/u/getmeta/" + c.Param("appid")
	if q := c.Request.URL.RawQuery; q != "" {
		target += "?" + q
	}
	c.Redirect(http.StatusMovedPermanently, target)
}
