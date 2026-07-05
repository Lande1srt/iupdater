package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"iupdater/models"
	"iupdater/storage"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
)

type CreateComponentRequest struct {
	AppID        string `json:"app_id" binding:"required"`
	ComponentKey string `json:"component_key" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	VersionName  string `json:"version_name" binding:"required"`
	Channel      string `json:"channel"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	DownloadURL  string `json:"download_url"`
	FileHash     string `json:"file_hash"`
	FileSize     int64  `json:"file_size"`
}

func GetComponents(c *gin.Context) {
	appID := c.Query("app_id")
	channel := c.Query("channel")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var components []models.Component
	var total int64

	query := models.DB.Model(&models.Component{})
	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&components)

	c.JSON(http.StatusOK, gin.H{
		"list":      components,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetComponent(c *gin.Context) {
	id := c.Param("id")

	var component models.Component
	if err := models.DB.First(&component, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
		return
	}

	c.JSON(http.StatusOK, component)
}

func CreateComponent(c *gin.Context) {
	var req CreateComponentRequest
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

	platform := req.Platform
	if platform == "" {
		platform = "all"
	}

	arch := req.Architecture
	if arch == "" {
		arch = "all"
	}

	versionCode, err := utils.GenerateVersionCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	component := models.Component{
		AppID:        req.AppID,
		ComponentKey: req.ComponentKey,
		Name:         req.Name,
		Description:  req.Description,
		VersionCode:  versionCode,
		VersionName:  req.VersionName,
		Channel:      channel,
		Platform:     platform,
		Architecture: arch,
		FileSize:     req.FileSize,
		FileHash:     req.FileHash,
		DownloadURL:  req.DownloadURL,
		Status:       1,
	}

	if err := models.DB.Create(&component).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, component)
}

func UploadComponentFile(c *gin.Context) {
	appID := c.PostForm("app_id")
	componentKey := c.PostForm("component_key")
	name := c.PostForm("name")
	description := c.PostForm("description")
	versionName := c.PostForm("version_name")
	channel := c.PostForm("channel")
	platform := c.PostForm("platform")
	architecture := c.PostForm("architecture")
	downloadURL := c.PostForm("download_url")

	if appID == "" || componentKey == "" || name == "" || versionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, component_key, name and version_name are required"})
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

	if app.StorageType == "metadata" {
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

		adapter, err := storage.NewStorageAdapter(app.StorageType, app.StorageConfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to init storage: " + err.Error()})
			return
		}

		finalDownloadURL, fileSize, err = adapter.Upload(header.Filename, fileData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file: " + err.Error()})
			return
		}
	}

	if channel == "" {
		channel = models.ChannelStable
	}
	if platform == "" {
		platform = "all"
	}
	if architecture == "" {
		architecture = "all"
	}

	versionCode, err := utils.GenerateVersionCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	component := models.Component{
		AppID:        appID,
		ComponentKey: componentKey,
		Name:         name,
		Description:  description,
		VersionCode:  versionCode,
		VersionName:  versionName,
		Channel:      channel,
		Platform:     platform,
		Architecture: architecture,
		FileSize:     fileSize,
		FileHash:     fileHash,
		DownloadURL:  finalDownloadURL,
		Status:       1,
	}

	if err := models.DB.Create(&component).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, component)
}

func UpdateComponentStatus(c *gin.Context) {
	id := c.Param("id")

	var component models.Component
	if err := models.DB.First(&component, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
		return
	}

	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	component.Status = req.Status
	if err := models.DB.Save(&component).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, component)
}

func DeleteComponent(c *gin.Context) {
	id := c.Param("id")

	var component models.Component
	if err := models.DB.First(&component, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
		return
	}

	if err := models.DB.Delete(&component).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Component deleted successfully"})
}
