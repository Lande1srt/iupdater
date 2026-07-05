package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     int
	AdminUser string
	AdminPass string
	JWTSecret string
	DBPath    string
}

var AppConfig *Config

func LoadConfig() error {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("PORT", "8080"))

	AppConfig = &Config{
		Port:      port,
		AdminUser: getEnv("ADMIN_USER", "admin"),
		AdminPass: getEnv("ADMIN_PASS", "admin123"),
		JWTSecret: getEnv("JWT_SECRET", "iupdater-secret-key-change-in-production"),
		DBPath:    getEnv("DB_PATH", "data/iupdater.db"),
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
