package models

import (
	"iupdater/config"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	dbDir := filepath.Dir(config.AppConfig.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(&App{}, &AppVersion{}, &Component{}, &StorageConfig{}, &StorageProfile{}, &Agent{})
	if err != nil {
		return err
	}

	return nil
}
