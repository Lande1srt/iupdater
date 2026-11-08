package handlers

import (
	"net/http"
	"strconv"

	"iupdater/models"

	"github.com/gin-gonic/gin"
)

// ==================== Admin APIKey 管理 ====================
// 注意：明文 key 只会在 Create / Regenerate 的响应里返回一次，
// 数据库只存 SHA256 哈希值，之后再也拿不到明文。

type createAPIKeyReq struct {
	Name        string `json:"name" binding:"required,max=100"`
	Permissions string `json:"permissions"` // 默认 "*"
}

type updateAPIKeyReq struct {
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
}

// ListAPIKeys GET /api/admin/api-keys
func ListAPIKeys(c *gin.Context) {
	var list []models.AdminAPIKey
	if err := models.DB.Order("id DESC").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": decorateKeys(list)})
}

// CreateAPIKey POST /api/admin/api-keys — 唯一会在响应里返回明文的接口
func CreateAPIKey(c *gin.Context) {
	var req createAPIKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plain, err := models.GenerateAdminAPIKeyPlain()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}
	perms := req.Permissions
	if perms == "" { perms = "*" }
	key := models.AdminAPIKey{
		Name:        req.Name,
		KeyHash:     models.Sha256Hex(plain),
		Permissions: perms,
		Enabled:     true,
	}
	if err := models.DB.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := decorateKeys([]models.AdminAPIKey{key})[0]
	resp["plain"] = plain // 唯一一次返回明文
	c.JSON(http.StatusOK, gin.H{"key": resp})
}

// UpdateAPIKey PUT /api/admin/api-keys/:id — 只改 name / permissions
func UpdateAPIKey(c *gin.Context) {
	id, ok := parseAPIKeyID(c)
	if !ok { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	var req updateAPIKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var key models.AdminAPIKey
	if err := models.DB.First(&key, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return
	}
	if req.Name != ""        { key.Name = req.Name }
	if req.Permissions != "" { key.Permissions = req.Permissions }
	if err := models.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"key": decorateKeys([]models.AdminAPIKey{key})[0]})
}

// DeleteAPIKey DELETE /api/admin/api-keys/:id
func DeleteAPIKey(c *gin.Context) {
	id, ok := parseAPIKeyID(c)
	if !ok { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	if err := models.DB.Delete(&models.AdminAPIKey{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RegenerateAPIKey POST /api/admin/api-keys/:id/regenerate — 换一把新 key（返回明文一次）
func RegenerateAPIKey(c *gin.Context) {
	id, ok := parseAPIKeyID(c)
	if !ok { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	var key models.AdminAPIKey
	if err := models.DB.First(&key, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return
	}
	plain, err := models.GenerateAdminAPIKeyPlain()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"}); return
	}
	key.KeyHash = models.Sha256Hex(plain)
	if err := models.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	resp := decorateKeys([]models.AdminAPIKey{key})[0]
	resp["plain"] = plain
	c.JSON(http.StatusOK, gin.H{"key": resp})
}

// ToggleAPIKey POST /api/admin/api-keys/:id/toggle — 临时禁用 / 启用
func ToggleAPIKey(c *gin.Context) {
	id, ok := parseAPIKeyID(c)
	if !ok { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
	var key models.AdminAPIKey
	if err := models.DB.First(&key, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return
	}
	key.Enabled = !key.Enabled
	if err := models.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"key": decorateKeys([]models.AdminAPIKey{key})[0]})
}

// ==================== helpers ====================

func parseAPIKeyID(c *gin.Context) (uint, bool) {
	v, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil { return 0, false }
	return uint(v), true
}

func decorateKeys(list []models.AdminAPIKey) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, k := range list {
		out = append(out, map[string]any{
			"id":           k.ID,
			"name":         k.Name,
			"enabled":      k.Enabled,
			"permissions":  k.Permissions,
			"has_key":      k.KeyHash != "",
			"last_used_at": k.LastUsedAt,
			"created_at":   k.CreatedAt,
			"updated_at":   k.UpdatedAt,
		})
	}
	return out
}
