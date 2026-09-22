package handlers

import (
	"net/http"

	"iupdater/config"
	"iupdater/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"` // 可选：启用 TOTP 时必填
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 用户名密码校验（第 1 步，始终执行）
	if req.Username != config.AppConfig.AdminUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	if req.Password != config.AppConfig.AdminPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	totpEnabled := config.AppConfig.TOTPSecret != ""

	// 启用了 TOTP 但前端还没提交 totp_code → 通知前端进入第 2 步
	if totpEnabled && req.TOTPCode == "" {
		c.JSON(http.StatusOK, gin.H{
			"step":          "totp",
			"totp_required": true,
			"totp_enabled":  true,
		})
		return
	}

	// TOTP 校验（第 2 步，用户已提交 totp_code）
	if totpEnabled {
		if !utils.ValidateTOTP(config.AppConfig.TOTPSecret, req.TOTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":         "Invalid TOTP code",
				"totp_required": true,
			})
			return
		}
	}

	token, err := utils.GenerateToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"username": req.Username,
		},
		"totp_enabled": totpEnabled,
	})
}

// GetLoginConfig 公开 endpoint，前端登录页调用判断是否需要显示 TOTP 输入框
func GetLoginConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"totp_enabled": config.AppConfig.TOTPSecret != "",
	})
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
