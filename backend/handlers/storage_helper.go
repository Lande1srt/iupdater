package handlers

import (
	"encoding/json"
	"path"
	"strings"

	"iupdater/models"
	"iupdater/storage"
)

// ResolveAppStorage 从 App 的 StorageProfileID + StoragePath 解出完整的存储配置
// App 没关联 StorageProfile 时视为 metadata 模式（空配置）
// 返回合并了 App.StoragePath 覆盖后的 StorageConfig JSON
func ResolveAppStorage(app *models.App) (storageType string, storageConfig string, err error) {
	if app.StorageProfileID == nil {
		return models.StorageTypeMetadata, "{}", nil
	}

	var profile models.StorageProfile
	if err := models.DB.First(&profile, *app.StorageProfileID).Error; err != nil {
		return "", "", err
	}

	storageType = profile.StorageType
	storageConfig = profile.Config

	// 如果 App 有自定义 StoragePath，覆盖档案的 base_path
	if app.StoragePath != "" {
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(storageConfig), &raw); err == nil {
			raw["base_path"] = app.StoragePath
			if b, e := json.Marshal(raw); e == nil {
				storageConfig = string(b)
			}
		}
	}

	return storageType, storageConfig, nil
}

// MergeBasePath 把档案 base_path + App 自定义 storage_path + fileKey 合成最终 object key
func MergeBasePath(profileBasePath, appStoragePath, fileKey string) string {
	basePath := profileBasePath
	if appStoragePath != "" {
		basePath = appStoragePath
	}
	return strings.TrimLeft(path.Join(basePath, fileKey), "/")
}

// NewAdapterFromApp 从 App 直接构造 storage adapter
func NewAdapterFromApp(app *models.App) (storage.StorageAdapter, error) {
	st, cfg, err := ResolveAppStorage(app)
	if err != nil {
		return nil, err
	}
	return storage.NewStorageAdapter(st, cfg)
}
