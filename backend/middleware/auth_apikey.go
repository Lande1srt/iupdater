package middleware

import (
	"net/http"
	"strings"
	"time"

	"iupdater/models"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
)

// ApiKeyMiddleware 是 /api/admin/** 的统一鉴权中间件。
// 按优先级尝试三种来源：
//
//  1. Header:  X-API-Key: <plain>
//  2. Header:  Authorization: Bearer APIKey-<plain>   (新 API Key 形态)
//  3. Header:  Authorization: Bearer <jwt>            (兼容旧 admin user JWT)
//
// 任一来源校验通过即放行。失败统一返回 401。
// 通过后 c.Set("admin_auth_type") = "api_key" | "jwt" 方便 handler 日志。
func ApiKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		xAPIKey := c.GetHeader("X-API-Key")

		if xAPIKey != "" {
			if tryAPIKey(c, xAPIKey) { c.Next(); return }
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or disabled X-API-Key"})
			c.Abort()
			return
		}
		if strings.HasPrefix(authHeader, "Bearer APIKey-") {
			key := strings.TrimPrefix(authHeader, "Bearer APIKey-")
			if tryAPIKey(c, key) { c.Next(); return }
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or disabled Bearer APIKey-..."})
			c.Abort()
			return
		}
		if strings.HasPrefix(authHeader, "Bearer ") {
			raw := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := utils.ParseToken(raw)
			if err == nil {
				c.Set("username", claims.Username)
				c.Set("admin_auth_type", "jwt")
				c.Next()
				return
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized. Provide X-API-Key header, or Authorization: Bearer APIKey-<key>, or legacy Bearer <admin-jwt>",
		})
		c.Abort()
	}
}

func tryAPIKey(c *gin.Context, plain string) bool {
	if plain == "" {
		return false
	}
	hash := models.Sha256Hex(plain)

	var key models.AdminAPIKey
	if err := models.DB.Where("key_hash = ? AND enabled = ?", hash, true).First(&key).Error; err != nil {
		return false
	}
	now := time.Now()
	_ = models.DB.Model(&key).Update("last_used_at", &now).Error

	c.Set("admin_api_key_id", key.ID)
	c.Set("admin_api_key_name", key.Name)
	c.Set("admin_permissions", key.Permissions)
	c.Set("admin_auth_type", "api_key")
	return true
}
