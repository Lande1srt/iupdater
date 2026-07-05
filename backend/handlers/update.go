package handlers

import (
	"net/http"

	"iupdater/models"

	"github.com/gin-gonic/gin"
)

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
			DownloadURL:  latestVersion.DownloadURL,
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
			DownloadURL:  comp.DownloadURL,
		})
	}

	c.JSON(http.StatusOK, resp)
}
