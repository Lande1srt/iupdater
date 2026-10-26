package handlers

import (
	"net/http"
	"strconv"

	"iupdater/models"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
)

type CreateAppRequest struct {
	AppID            string `json:"app_id" binding:"required"`
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	Icon             string `json:"icon"`
	StorageProfileID *uint  `json:"storage_profile_id"` // null = 元数据模式
	StoragePath      string `json:"storage_path"`
	EnableAuth       bool   `json:"enable_auth"`
}

type UpdateAppRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Icon             string `json:"icon"`
	StorageProfileID *uint  `json:"storage_profile_id"`
	StoragePath      string `json:"storage_path"`
	EnableAuth       bool   `json:"enable_auth"`
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
	query.Preload("StorageProfile").Offset(offset).Limit(pageSize).Order("id DESC").Find(&apps)

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
	if err := models.DB.Preload("StorageProfile").First(&app, id).Error; err != nil {
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

	// 检查冲突：包括 soft-deleted 的旧记录（SQLite uniqueIndex 会把 soft-deleted 行也算进去）
	var existing models.App
	if err := models.DB.Unscoped().Where("app_id = ?", req.AppID).First(&existing).Error; err == nil {
		// 如果已存在且未删除，直接报错
		if existing.DeletedAt.Time.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "App ID already exists"})
			return
		}
		// soft-deleted 旧记录：物理删干净再创建，避免 UNIQUE 冲突
		models.DB.Unscoped().Delete(&existing)
		// 同步清理关联表的软删残留（也物理删）
		models.DB.Unscoped().Where("app_id = ?", req.AppID).Delete(&models.AppVersion{})
		models.DB.Unscoped().Where("app_id = ?", req.AppID).Delete(&models.Component{})
	}

	// 校验 storage_profile_id 有效性
	if req.StorageProfileID != nil {
		var sp models.StorageProfile
		if err := models.DB.First(&sp, *req.StorageProfileID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Storage profile not found"})
			return
		}
	}

	var accessKey string
	if req.EnableAuth {
		accessKey = utils.GenerateAccessKey()
	}

	app := models.App{
		AppID:            req.AppID,
		Name:             req.Name,
		Description:      req.Description,
		Icon:             req.Icon,
		StorageProfileID: req.StorageProfileID,
		StoragePath:      req.StoragePath,
		EnableAuth:       req.EnableAuth,
		AccessKey:        accessKey,
	}

	if err := models.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	models.DB.Preload("StorageProfile").First(&app, app.ID)
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
	app.StoragePath = req.StoragePath

	// 支持设为 null（metadata 模式）
	if req.StorageProfileID != nil {
		var sp models.StorageProfile
		if err := models.DB.First(&sp, *req.StorageProfileID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Storage profile not found"})
			return
		}
		app.StorageProfileID = req.StorageProfileID
	}

	// 处理校验启用/禁用
	if req.EnableAuth && !app.EnableAuth {
		app.EnableAuth = true
		app.AccessKey = utils.GenerateAccessKey()
	} else if !req.EnableAuth && app.EnableAuth {
		app.EnableAuth = false
		app.AccessKey = ""
	}

	if err := models.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	models.DB.Preload("StorageProfile").First(&app, app.ID)
	c.JSON(http.StatusOK, app)
}

func DeleteApp(c *gin.Context) {
	id := c.Param("id")

	var app models.App
	if err := models.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	// 关联表先物理删除（带软删残留）
	models.DB.Unscoped().Where("app_id = ?", app.AppID).Delete(&models.AppVersion{})
	models.DB.Unscoped().Where("app_id = ?", app.AppID).Delete(&models.Component{})
	// Agent 也要物理删（避免 agent 表 app_id 约束冲突）
	models.DB.Unscoped().Where("app_id = ?", app.AppID).Delete(&models.Agent{})

	// App 本体物理删除（SQLite uniqueIndex 会把 soft-deleted 行也算进去，不能只软删）
	if err := models.DB.Unscoped().Delete(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App deleted successfully"})
}
