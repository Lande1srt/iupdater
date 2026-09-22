package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// NewS3Client 用 aws-sdk-go-v2 创建已签名的 S3 客户端（支持 MinIO/OSS/COS 等 S3 兼容服务）
// 导出供 download handler 等其他模块复用
func NewS3Client(cfg S3Config) (*s3.Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint 未配置")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("bucket 未配置")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	usePathStyle := false
	if cfg.UsePathStyle != nil {
		usePathStyle = *cfg.UsePathStyle
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 AWS config 失败: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.HTTPClient = &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		o.UsePathStyle = usePathStyle
	}), nil
}

// s3ObjectKey 拼接 S3 object key 并规范化（去掉前导 /，S3 object key 不允许以 / 开头）
// base_path 不管填 "/coldsea/Appack" 还是 "coldsea/Appack"，key 都会变成 "coldsea/Appack/uuid/file"
func s3ObjectKey(basePath, key string) string {
	return strings.TrimLeft(path.Join(basePath, key), "/")
}

// HTTPClient 带统一超时的共享客户端，所有存储实现都用这个避免零超时卡死
var HTTPClient = &http.Client{
	Timeout: 5 * time.Minute, // 对象存储上传大文件可能较慢，给 5 分钟
}

// CheckHTTPClient 用于连通性探测的短超时 client，快速返回结果
var CheckHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

type StorageAdapter interface {
	Upload(fileName string, data []byte) (string, int64, error)
	Delete(fileURL string) error
	GetURL(fileName string) string
	CheckConnection() error
}

type S3Config struct {
	Endpoint     string `json:"endpoint"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	BaseURL      string `json:"base_url"`
	BasePath     string `json:"base_path"`
	DownloadMode string `json:"download_mode"`  // "proxy"(默认) | "redirect"
	UsePathStyle *bool  `json:"use_path_style"` // MinIO/Ceph 用 true；OSS/COS/七牛 用 false；默认 false
}

type WebDAVConfig struct {
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	BasePath     string `json:"base_path"`
	DownloadMode string `json:"download_mode"` // proxy | redirect, default redirect
}

type MetadataConfig struct {
	BaseURL string `json:"base_url"`
}

func NewStorageAdapter(storageType string, configJSON string) (StorageAdapter, error) {
	switch storageType {
	case "s3":
		var cfg S3Config
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("invalid s3 config: %w", err)
		}
		return NewS3Storage(cfg), nil
	case "webdav":
		var cfg WebDAVConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("invalid webdav config: %w", err)
		}
		return NewWebDAVStorage(cfg), nil
	case "metadata":
		var cfg MetadataConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("invalid metadata config: %w", err)
		}
		return NewMetadataStorage(cfg), nil
	default:
		return nil, errors.New("unknown storage type")
	}
}

type S3Storage struct {
	config S3Config
}

func NewS3Storage(config S3Config) *S3Storage {
	return &S3Storage{config: config}
}

func (s *S3Storage) Upload(fileName string, data []byte) (string, int64, error) {
	client, err := NewS3Client(s.config)
	if err != nil {
		return "", 0, err
	}
	key := fmt.Sprintf("%s/%s", uuid.New().String(), fileName)
	objectKey := s3ObjectKey(s.config.BasePath, key)
	log.Printf("[S3 Upload] bucket=%s base_path=%q key=%q -> objectKey=%q", s.config.Bucket, s.config.BasePath, key, objectKey)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", 0, fmt.Errorf("s3 upload failed: %w", err)
	}

	// 上传后立即 HeadObject 验证写入（防止 SDK 静默成功）
	_, err = client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("[S3 Upload] WARNING PutObject 成功但 HeadObject 验证失败: %v", err)
	} else {
		log.Printf("[S3 Upload] HeadObject 验证通过 ✓ key=%s", objectKey)
	}

	return key, int64(len(data)), nil
}

func (s *S3Storage) Delete(fileURL string) error {
	client, err := NewS3Client(s.config)
	if err != nil {
		return err
	}
	objectKey := s3ObjectKey(s.config.BasePath, fileURL)

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	})
	return err
}

func (s *S3Storage) GetURL(fileName string) string {
	return fmt.Sprintf("/api/download/%s", fileName)
}

// CheckConnection 用 aws-sdk-go-v2 HeadBucket 探测（会正确带 AWS 签名）
func (s *S3Storage) CheckConnection() error {
	client, err := NewS3Client(s.config)
	if err != nil {
		return err
	}
	_, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(s.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("无法连通 S3 (%s/%s): %w", s.config.Endpoint, s.config.Bucket, err)
	}
	return nil
}

type WebDAVStorage struct {
	config WebDAVConfig
}

func NewWebDAVStorage(config WebDAVConfig) *WebDAVStorage {
	return &WebDAVStorage{config: config}
}

func (w *WebDAVStorage) Upload(fileName string, data []byte) (string, int64, error) {
	key := fmt.Sprintf("%s/%s", uuid.New().String(), fileName)
	remotePath := path.Join(w.config.BasePath, key)
	url := strings.TrimRight(w.config.URL, "/") + "/" + strings.TrimLeft(remotePath, "/")

	// WebDAV 服务器不会自动创建中间目录，需要先 MKCOL 确保父目录存在
	dirPath := path.Join(w.config.BasePath, path.Dir(key))
	dirURL := strings.TrimRight(w.config.URL, "/") + "/" + strings.TrimLeft(dirPath, "/")

	mkcolReq, _ := http.NewRequest("MKCOL", dirURL, nil)
	mkcolReq.SetBasicAuth(w.config.Username, w.config.Password)
	mkcolResp, mkcolErr := HTTPClient.Do(mkcolReq)
	if mkcolErr == nil {
		mkcolResp.Body.Close()
	}
	// MKCOL 可能返回 405/409（目录已存在），忽略即可

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return "", 0, err
	}
	req.SetBasicAuth(w.config.Username, w.config.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("webdav upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("webdav upload failed: %s - %s", resp.Status, string(body))
	}

	return key, int64(len(data)), nil
}

func (w *WebDAVStorage) Delete(fileURL string) error {
	fullURL := strings.TrimRight(w.config.URL, "/") + "/" + strings.TrimLeft(path.Join(w.config.BasePath, fileURL), "/")

	req, err := http.NewRequest("DELETE", fullURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(w.config.Username, w.config.Password)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (w *WebDAVStorage) GetURL(fileName string) string {
	// WebDAV统一使用系统代理转发
	return fmt.Sprintf("/api/download/%s", fileName)
}

// CheckConnection 探测 WebDAV URL 可达性（PROPFIND base_path）
func (w *WebDAVStorage) CheckConnection() error {
	if w.config.URL == "" {
		return errors.New("webdav URL 未配置")
	}
	url := strings.TrimRight(w.config.URL, "/") + "/" + strings.TrimLeft(w.config.BasePath, "/")

	req, err := http.NewRequest("PROPFIND", url, strings.NewReader(`<?xml version="1.0" encoding="utf-8"?><D:propfind xmlns:D="DAV:"><D:prop><D:displayname/></D:prop></D:propfind>`))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.SetBasicAuth(w.config.Username, w.config.Password)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "0")

	resp, err := CheckHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法连通 WebDAV (%s): %w (请检查 URL、端口和网络)", w.config.URL, err)
	}
	defer resp.Body.Close()

	// 收到 HTTP 响应 = 网络通，不管状态码
	switch resp.StatusCode {
	case http.StatusMultiStatus, http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil // 网络通的，只是用户名密码错
	default:
		return nil // 任何其他 HTTP 状态码也说明网络通
	}
}

type MetadataStorage struct {
	config MetadataConfig
}

func NewMetadataStorage(config MetadataConfig) *MetadataStorage {
	return &MetadataStorage{config: config}
}

func (m *MetadataStorage) Upload(fileName string, data []byte) (string, int64, error) {
	uploadDir := "data/uploads"
	os.MkdirAll(uploadDir, 0755)

	uuidDir := filepath.Join(uploadDir, uuid.New().String())
	os.MkdirAll(uuidDir, 0755)

	fullPath := filepath.Join(uuidDir, fileName)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", 0, err
	}

	relPath := filepath.ToSlash(filepath.Join(uuidDir, fileName))
	return relPath, int64(len(data)), nil
}

func (m *MetadataStorage) Delete(fileURL string) error {
	if _, err := os.Stat(fileURL); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(fileURL)
}

func (m *MetadataStorage) GetURL(fileName string) string {
	if m.config.BaseURL != "" {
		return fmt.Sprintf("%s/%s", m.config.BaseURL, fileName)
	}
	return fileName
}

// CheckConnection Metadata 模式不做上传，始终返回成功
func (m *MetadataStorage) CheckConnection() error {
	return nil
}
