package models

import (
	"time"

	"gorm.io/gorm"
)

// AdminAPIKey 是独立于 Admin User (JWT) 的长期 API 密钥，
// 用于 SeaStack / CI / 脚本等非交互式客户端调用 /api/admin/**。
//
// 设计要点：
//   - 明文 key 永不入库，入库的是 sha256 哈希值（key_hash）
//   - 明文只在创建 / 重置时返回一次，之后再也拿不到
//   - permissions 是 JSON 字符串，存 scope 列表，当前 scope 定义：
//       * "*"               全部（等同 super-admin）
//       * "apps:read" / "apps:write"
//       * "versions:read" / "versions:write"
//       * "components:read" / "components:write"
//       * "storage:read" / "storage:write"
//       * "agents:read" / "agents:write"
//   - enabled=false 可以临时禁用，不用删除
//   - last_used_at 记录最近一次使用时间（诊断谁在用）
type AdminAPIKey struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`       // 给人看的标签，例如 "SeaStack基座"
	KeyHash     string         `gorm:"size:64;uniqueIndex;not null" json:"-"` // sha256(key)
	Permissions string         `gorm:"size:1000;not null;default:'*'" json:"permissions"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	LastUsedAt  *time.Time     `json:"last_used_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
