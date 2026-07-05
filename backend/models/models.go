package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	StorageTypeS3       = "s3"
	StorageTypeWebDAV   = "webdav"
	StorageTypeMetadata = "metadata"
)

const (
	ChannelStable = "stable"
	ChannelBeta   = "beta"
)

const (
	PackageTypeFull      = "full"
	PackageTypeComponent = "component"
)

type App struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppID         string         `gorm:"uniqueIndex;size:100;not null" json:"app_id"`
	Name          string         `gorm:"size:200;not null" json:"name"`
	Description   string         `gorm:"type:text" json:"description"`
	Icon          string         `gorm:"size:500" json:"icon"`
	StorageType   string         `gorm:"size:20;not null;default:'metadata'" json:"storage_type"`
	StorageConfig string         `gorm:"type:text" json:"storage_config"`
	EnableAuth    bool           `gorm:"default:false" json:"enable_auth"`
	AccessKey     string         `gorm:"size:16" json:"access_key"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type AppVersion struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppID         string         `gorm:"index;size:100;not null" json:"app_id"`
	VersionCode   int64          `gorm:"index;not null" json:"version_code"`
	VersionName   string         `gorm:"size:100;not null" json:"version_name"`
	Channel       string         `gorm:"index;size:20;not null;default:'stable'" json:"channel"`
	Platform      string         `gorm:"index;size:50;not null" json:"platform"`
	Architecture  string         `gorm:"index;size:50;not null;default:'all'" json:"architecture"`
	PackageType   string         `gorm:"size:20;not null;default:'full'" json:"package_type"`
	IsForceUpdate bool           `gorm:"default:false" json:"is_force_update"`
	FileSize      int64          `json:"file_size"`
	FileHash      string         `gorm:"size:128" json:"file_hash"`
	DownloadURL   string         `gorm:"size:1000" json:"download_url"`
	ReleaseNotes  string         `gorm:"type:text" json:"release_notes"`
	Status        int            `gorm:"default:1" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type Component struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	AppID        string         `gorm:"index;size:100;not null" json:"app_id"`
	ComponentKey string         `gorm:"size:100;not null" json:"component_key"`
	Name         string         `gorm:"size:200;not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	VersionCode  int64          `gorm:"not null" json:"version_code"`
	VersionName  string         `gorm:"size:100;not null" json:"version_name"`
	Channel      string         `gorm:"index;size:20;not null;default:'stable'" json:"channel"`
	Platform     string         `gorm:"index;size:50;not null;default:'all'" json:"platform"`
	Architecture string         `gorm:"index;size:50;not null;default:'all'" json:"architecture"`
	FileSize     int64          `json:"file_size"`
	FileHash     string         `gorm:"size:128" json:"file_hash"`
	DownloadURL  string         `gorm:"size:1000" json:"download_url"`
	Status       int            `gorm:"default:1" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type StorageConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AppID       string    `gorm:"uniqueIndex;size:100;not null" json:"app_id"`
	StorageType string    `gorm:"size:20;not null" json:"storage_type"`
	Config      string    `gorm:"type:text" json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
