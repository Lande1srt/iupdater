package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"iupdater/models"
	"iupdater/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func ProxyDownload(c *gin.Context) {
	fileKey := c.Param("filekey")
	if fileKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file key is required"})
		return
	}

	fileKey = strings.TrimLeft(fileKey, "/")

	var version models.AppVersion
	var component models.Component
	var app models.App

	patterns := []string{
		"/api/download/" + fileKey,
		fileKey,
	}
	found := false

	for _, pattern := range patterns {
		if err := models.DB.Where("download_url LIKE ?", "%"+pattern+"%").First(&version).Error; err == nil {
			if err := models.DB.Where("app_id = ?", version.AppID).First(&app).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
				return
			}
			found = true
			break
		}
		if err := models.DB.Where("download_url LIKE ?", "%"+pattern+"%").First(&component).Error; err == nil {
			if err := models.DB.Where("app_id = ?", component.AppID).First(&app).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
				return
			}
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	storageType, storageConfig, err := ResolveAppStorage(&app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage profile not found"})
		return
	}

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
	objectKey := strings.TrimLeft(path.Join(cfg.BasePath, fileKey), "/")
	log.Printf("[S3 Download] bucket=%s base_path=%q fileKey=%q -> objectKey=%q", cfg.Bucket, cfg.BasePath, fileKey, objectKey)

	// 优先：如果配了 BaseURL（CDN 加速域名），直接 302 跳 CDN，无需预签名
	if cfg.BaseURL != "" {
		cdnURL := strings.TrimRight(cfg.BaseURL, "/") + "/" + objectKey
		log.Printf("[S3 Download] redirect to CDN: %s", cdnURL)
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Redirect(http.StatusFound, cdnURL)
		return
	}

	client, err := storage.NewS3Client(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to init s3 client: " + err.Error()})
		return
	}
	ctx := context.TODO()

	// 重定向模式：生成预签名 URL（bucket 即使是 private 也能临时访问）
	if cfg.DownloadMode == "redirect" {
		presignClient := s3.NewPresignClient(client)
		req, err := presignClient.PresignGetObject(ctx,
			&s3.GetObjectInput{
				Bucket: aws.String(cfg.Bucket),
				Key:    aws.String(objectKey),
			},
			func(o *s3.PresignOptions) {
				o.Expires = 15 * time.Minute
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to presign s3 url: " + err.Error()})
			return
		}
		log.Printf("[S3 Download] presigned redirect: %s", req.URL)
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Redirect(http.StatusFound, req.URL)
		return
	}

	// 代理模式（默认）：后端 GetObject + io.Copy
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download from s3: " + err.Error()})
		return
	}
	defer out.Body.Close()

	c.Header("Content-Type", aws.ToString(out.ContentType))
	if out.ContentLength != nil {
		c.Header("Content-Length", fmt.Sprintf("%d", *out.ContentLength))
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileKey)))

	io.Copy(c.Writer, out.Body)
}

func proxyWebDAVDownload(c *gin.Context, cfg storage.WebDAVConfig, fileKey string) {
	remotePath := path.Join(cfg.BasePath, fileKey)
	url := strings.TrimRight(cfg.URL, "/") + "/" + strings.TrimLeft(remotePath, "/")

	// 重定向模式（WebDAV 默认）
	if cfg.DownloadMode != "proxy" {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Redirect(http.StatusFound, url)
		return
	}

	// 代理模式
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)

	resp, err := storage.HTTPClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download from webdav"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webdav returned error"})
		return
	}

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Length", resp.Header.Get("Content-Length"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileKey)))

	io.Copy(c.Writer, resp.Body)
}
