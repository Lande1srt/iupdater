package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"iupdater/models"
	"iupdater/storage"

	"github.com/gin-gonic/gin"
)

func ProxyDownload(c *gin.Context) {
	fileKey := c.Param("filekey")
	if fileKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file key is required"})
		return
	}

	// 从数据库查找包含此文件key的版本或组件
	var version models.AppVersion
	var component models.Component
	var app models.App
	var storageType, storageConfig string

	// 尝试从AppVersion表查找
	if err := models.DB.Where("download_url LIKE ?", "%"+fileKey+"%").First(&version).Error; err == nil {
		if err := models.DB.Where("app_id = ?", version.AppID).First(&app).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		storageType = app.StorageType
		storageConfig = app.StorageConfig
	} else {
		// 尝试从Component表查找
		if err := models.DB.Where("download_url LIKE ?", "%"+fileKey+"%").First(&component).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		if err := models.DB.Where("app_id = ?", component.AppID).First(&app).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		storageType = app.StorageType
		storageConfig = app.StorageConfig
	}

	// 根据存储类型获取实际文件
	switch storageType {
	case "s3":
		var cfg storage.S3Config
		if err := json.Unmarshal([]byte(storageConfig), &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid s3 config"})
			return
		}
		proxyS3Download(c, cfg, fileKey)

	case "webdav":
		var cfg storage.WebDAVConfig
		if err := json.Unmarshal([]byte(storageConfig), &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid webdav config"})
			return
		}
		proxyWebDAVDownload(c, cfg, fileKey)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported storage type for proxy"})
	}
}

func proxyS3Download(c *gin.Context, cfg storage.S3Config, fileKey string) {
	url := fmt.Sprintf("%s/%s/%s", cfg.Endpoint, cfg.Bucket, fileKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download from s3"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "s3 returned error"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Length", resp.Header.Get("Content-Length"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileKey)))

	// 流式转发
	io.Copy(c.Writer, resp.Body)
}

func proxyWebDAVDownload(c *gin.Context, cfg storage.WebDAVConfig, fileKey string) {
	remotePath := filepath.Join(cfg.BasePath, fileKey)
	url := strings.TrimRight(cfg.URL, "/") + "/" + strings.TrimLeft(remotePath, "/")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download from webdav"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webdav returned error"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Length", resp.Header.Get("Content-Length"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileKey)))

	// 流式转发
	io.Copy(c.Writer, resp.Body)
}
