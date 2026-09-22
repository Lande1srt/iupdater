package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"iupdater/models"
	"iupdater/storage"

	"github.com/gin-gonic/gin"
)

type CreateStorageProfileRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	StorageType string `json:"storage_type" binding:"required"`
	Config      string `json:"config" binding:"required"`
}

type UpdateStorageProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StorageType string `json:"storage_type"`
	Config      string `json:"config"`
}

func GetStorageProfiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	var profiles []models.StorageProfile
	var total int64

	query := models.DB.Model(&models.StorageProfile{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	if pageSize > 100 {
		pageSize = 100
	}
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&profiles)

	c.JSON(http.StatusOK, gin.H{
		"list":      profiles,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetStorageProfile(c *gin.Context) {
	id := c.Param("id")
	var profile models.StorageProfile
	if err := models.DB.First(&profile, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Storage profile not found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func CreateStorageProfile(c *gin.Context) {
	var req CreateStorageProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 校验 Config JSON 合法性和 storage_type 有效性
	if req.StorageType != models.StorageTypeS3 && req.StorageType != models.StorageTypeWebDAV && req.StorageType != models.StorageTypeMetadata {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage_type"})
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(req.Config), &raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config must be valid JSON"})
		return
	}

	profile := models.StorageProfile{
		Name:        req.Name,
		Description: req.Description,
		StorageType: req.StorageType,
		Config:      req.Config,
	}

	if err := models.DB.Create(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func UpdateStorageProfile(c *gin.Context) {
	id := c.Param("id")
	var profile models.StorageProfile
	if err := models.DB.First(&profile, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Storage profile not found"})
		return
	}

	var req UpdateStorageProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		profile.Name = req.Name
	}
	profile.Description = req.Description
	if req.StorageType != "" {
		profile.StorageType = req.StorageType
	}
	if req.Config != "" {
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(req.Config), &raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "config must be valid JSON"})
			return
		}
		profile.Config = req.Config
	}

	if err := models.DB.Save(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func DeleteStorageProfile(c *gin.Context) {
	id := c.Param("id")

	var profile models.StorageProfile
	if err := models.DB.First(&profile, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Storage profile not found"})
		return
	}

	if err := models.DB.Delete(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Storage profile deleted"})
}

// CheckStorageProfileConnection 测试档案连通性
func CheckStorageProfileConnection(c *gin.Context) {
	id := c.Param("id")

	var profile models.StorageProfile
	if err := models.DB.First(&profile, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Storage profile not found"})
		return
	}

	adapter, err := storage.NewStorageAdapter(profile.StorageType, profile.Config)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := adapter.CheckConnection(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "连通正常"})
}
