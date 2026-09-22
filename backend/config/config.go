package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       int
	AdminUser  string
	AdminPass  string
	JWTSecret  string
	DBPath     string
	TOTPSecret string // 可选 TOTP 秘钥，空则不启用两步验证
}

var AppConfig *Config

func LoadConfig() error {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("PORT", "8080"))

	AppConfig = &Config{
		Port:       port,
		AdminUser:  getEnv("ADMIN_USER", "admin"),
		AdminPass:  getEnv("ADMIN_PASS", "admin123"),
		JWTSecret:  getEnv("JWT_SECRET", "iupdater-secret-key-change-in-production"),
		DBPath:     getEnv("DB_PATH", "data/iupdater.db"),
		TOTPSecret: getEnv("TOTP_SECRET", ""), // 空 = 不启用两步验证
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
