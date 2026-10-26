package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UpdateTask 主控后端下发的更新任务
type UpdateTask struct {
	TaskID       string `json:"task_id"`
	VersionName  string `json:"version_name"`
	VersionCode  int64  `json:"version_code"`
	DownloadURL  string `json:"download_url"`
	FileHash     string `json:"file_hash"`
	ReleaseNotes string `json:"release_notes"`
	// Force=true 时忽略本地版本比较，主控命令强制推送
	Force bool `json:"force"`
}

// ControlAction 主控下发的远程控制指令 / 配置覆盖
type ControlAction struct {
	// Command: "check_now" | "restart" | "heartbeat" | ""
	Command   string     `json:"remote_command"`
	CommandAt *time.Time `json:"command_at"`
	// ConfigPatch 合并进 agent 本地 config.json 的 JSON 覆盖（只覆盖非空字段）
	ConfigPatch map[string]any `json:"config_patch"`
}

// Client 与主控后端通信的 HTTP 客户端
type Client struct {
	http    *http.Client
	baseURL string
	key     string
}

func NewClient(cfg *Config) *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: cfg.ServerURL,
		key:     cfg.AccessKey,
	}
}

// Checkin 上报当前状态 + 面板快照，拉取更新任务和控制指令
// 返回 (task, control) —— task 为 nil 表示无更新；control 含主控下发的 command / config_patch
func (c *Client) Checkin(cfg *Config, currentVersion string, currentCode int64, statusSnapshot map[string]any) (*UpdateTask, *ControlAction) {
	body := map[string]any{
		"app_id":          cfg.AppID,
		"channel":         cfg.Channel,
		"platform":        cfg.Platform,
		"architecture":    cfg.Architecture,
		"current_version": currentVersion,
		"current_code":    currentCode,
		"status":          statusSnapshot, // agent 面板实时快照，主控存库
	}

	data, err := json.Marshal(body)
	if err != nil {
		DefaultLogger.Error("checkin: 序列化失败: %v", err)
		return nil, nil
	}

	url := joinURL(c.baseURL, "/api/agent/checkin")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		DefaultLogger.Error("checkin: 构造请求失败: %v", err)
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		DefaultLogger.Warn("checkin: 请求失败: %v", err)
		return nil, nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		DefaultLogger.Warn("checkin: 后端返回 %d: %s", resp.StatusCode, string(raw))
		return nil, nil
	}

	// 解析响应 —— 可能有 task_id（更新）和 remote_command/config_patch（控制）
	var rawResp map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawResp); err != nil {
		return nil, nil
	}

	// 提取控制字段
	control := &ControlAction{}
	if v, ok := rawResp["remote_command"]; ok {
		json.Unmarshal(v, &control.Command)
	}
	if v, ok := rawResp["config_patch"]; ok {
		json.Unmarshal(v, &control.ConfigPatch)
	}
	if control.Command == "" && len(control.ConfigPatch) == 0 {
		control = nil
	}

	// 提取任务
	if _, ok := rawResp["task_id"]; ok {
		var task UpdateTask
		_ = json.Unmarshal(raw, &task)
		if task.TaskID != "" {
			return &task, control
		}
	}
	return nil, control
}

// ReportResult 上报更新结果
func (c *Client) ReportResult(task *UpdateTask, status, message string) {
	body := map[string]any{
		"task_id":     task.TaskID,
		"status":      status,
		"message":     message,
		"new_version": task.VersionName,
	}

	data, err := json.Marshal(body)
	if err != nil {
		DefaultLogger.Error("report: 序列化失败: %v", err)
		return
	}

	url := joinURL(c.baseURL, "/api/agent/report")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		DefaultLogger.Error("report: 构造请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		DefaultLogger.Warn("report: 请求失败: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		DefaultLogger.Warn("report: 后端返回 %d", resp.StatusCode)
	}
}

// Download 下载文件到目标路径，支持进度回调
func (c *Client) Download(urlStr, destPath string, onProgress func(downloaded, total int64)) error {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return fmt.Errorf("构造下载请求失败: %w", err)
	}
	c.setAuth(req)

	cli := &http.Client{Timeout: 0}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}

	out, err := createFile(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, werr := out.Write(chunk); werr != nil {
				return fmt.Errorf("写入文件失败: %w", err)
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取下载流失败: %w", err)
		}
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
}

func joinURL(base, path string) string {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if base[len(base)-1] != '/' {
		base += "/"
	}
	return base + path
}
