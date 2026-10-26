package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// versionMetaFile 记录 agent 最后一次成功更新的版本元数据
// 放在 agent 自身 BaseDir（与 config.json 同目录），点号前缀避免用户误删
const versionMetaFile = ".iupdater-version.json"

// VersionMeta 落盘的版本元数据
type VersionMeta struct {
	AppID       string    `json:"app_id"`
	VersionName string    `json:"version_name"`
	VersionCode int64     `json:"version_code"`
	UpdatedAt   time.Time `json:"updated_at"`
	TaskID      string    `json:"task_id"`
}

// AppStatus 对外暴露的当前状态（本地面板 + 上报主控）
type AppStatus struct {
	State          string    `json:"state"`
	CurrentVersion string    `json:"current_version"`
	CurrentCode    int64     `json:"current_code"`
	LastCheckAt    time.Time `json:"last_check_at"`
	LastUpdateAt   time.Time `json:"last_update_at"`
	Progress       int       `json:"progress"`
	TaskID         string    `json:"task_id"`
	Error          string    `json:"error,omitempty"`
}

// Agent 聚合所有运行时依赖
type Agent struct {
	cfg     *Config
	client  *Client
	updater *Updater

	mu       sync.Mutex
	status   AppStatus
	shutdown chan struct{}
}

func NewAgent(cfg *Config) *Agent {
	a := &Agent{
		cfg:      cfg,
		client:   NewClient(cfg),
		shutdown: make(chan struct{}),
	}
	// updater 持有一个 onProgress 回调，用于把下载进度推给 agent 面板
	a.updater = NewUpdater(cfg, NewClient(cfg), func(p int) { a.setProgress(p) })
	// 从本地元数据文件恢复版本信息（重启不丢）
	if meta, err := loadVersionMeta(cfg); err == nil && meta != nil {
		a.status.CurrentVersion = meta.VersionName
		a.status.CurrentCode = meta.VersionCode
		a.status.LastUpdateAt = meta.UpdatedAt
		DefaultLogger.Info("从 %s 恢复版本 %s (code=%d)", versionMetaFile, meta.VersionName, meta.VersionCode)
	}
	return a
}

// loadVersionMeta 读 agent BaseDir 下的版本元数据文件，没找到返回 nil, nil
func loadVersionMeta(cfg *Config) (*VersionMeta, error) {
	path := filepath.Join(cfg.BaseDir, versionMetaFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta VersionMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// saveVersionMeta 把最后一次成功更新的版本信息原子写入 .iupdater-version.json
func saveVersionMeta(cfg *Config, task *UpdateTask) error {
	meta := VersionMeta{
		AppID:       cfg.AppID,
		VersionName: task.VersionName,
		VersionCode: task.VersionCode,
		UpdatedAt:   time.Now(),
		TaskID:      task.TaskID,
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(cfg.BaseDir, versionMetaFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (a *Agent) setState(state string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.State = state
}

func (a *Agent) setProgress(p int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.Progress = p
}

func (a *Agent) getStatus() AppStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

// Run 启动轮询主循环 + HTTP 面板，阻塞直到退出
func (a *Agent) Run() error {
	DefaultLogger.Info("iUpdater Agent 启动, app_id=%s channel=%s platform=%s arch=%s",
		a.cfg.AppID, a.cfg.Channel, a.cfg.Platform, a.cfg.Architecture)

	go a.pollOnce()
	go a.pollLoop()

	mux := http.NewServeMux()
	mux.Handle("/", a.webHandler())
	mux.HandleFunc("/api/status", a.apiStatus)
	mux.HandleFunc("/api/logs", a.apiLogs)
	mux.HandleFunc("/api/checknow", a.apiCheckNow)

	addr := fmt.Sprintf(":%d", a.cfg.APIPort)
	DefaultLogger.Info("本地管理面板监听 http://127.0.0.1%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (a *Agent) pollLoop() {
	ticker := time.NewTicker(a.cfg.PollInterval())
	defer ticker.Stop()
	for {
		select {
		case <-a.shutdown:
			return
		case <-ticker.C:
			a.pollOnce()
		}
	}
}

// buildStatusSnapshot 把本地面板状态 + 环境信息打包成 checkin 载荷
func (a *Agent) buildStatusSnapshot() map[string]any {
	st := a.getStatus()
	now := time.Now()
	return map[string]any{
		"state":           st.State,
		"current_version": st.CurrentVersion,
		"current_code":    st.CurrentCode,
		"last_check_at":   st.LastCheckAt,
		"last_update_at":  st.LastUpdateAt,
		"progress":        st.Progress,
		"task_id":         st.TaskID,
		"error":           st.Error,
		// 额外环境信息（主控面板可以直接展示）
		"hostname": hostnameSafe(),
		"uptime":   uptimeSafe(),
		// 当前运行时配置（敏感字段 access_key / server_url 不回传）
		"cfg": map[string]any{
			"app_id":                a.cfg.AppID,
			"channel":               a.cfg.Channel,
			"platform":              a.cfg.Platform,
			"architecture":          a.cfg.Architecture,
			"target_dir":            a.cfg.TargetDir,
			"backup_dir":            a.cfg.BackupDir,
			"download_dir":          a.cfg.DownloadDir,
			"backup_keep":           a.cfg.BackupKeep,
			"poll_interval_seconds": a.cfg.PollIntervalSec,
			"api_port":              a.cfg.APIPort,
			"stop_command":          a.cfg.EffectiveStopCommand(),
			"start_command":         a.cfg.EffectiveStartCommand(),
			"reported_at":           now,
		},
	}
}

// pollOnce 单次 checkin + 更新流程 + 执行主控下发的控制指令
func (a *Agent) pollOnce() {
	a.mu.Lock()
	a.status.LastCheckAt = time.Now()
	curVer := a.status.CurrentVersion
	curCode := a.status.CurrentCode
	a.mu.Unlock()

	snapshot := a.buildStatusSnapshot()

	DefaultLogger.Info("checkin (version=%s, code=%d, state=%s)", curVer, curCode, snapshot["state"])
	a.setState("checking")

	// 主控返回 task + 可能的 control 指令
	task, control := a.client.Checkin(a.cfg, curVer, curCode, snapshot)

	// ============ 先执行远程配置覆盖（不阻塞更新流程）============
	if control != nil && len(control.ConfigPatch) > 0 {
		DefaultLogger.Info("主控下发 config_patch: %v", control.ConfigPatch)
		if err := a.applyConfigPatch(control.ConfigPatch); err != nil {
			DefaultLogger.Warn("应用远程配置失败: %v", err)
		} else {
			DefaultLogger.Info("远程配置已合并到本地 config.json")
		}
	}

	// ============ 执行远程控制指令 ============
	if control != nil && control.Command != "" {
		switch control.Command {
		case "check_now":
			DefaultLogger.Info("主控下发 check_now，跳过本次更新检查，重新 poll")
			// 让外层循环立即再来一次（通过 goroutine 避免阻塞）
			a.setState("idle")
			time.AfterFunc(2*time.Second, func() { go a.pollOnce() })
			// 不做本次更新
			if task == nil {
				return
			}
		case "heartbeat":
			// 无操作，主控只是探活
			DefaultLogger.Debug("主控 heartbeat")
		case "restart":
			DefaultLogger.Info("主控下发 restart — 延迟 3s 重启自身")
			a.setState("idle")
			go func() {
				time.Sleep(3 * time.Second)
				execSelfRestart() // agent 自身重 exec，配置热加载后生效
			}()
			if task == nil {
				return
			}
		default:
			DefaultLogger.Warn("未知远程命令: %s", control.Command)
		}
	}

	// ============ 更新流程 ============
	if task == nil {
		DefaultLogger.Debug("无更新")
		a.setState("idle")
		return
	}

	DefaultLogger.Info("发现新版本: %s (%d) task=%s  force=%v", task.VersionName, task.VersionCode, task.TaskID, task.Force)

	a.mu.Lock()
	a.status.State = "downloading"
	a.status.TaskID = task.TaskID
	a.status.Progress = 0
	a.status.Error = ""
	a.mu.Unlock()

	a.setState("updating")
	err := a.updater.DoUpdate(task)

	a.mu.Lock()
	defer a.mu.Unlock()

	if err != nil {
		a.status.State = "fail"
		a.status.Error = err.Error()
		a.client.ReportResult(task, "fail", err.Error())
	} else {
		a.status.State = "success"
		a.status.CurrentVersion = task.VersionName
		a.status.CurrentCode = task.VersionCode
		a.status.LastUpdateAt = time.Now()
		a.status.Error = ""
		a.client.ReportResult(task, "success", "")
		// 原子写入本地元数据文件 — 重启后可恢复版本，避免主控重复推送
		if err := saveVersionMeta(a.cfg, task); err != nil {
			DefaultLogger.Warn("写入版本元数据失败: %v", err)
		} else {
			DefaultLogger.Info("版本元数据已写入 %s → %s (code=%d)", versionMetaFile, task.VersionName, task.VersionCode)
		}
	}
}

// applyConfigPatch 把主控下发的 config_patch 合并到本地 config.json
// 规则：patch 里非空字段覆盖本地；保留本地 patch 里没有的字段
func (a *Agent) applyConfigPatch(patch map[string]any) error {
	// 读现有 config.json
	cfgPath := filepath.Join(a.cfg.BaseDir, "config.json")
	raw, err := os.ReadFile(cfgPath)
	var local map[string]any
	if err != nil {
		// 没有 config.json 就从空开始写
		local = map[string]any{}
	} else {
		_ = json.Unmarshal(raw, &local)
	}

	// 合并 patch 中非空字段
	for k, v := range patch {
		if v == nil {
			continue
		}
		// 字符串不能空
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		local[k] = v
	}

	newRaw, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return err
	}

	// 原子写入：先写 .tmp 再 rename
	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, newRaw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, cfgPath)
}

// --- HTTP handlers ---

func (a *Agent) webHandler() http.Handler {
	sub, _ := fs.Sub(webFS, "web")
	return http.FileServer(http.FS(sub))
}

func (a *Agent) apiStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.getStatus())
}

func (a *Agent) apiLogs(w http.ResponseWriter, _ *http.Request) {
	entries := DefaultLogger.Snapshot(200)
	writeJSON(w, entries)
}

func (a *Agent) apiCheckNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := a.getStatus()
	if st.State == "downloading" || st.State == "updating" || st.State == "checking" {
		writeJSON(w, map[string]any{"ok": false, "message": "正在进行中，请稍候"})
		return
	}
	go a.pollOnce()
	writeJSON(w, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

// --- 辅助 ---

func hostnameSafe() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func uptimeSafe() int64 {
	// 简单返回进程启动以来的秒数，主控面板用这个显示 agent 运行时长
	start := time.Now()
	defer func() { time.Now() }()
	_ = start
	// 进程启动时间不好拿，这里简单返回 0 让主控展示为空即可
	// 真实 uptime 可以从 /proc/uptime 读取（Linux），但跨平台就不做了
	return 0
}

func execSelfRestart() {
	// Windows 下 exec 不能替换进程，这里只打印日志然后让下一次 poll 继续工作
	// 真正完整的重启应该由外部 supervisor（systemd / NSSM）处理
	DefaultLogger.Info("restart 指令收到 — agent 将在下次 poll 时继续工作；如需完整重启请由外部 supervisor 接管")
	// Linux exec:
	// exec.Command(os.Args[0], os.Args[1:]...).Run()
	// 但为了跨平台一致性，这里不做真正的重启
}

// --- main ---

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("配置加载失败: %v\n", err)
		return
	}
	agent := NewAgent(cfg)
	if err := agent.Run(); err != nil {
		DefaultLogger.Error("Agent 运行退出: %v", err)
	}
}
