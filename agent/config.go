package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config agent 运行时配置，字段与 config.json 一一对应
type Config struct {
	ServerURL       string `json:"server_url"`
	AccessKey       string `json:"access_key"`
	PollIntervalSec int    `json:"poll_interval_seconds"`
	AppID           string `json:"app_id"`
	Channel         string `json:"channel"`
	Platform        string `json:"platform"`
	Architecture    string `json:"architecture"`
	TargetDir       string `json:"target_dir"`
	BackupDir       string `json:"backup_dir"`
	DownloadDir     string `json:"download_dir"`

	// BackupKeep 保留的备份目录数量上限，0 或负数回退到默认值 1；最大 200，超出会被截断
	BackupKeep int `json:"backup_keep"`

	// StopCommand / StartCommand: 完整终端命令，支持任意形式：
	//   "./stop.bat"
	//   "./run.sh stop"
	//   "/etc/init.d/myapp restart"
	//   "systemctl restart myapp"
	//   "docker restart mycontainer"
	// 程序始终通过 shell 执行，保留空格 / 引号 / 管道等 shell 语义
	StopCommand  string `json:"stop_command"`
	StartCommand string `json:"start_command"`

	// 向后兼容：旧字段 stop_script / start_script 仍支持
	StopScript  string `json:"stop_script"`
	StartScript string `json:"start_script"`

	APIPort int `json:"api_port"`

	// 运行期计算字段
	BaseDir string `json:"-"` // agent 自身所在目录
}

// EffectiveStopCommand 优先用 stop_command，兼容旧字段 stop_script
func (c *Config) EffectiveStopCommand() string {
	if c.StopCommand != "" {
		return c.StopCommand
	}
	return c.StopScript
}

// EffectiveStartCommand 优先用 start_command，兼容旧字段 start_script
func (c *Config) EffectiveStartCommand() string {
	if c.StartCommand != "" {
		return c.StartCommand
	}
	return c.StartScript
}

// PollInterval 返回解析后的轮询间隔
func (c *Config) PollInterval() time.Duration {
	if c.PollIntervalSec <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.PollIntervalSec) * time.Second
}

// LoadConfig 加载配置：优先读取 --config 指定路径，其次读程序同目录 config.json
func LoadConfig() (*Config, error) {
	var configPath string
	flag.StringVar(&configPath, "config", "", "配置文件路径，默认与 agent 同目录 config.json")
	flag.Parse()

	if configPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.json")
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败 (%s): %w", configPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 基础校验
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("配置项 server_url 不能为空")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("配置项 app_id 不能为空")
	}
	if cfg.Platform == "" {
		cfg.Platform = runtimePlatform()
	}
	if cfg.Architecture == "" {
		cfg.Architecture = runtimeArch()
	}
	if cfg.PollIntervalSec <= 0 {
		cfg.PollIntervalSec = 60
	}
	if cfg.APIPort <= 0 {
		cfg.APIPort = 18080
	}
	// BackupKeep: 0 = 禁用备份（永不触发）；负数 回退默认 1；上限 200 超限截断
	switch {
	case cfg.BackupKeep < 0:
		cfg.BackupKeep = 1
	case cfg.BackupKeep > 200:
		cfg.BackupKeep = 200
	}

	cfg.BaseDir = filepath.Dir(configPath)
	// 相对路径转绝对（相对 agent 自身目录）
	cfg.TargetDir = resolveAbs(cfg.BaseDir, cfg.TargetDir)
	cfg.BackupDir = resolveAbs(cfg.BaseDir, cfg.BackupDir)
	cfg.DownloadDir = resolveAbs(cfg.BaseDir, cfg.DownloadDir)

	// 处理命令：只把命令里的"第一个词"（可执行文件/脚本路径）做相对路径解析，
	// 后面的参数保留原样（包括参数里的引号、空格、分号等）
	cfg.StopCommand = resolveCmdFirstToken(cfg.BaseDir, cfg.EffectiveStopCommand())
	cfg.StartCommand = resolveCmdFirstToken(cfg.BaseDir, cfg.EffectiveStartCommand())
	cfg.StopScript = "" // 已合并到 StopCommand，清空避免重复
	cfg.StartScript = ""

	return &cfg, nil
}

// resolveAbs 把相对路径（相对 base）转绝对路径；空值 / 已是绝对原样返回
func resolveAbs(base, p string) string {
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

// resolveCmdFirstToken 解析命令字符串里的"第一个词"（可执行文件/脚本）：
//
//	"./stop.bat"                 → "C:/agent/stop.bat"           （完整替换）
//	"./run.sh stop"              → "C:/agent/run.sh stop"         （只替换第一个词）
//	"/etc/init.d/app restart"    → 不变（绝对路径）
//	"systemctl restart myapp"    → 不变（系统命令）
//
// 规则：第一个词不含路径分隔符（/ 或 \）→ 认为是系统命令，原样返回
func resolveCmdFirstToken(base, cmd string) string {
	if cmd == "" {
		return cmd
	}
	// 找第一个非空白字符的起点 + 终点，处理引号包裹的情况
	firstWord, rest := splitFirstToken(cmd)

	// 无路径分隔符 → 系统命令（如 systemctl、docker、service），原样返回
	if !strings.ContainsAny(firstWord, `/\`) {
		return cmd
	}

	// 相对或绝对路径 → 只替换第一个词
	newFirst := resolveAbs(base, firstWord)
	return newFirst + rest
}

// splitFirstToken 把命令行切成「第一个词（含引号处理）」和「剩余部分」
// 例："./run.sh stop  -f" → "./run.sh" + " stop  -f"
//
//	"\"C:/path/app.exe\" -x y" → "\"C:/path/app.exe\"" + " -x y"
func splitFirstToken(cmd string) (string, string) {
	i := 0
	n := len(cmd)
	// 跳过前导空白
	for i < n && (cmd[i] == ' ' || cmd[i] == '\t') {
		i++
	}
	start := i
	inQuote := false
	for i < n {
		c := cmd[i]
		if c == '"' {
			inQuote = !inQuote
			i++
			continue
		}
		if !inQuote && (c == ' ' || c == '\t') {
			break
		}
		i++
	}
	return cmd[start:i], cmd[i:]
}

// 保留 runtimePlatform / runtimeArch 函数（config.go 里也用到）
func runtimePlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

func runtimeArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}
