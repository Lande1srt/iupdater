package models

import (
	"iupdater/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(&App{}, &AppVersion{}, &Component{}, &StorageConfig{})
	if err != nil {
		return err
	}

	return nil
}
