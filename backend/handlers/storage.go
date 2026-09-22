package handlers

import (
	"net/http"

	"iupdater/storage"

	"github.com/gin-gonic/gin"
)

// CheckStorageConnection 探测存储配置的连通性
// 请求体: { "storage_type": "s3|webdav|metadata", "storage_config": { ... } }
// 响应:   { "success": true, "message": "连通正常" }
//         { "success": false, "message": "无法连通 S3 endpoint (...)" }
func CheckStorageConnection(c *gin.Context) {
	var req struct {
		StorageType   string `json:"storage_type" binding:"required"`
		StorageConfig string `json:"storage_config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误: " + err.Error()})
		return
	}

	adapter, err := storage.NewStorageAdapter(req.StorageType, req.StorageConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "初始化存储失败: " + err.Error()})
		return
	}

	if err := adapter.CheckConnection(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "连通正常"})
}
