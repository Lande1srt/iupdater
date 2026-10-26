package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"iupdater/models"

	"github.com/gin-gonic/gin"
)

// ============ 鉴权辅助 ============

func extractAgentAccessKey(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

// authenticateAgent 校验 access_key：优先查 Agent 记录，再兜底 App.AccessKey
func authenticateAgent(c *gin.Context) (*models.App, *models.Agent, bool) {
	key := extractAgentAccessKey(c)
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing access key"})
		return nil, nil, false
	}

	var agent models.Agent
	if err := models.DB.Where("access_key = ?", key).First(&agent).Error; err == nil {
		var app models.App
		if err := models.DB.Where("app_id = ?", agent.AppID).First(&app).Error; err == nil {
			return &app, &agent, true
		}
	}

	var app models.App
	if err := models.DB.Where("access_key = ? AND enable_auth = true", key).First(&app).Error; err == nil {
		return &app, nil, true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access key"})
	return nil, nil, false
}

// ============ Register ============

type AgentRegisterRequest struct {
	Name         string `json:"name"`
	AppID        string `json:"app_id" binding:"required"`
	Channel      string `json:"channel"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
}

// RegisterAgent 注册新的 agent，生成 access_key
func RegisterAgent(c *gin.Context) {
	var req AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var app models.App
	if err := models.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	accessKey := hex.EncodeToString(keyBytes)

	channel := req.Channel
	if channel == "" {
		channel = models.ChannelStable
	}

	agent := models.Agent{
		Name:         req.Name,
		AccessKey:    accessKey,
		AppID:        req.AppID,
		Channel:      channel,
		Platform:     req.Platform,
		Architecture: req.Architecture,
	}
	if agent.Name == "" {
		agent.Name = "agent-" + accessKey[:6]
	}

	if err := models.DB.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_key": accessKey,
		"agent_id":   agent.ID,
		"message":    "请保存好这个 access_key，后续 checkin 时需要",
	})
}

// ============ Update Agent (管理端：基础信息编辑) ============

type AgentUpdateRequest struct {
	Name         *string `json:"name"`
	Channel      *string `json:"channel"`
	Platform     *string `json:"platform"`
	Architecture *string `json:"architecture"`
	// 是否重置 access_key（可选，后端会生成新的）
	ResetAccessKey bool `json:"reset_access_key"`
}

func UpdateAgent(c *gin.Context) {
	id := c.Param("id")
	var agent models.Agent
	if err := models.DB.First(&agent, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	var req AgentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		agent.Name = *req.Name
	}
	if req.Channel != nil && *req.Channel != "" {
		agent.Channel = *req.Channel
	}
	if req.Platform != nil {
		agent.Platform = *req.Platform
	}
	if req.Architecture != nil {
		agent.Architecture = *req.Architecture
	}
	if req.ResetAccessKey {
		keyBytes := make([]byte, 16)
		rand.Read(keyBytes)
		agent.AccessKey = hex.EncodeToString(keyBytes)
	}

	if err := models.DB.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{"ok": true, "access_key": agent.AccessKey}
	if !req.ResetAccessKey {
		// 非重置场景不回显 access_key（前端已有）
		delete(resp, "access_key")
	}
	c.JSON(http.StatusOK, resp)
}

// ============ Agent Control (管理端：下发控制指令/远程配置) ============

// agentConfigPatchWhitelist 允许主控下发到 agent config.json 的字段白名单
// 敏感字段（access_key / server_url / app_id）永远不允许从主控覆盖
var agentConfigPatchWhitelist = map[string]struct{}{
	"poll_interval_seconds": {},
	"channel":               {},
	"platform":              {},
	"architecture":          {},
	"target_dir":            {},
	"backup_dir":            {},
	"download_dir":          {},
	"backup_keep":           {},
	"stop_command":          {},
	"start_command":         {},
	"stop_script":           {},
	"start_script":          {},
	"api_port":              {},
}

// filterAgentConfigPatch 把前端/请求里的 config_patch 按白名单过滤掉敏感/未知字段
func filterAgentConfigPatch(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		if _, ok := agentConfigPatchWhitelist[k]; ok {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type AgentControlRequest struct {
	// 二选一：要么 Command（控制），要么 ConfigPatch（远程配置覆盖）
	Command string `json:"command"` // check_now | restart | heartbeat | force_update
	// ForceVersionCode 可选：force_update 时指定要强制推送的版本 code；0 表示最新
	ForceVersionCode int64 `json:"force_version_code"`
	// ConfigPatch 是要合并进 agent 本地 config.json 的 JSON 覆盖
	// agent 收到后会 merge 到本地 config 并自动生效；敏感字段会被后端白名单过滤
	ConfigPatch map[string]any `json:"config_patch"`
}

func AgentControl(c *gin.Context) {
	id := c.Param("id")
	var agent models.Agent
	if err := models.DB.First(&agent, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	var req AgentControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 下发控制指令（仅当没有远程配置时）
	if req.Command != "" {
		switch req.Command {
		case "check_now", "restart", "heartbeat", "force_update":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "command must be one of: check_now, restart, heartbeat, force_update"})
			return
		}
		agent.PendingCommand = req.Command
		now := time.Now()
		agent.CommandAt = &now
		if req.Command == "force_update" {
			agent.ForceVersionCode = req.ForceVersionCode // 0 = 最新
		}
	}

	// 下发远程配置（可选，JSON 格式，agent 会 merge 到本地 config.json 的非空字段）
	// 注意：会先按白名单过滤敏感字段（access_key / server_url / app_id 等永远不下发）
	if patch := filterAgentConfigPatch(req.ConfigPatch); len(patch) > 0 {
		raw, _ := json.Marshal(patch)
		agent.RemoteConfig = string(raw)
	}

	if err := models.DB.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算下次 checkin 预估（agent 默认 poll_interval_seconds 分钟一次）
	nextHint := "下次 checkin 后生效"
	if agent.LastCheckinAt != nil {
		elapsed := time.Since(*agent.LastCheckinAt)
		if elapsed < 30*time.Second {
			nextHint = "agent 刚刚 checkin 过，预计 ~60s 内生效"
		} else if elapsed < 2*time.Minute {
			nextHint = "预计 ~1 分钟内生效"
		} else {
			nextHint = "agent 可能已离线（超过 5 分钟未 checkin）"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"message":      nextHint,
		"agent_status": agent.LastResult,
	})
}

// ============ Checkin ============

type AgentCheckinRequest struct {
	AppID          string `json:"app_id"`
	Channel        string `json:"channel"`
	Platform       string `json:"platform"`
	Architecture   string `json:"architecture"`
	CurrentVersion string `json:"current_version"`
	CurrentCode    int64  `json:"current_code"`
	// 可选：agent 把本地面板的完整状态快照一起上报
	Status map[string]any   `json:"status,omitempty"`
	Logs   []map[string]any `json:"logs,omitempty"`
	Extra  map[string]any   `json:"extra,omitempty"` // hostname / os 等扩展
}

func AgentCheckin(c *gin.Context) {
	app, agent, ok := authenticateAgent(c)
	if !ok {
		return
	}

	var req AgentCheckinRequest
	_ = c.ShouldBindJSON(&req)

	channel := req.Channel
	if channel == "" {
		if agent != nil && agent.Channel != "" {
			channel = agent.Channel
		} else {
			channel = models.ChannelStable
		}
	}
	platform := req.Platform
	if platform == "" {
		if agent != nil && agent.Platform != "" {
			platform = agent.Platform
		} else {
			platform = "all"
		}
	}
	arch := req.Architecture
	if arch == "" {
		if agent != nil && agent.Architecture != "" {
			arch = agent.Architecture
		} else {
			arch = "all"
		}
	}

	// ======== 更新 agent 的实时状态快照 ========
	if agent != nil {
		now := time.Now()
		agent.LastCheckinAt = &now

		if req.CurrentVersion != "" {
			agent.CurrentVersion = req.CurrentVersion
		}

		// 面板状态 JSON 序列化存库（限长避免过大）
		if req.Status != nil {
			raw, _ := json.Marshal(req.Status)
			if len(raw) > 8192 {
				raw = raw[:8192]
			}
			agent.StatusJSON = string(raw)

			// agent 自身状态报告为非失败 → 自动清除上一次的 fail 记录，让界面回到 idle
			if state := pickStatusState(req.Status); state != "fail" && agent.LastResult == "fail" {
				agent.LastResult = ""
				agent.LastMessage = ""
			}
		}

		// force_update 指令来了：跳过版本比较，不管 agent 什么版本都下发最新/指定版本
		forceUpdate := agent.PendingCommand == "force_update"
		targetCode := agent.ForceVersionCode

		// 先看有没有更新任务要下发（有就直接返回）
		hasUpdate := false
		latest, taskID, downloadURL, taskFound := findLatestVersion(app, channel, platform, arch, req, forceUpdate, targetCode)
		if taskFound {
			hasUpdate = true
		}

		var resp gin.H
		if hasUpdate {
			resp = buildTaskResponse(latest, taskID, downloadURL, c, *app)
			if forceUpdate {
				resp["force"] = true // 告诉 agent 忽略版本比较，强制覆盖
			}
			agent.LastTaskID = taskID
			agent.LastTaskAt = &now
			// force_update 指令已执行完（任务已下发），清空
			if forceUpdate {
				agent.PendingCommand = ""
				agent.CommandAt = nil
				agent.ForceVersionCode = 0
			}
		} else {
			resp = gin.H{}
			if forceUpdate {
				agent.PendingCommand = ""
				agent.CommandAt = nil
				agent.ForceVersionCode = 0
			}
		}

		// ======== 注入主控下发的控制指令 / 远程配置 ========
		// （如果有更新任务，也一起下发 — 让 agent 拿到更新的同时执行控制）
		if agent.PendingCommand != "" {
			resp["remote_command"] = agent.PendingCommand
			resp["command_at"] = agent.CommandAt
			agent.PendingCommand = "" // 领取后立即清空（agent 执行完会 report）
			agent.CommandAt = nil
		}
		if agent.RemoteConfig != "" {
			var patch map[string]any
			if err := json.Unmarshal([]byte(agent.RemoteConfig), &patch); err == nil {
				resp["config_patch"] = patch
				agent.RemoteConfig = "" // 领取后清空
			}
		}

		models.DB.Save(agent)
		c.JSON(http.StatusOK, resp)
		return
	}

	// 无 agent 记录（旧方式：直接用 App.AccessKey），只下发更新任务
	latest, taskID, downloadURL, taskFound := findLatestVersion(app, channel, platform, arch, req, false, 0)
	if !taskFound {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, buildTaskResponse(latest, taskID, downloadURL, c, *app))
}

// findLatestVersion 查找最新发布版本，返回是否要下发
// force=true 时忽略 current_code 比较，直接把最新/指定版本下发给 agent
func findLatestVersion(app *models.App, channel, platform, arch string, req AgentCheckinRequest, force bool, targetCode int64) (*models.AppVersion, string, string, bool) {
	query := models.DB.Where("app_id = ? AND channel = ? AND status = 1 AND package_type = ?",
		app.AppID, channel, models.PackageTypeFull).
		Where("platform IN ?", []string{platform, "all"}).
		Where("architecture IN ?", []string{arch, "all"})

	// 指定目标版本 code 时优先命中（没匹配到就回退最新）
	if targetCode > 0 {
		var v models.AppVersion
		if err := query.Where("version_code = ?", targetCode).First(&v).Error; err == nil {
			return &v, generateTaskID(), "", true
		}
	}

	var latest models.AppVersion
	if err := query.Order("version_code DESC").First(&latest).Error; err != nil {
		return nil, "", "", false
	}

	// force_update 跳过版本比较，不管 agent 当前版本是什么都下发
	if force {
		return &latest, generateTaskID(), "", true
	}

	var currentCode int64
	if req.CurrentCode > 0 {
		currentCode = req.CurrentCode
	} else if req.CurrentVersion != "" {
		var curVer models.AppVersion
		if models.DB.Where("app_id = ? AND version_name = ? AND status = 1",
			app.AppID, req.CurrentVersion).First(&curVer).Error == nil {
			currentCode = curVer.VersionCode
		}
	}

	if currentCode > 0 && latest.VersionCode <= currentCode {
		return nil, "", "", false
	}

	return &latest, generateTaskID(), "", true
}

func buildTaskResponse(latest *models.AppVersion, taskID, _ string, c *gin.Context, app models.App) gin.H {
	downloadURL := resolveDownloadURL(c, latest.DownloadURL, app)
	return gin.H{
		"task_id":       taskID,
		"version_name":  latest.VersionName,
		"version_code":  latest.VersionCode,
		"download_url":  downloadURL,
		"file_hash":     latest.FileHash,
		"release_notes": latest.ReleaseNotes,
		"file_size":     latest.FileSize,
	}
}

// ============ Report ============

type AgentReportRequest struct {
	TaskID     string `json:"task_id" binding:"required"`
	Status     string `json:"status" binding:"required"` // success | fail
	Message    string `json:"message"`
	NewVersion string `json:"new_version"`
}

func AgentReport(c *gin.Context) {
	_, agent, ok := authenticateAgent(c)
	if !ok {
		return
	}

	var req AgentReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != "success" && req.Status != "fail" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'success' or 'fail'"})
		return
	}

	if agent != nil {
		now := time.Now()
		agent.LastCheckinAt = &now
		agent.LastResult = req.Status
		agent.LastMessage = req.Message
		agent.CurrentVersion = req.NewVersion
		models.DB.Save(agent)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ============ 管理端：Agent 列表 ============

func GetAgents(c *gin.Context) {
	var agents []models.Agent
	if err := models.DB.Order("id DESC").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agents)
}

func GetAgent(c *gin.Context) {
	id := c.Param("id")
	var agent models.Agent
	if err := models.DB.First(&agent, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func DeleteAgent(c *gin.Context) {
	id := c.Param("id")
	if err := models.DB.Delete(&models.Agent{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func generateTaskID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// pickStatusState 从 agent checkin 上报的 status JSON 里提取 state 字段（用于自动清除 fail）
func pickStatusState(st map[string]any) string {
	if v, ok := st["state"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ============ 管理端：清除 agent 错误状态 ============

// ClearAgentError 手动清空 agent 的 last_result / last_message / status_json 中的 error，
// 适用于 agent 已恢复但主控界面错误没自动消掉的场景
func ClearAgentError(c *gin.Context) {
	id := c.Param("id")
	var agent models.Agent
	if err := models.DB.First(&agent, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	agent.LastResult = ""
	agent.LastMessage = ""

	// 如果 status_json 里也有 error 字段，一并清掉，避免前端重复展示
	if agent.StatusJSON != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(agent.StatusJSON), &m); err == nil {
			delete(m, "error")
			// 某些状态（"fail"/"success"）也应该回退到 idle，让界面显示当前真实状态
			if state, ok := m["state"].(string); ok && (state == "fail" || state == "success") {
				m["state"] = "idle"
			}
			if raw, err := json.Marshal(m); err == nil {
				agent.StatusJSON = string(raw)
			}
		}
	}

	if err := models.DB.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
