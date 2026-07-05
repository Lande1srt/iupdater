package handlers

import (
	"net/http"
	"strconv"

	"iupdater/models"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
)

type CreateAppRequest struct {
	AppID         string `json:"app_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Icon          string `json:"icon"`
	StorageType   string `json:"storage_type"`
	StorageConfig string `json:"storage_config"`
	EnableAuth    bool   `json:"enable_auth"`
}

type UpdateAppRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Icon          string `json:"icon"`
	StorageType   string `json:"storage_type"`
	StorageConfig string `json:"storage_config"`
	EnableAuth    bool   `json:"enable_auth"`
}

func GetApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	var apps []models.App
	var total int64

	query := models.DB.Model(&models.App{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR app_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&apps)

	c.JSON(http.StatusOK, gin.H{
		"list":      apps,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetApp(c *gin.Context) {
	id := c.Param("id")

	var app models.App
	if err := models.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	c.JSON(http.StatusOK, app)
}

func CreateApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.App
	if err := models.DB.Where("app_id = ?", req.AppID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID already exists"})
		return
	}

	storageType := req.StorageType
	if storageType == "" {
		storageType = models.StorageTypeMetadata
	}

	var accessKey string
	if req.EnableAuth {
		accessKey = utils.GenerateAccessKey()
	}

	app := models.App{
		AppID:         req.AppID,
		Name:          req.Name,
		Description:   req.Description,
		Icon:          req.Icon,
		StorageType:   storageType,
		StorageConfig: req.StorageConfig,
		EnableAuth:    req.EnableAuth,
		AccessKey:     accessKey,
	}

	if err := models.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

func UpdateApp(c *gin.Context) {
	id := c.Param("id")

	var app models.App
	if err := models.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	var req UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	app.Description = req.Description
	app.Icon = req.Icon
	if req.StorageType != "" {
		app.StorageType = req.StorageType
	}
	app.StorageConfig = req.StorageConfig

	// 处理校验启用/禁用
	if req.EnableAuth && !app.EnableAuth {
		// 从禁用改为启用，生成新的 AccessKey
		app.EnableAuth = true
		app.AccessKey = utils.GenerateAccessKey()
	} else if !req.EnableAuth && app.EnableAuth {
		// 从启用改为禁用，清空 AccessKey
		app.EnableAuth = false
		app.AccessKey = ""
	}

	if err := models.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

func DeleteApp(c *gin.Context) {
	id := c.Param("id")

	var app models.App
	if err := models.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	models.DB.Where("app_id = ?", app.AppID).Delete(&models.AppVersion{})
	models.DB.Where("app_id = ?", app.AppID).Delete(&models.Component{})

	if err := models.DB.Delete(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App deleted successfully"})
}
