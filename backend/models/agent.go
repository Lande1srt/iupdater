package models

import (
	"time"

	"gorm.io/gorm"
)

// Agent 记录辅助更新助手
type Agent struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"size:200" json:"name"`
	AccessKey    string `gorm:"index;size:32;not null" json:"access_key"`
	AppID        string `gorm:"index;size:100;not null" json:"app_id"`
	Channel      string `gorm:"size:20;default:'stable'" json:"channel"`
	Platform     string `gorm:"size:50" json:"platform"`
	Architecture string `gorm:"size:50" json:"architecture"`

	// 运行时状态（agent 通过 checkin 上报，存完整 JSON 快照，面板直接显示）
	StatusJSON string `gorm:"type:text" json:"status_json,omitempty"`

	// 当前版本
	CurrentVersion string `gorm:"size:100" json:"current_version"`

	// 主控下发的待执行指令（agent 下次 checkin 时领取执行，执行后自动置空）
	// 取值: "check_now" | "restart" | "heartbeat" | "force_update" | ""
	PendingCommand string     `gorm:"size:30;default:''" json:"pending_command,omitempty"`
	CommandAt      *time.Time `json:"command_at,omitempty"`

	// 强制推送：当 PendingCommand=="force_update" 时，主控会忽略当前版本比较，
	// 把最新版本直接下发给 agent 执行更新。这个字段记录目标版本 code（可选）。
	ForceVersionCode int64 `gorm:"default:0" json:"force_version_code,omitempty"`

	// 主控下发的远程配置覆盖（agent 合并写入本地 config.json 后自动置空）
	// JSON 对象，字段与 agent config 一致，只覆盖非空字段
	RemoteConfig string `gorm:"type:text" json:"remote_config,omitempty"`

	LastCheckinAt *time.Time     `json:"last_checkin_at"`
	LastResult    string         `gorm:"size:20" json:"last_result"` // success / fail / empty
	LastMessage   string         `gorm:"type:text" json:"last_message"`
	LastTaskID    string         `gorm:"size:50" json:"last_task_id"`
	LastTaskAt    *time.Time     `json:"last_task_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
