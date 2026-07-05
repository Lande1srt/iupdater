package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type StorageAdapter interface {
	Upload(fileName string, data []byte) (string, int64, error)
	Delete(fileURL string) error
	GetURL(fileName string) string
}

type S3Config struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	BaseURL   string `json:"base_url"`
	UseProxy  bool   `json:"use_proxy"`
}

type WebDAVConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	BasePath string `json:"base_path"`
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
	key := fmt.Sprintf("%s/%s", uuid.New().String(), fileName)
	url := fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, key)

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("s3 upload failed: %s - %s", resp.Status, string(body))
	}

	// 返回存储key，GetURL方法会根据配置生成实际URL
	return key, int64(len(data)), nil
}

func (s *S3Storage) Delete(fileURL string) error {
	url := fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, fileURL)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *S3Storage) GetURL(fileName string) string {
	// 如果启用代理转发，返回系统代理URL
	if s.config.UseProxy {
		return fmt.Sprintf("/api/download/%s", fileName)
	}
	// 如果配置了外部CDN，使用CDN URL
	if s.config.BaseURL != "" {
		return fmt.Sprintf("%s/%s", s.config.BaseURL, fileName)
	}
	// 默认返回直连URL
	return fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, fileName)
}

type WebDAVStorage struct {
	config WebDAVConfig
}

func NewWebDAVStorage(config WebDAVConfig) *WebDAVStorage {
	return &WebDAVStorage{config: config}
}

func (w *WebDAVStorage) Upload(fileName string, data []byte) (string, int64, error) {
	key := fmt.Sprintf("%s/%s", uuid.New().String(), fileName)
	remotePath := filepath.Join(w.config.BasePath, key)
	url := strings.TrimRight(w.config.URL, "/") + "/" + strings.TrimLeft(remotePath, "/")

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return "", 0, err
	}
	req.SetBasicAuth(w.config.Username, w.config.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("webdav upload failed: %s - %s", resp.Status, string(body))
	}

	// 返回存储key，不是完整URL
	return key, int64(len(data)), nil
}

func (w *WebDAVStorage) Delete(fileURL string) error {
	req, err := http.NewRequest("DELETE", fileURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(w.config.Username, w.config.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
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
