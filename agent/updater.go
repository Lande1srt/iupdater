package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Updater 执行具体的更新流程
type Updater struct {
	cfg        *Config
	cli        *Client
	onProgress func(pct int) // 可选：进度回调（用于驱动本地面板）
}

// NewUpdater 构造 Updater
func NewUpdater(cfg *Config, cli *Client, onProgress func(pct int)) *Updater {
	return &Updater{cfg: cfg, cli: cli, onProgress: onProgress}
}

// reportProgress 向上层回调，nil 时静默
func (u *Updater) reportProgress(pct int) {
	if u.onProgress != nil {
		u.onProgress(pct)
	}
}

// scriptRunner 返回适合当前平台的 shell 执行器（保留，供 runCommand 使用）
func scriptRunner() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C"}
	}
	return "/bin/sh", []string{"-c"}
}

// DoUpdate 执行完整更新流程：下载 → 校验 → stop → 备份 → 解压覆盖 → start → 上报 → 清理
func (u *Updater) DoUpdate(task *UpdateTask) error {
	cfg := u.cfg

	// 0. 准备目录
	if err := os.MkdirAll(cfg.DownloadDir, 0o755); err != nil {
		return fmt.Errorf("创建下载目录失败: %w", err)
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	zipPath := filepath.Join(cfg.DownloadDir, task.TaskID+".zip")
	tempDir := filepath.Join(cfg.DownloadDir, task.TaskID)

	// 1. 下载
	DefaultLogger.Info("开始下载任务 %s → %s", task.TaskID, task.DownloadURL)
	start := time.Now()
	// 下载进度日志节流：按 5% 台阶 + 1s 最小间隔，避免每 8KB 就刷一条
	var (
		lastPct     int = -1
		lastLogAt   time.Time
		logThrottle = func(d, total int64) {
			pct := 0
			if total > 0 {
				pct = int(d * 100 / total)
			} else {
				pct = -1
			}
			forcePrint := pct == 0 || (total > 0 && pct >= 100)
			enoughStep := total > 0 && (pct-lastPct >= 5 || pct > lastPct)
			enoughTime := time.Since(lastLogAt) >= time.Second
			if !forcePrint && !enoughStep && !enoughTime {
				// 顺便把 progress 推给上层（但不打日志）
				if total > 0 {
					u.reportProgress(pct)
				} else {
					u.reportProgress(int(float64(d) / (1024 * 1024)))
				}
				return
			}
			lastPct = pct
			lastLogAt = time.Now()
			if total > 0 {
				DefaultLogger.Info("下载进度 %3d%%  %s / %s",
					pct, humanSizeBytes(d), humanSizeBytes(total))
				u.reportProgress(pct)
			} else {
				DefaultLogger.Info("下载中  %s（总大小未知）", humanSizeBytes(d))
				u.reportProgress(int(float64(d) / (1024 * 1024)))
			}
		}
	)
	err := u.cli.Download(task.DownloadURL, zipPath, logThrottle)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	DefaultLogger.Info("下载完成 %s, 耗时 %v", humanSize(zipPath), time.Since(start).Round(time.Second))

	// 2. SHA256 校验
	if task.FileHash != "" {
		actual, err := fileSHA256(zipPath)
		if err != nil {
			return fmt.Errorf("校验文件 hash 失败: %w", err)
		}
		if actual != task.FileHash {
			return fmt.Errorf("文件 hash 不匹配: 期望 %s 实际 %s", task.FileHash, actual)
		}
		DefaultLogger.Info("SHA256 校验通过")
	}

	// 3. stop_command
	if cmd := cfg.EffectiveStopCommand(); cmd != "" {
		DefaultLogger.Info("执行停止命令: %s", cmd)
		if err := u.runCommand(cmd); err != nil {
			return fmt.Errorf("停止命令失败: %w", err)
		}
		DefaultLogger.Info("停止命令执行完成")
	}

	// 4. 备份 target_dir（backup_keep=0 表示彻底禁用）
	// 重要：backupDir 本身绝不允许出现在 target_dir 里（用户配置 target_dir 包含 backup_dir 时），
	// 否则 copyDir 会把备份副本作为 target 的子目录再次递归拷贝，导致磁盘无限膨胀。
	var backupDir string
	switch {
	case cfg.BackupKeep == 0:
		DefaultLogger.Info("备份已禁用（backup_keep=0）")
	case insideOrEqualDir(cfg.TargetDir, cfg.BackupDir):
		DefaultLogger.Warn("target_dir (%s) 包含 backup_dir (%s)，跳过备份以避免嵌套膨胀", cfg.TargetDir, cfg.BackupDir)
	default:
		backupDir = filepath.Join(cfg.BackupDir, time.Now().Format("20060102-150405"))
		if err := copyDirSafe(cfg.TargetDir, backupDir, cfg.BackupDir, backupDir); err != nil {
			DefaultLogger.Warn("备份失败，继续更新但可能导致无法回滚: %v", err)
			backupDir = ""
		} else {
			DefaultLogger.Info("已备份到 %s", backupDir)
		}
	}

	// 5. 解压覆盖
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		u.rollback(backupDir)
		return fmt.Errorf("创建临时解压目录失败: %w", err)
	}

	if err := unzip(zipPath, tempDir); err != nil {
		u.rollback(backupDir)
		return fmt.Errorf("解压失败: %w", err)
	}

	if err := replaceDir(cfg.TargetDir, tempDir); err != nil {
		u.rollback(backupDir)
		return fmt.Errorf("覆盖写入失败: %w", err)
	}

	// 清理临时目录和 zip
	_ = os.RemoveAll(tempDir)
	_ = os.Remove(zipPath)

	// 6. start_command
	var startErr error
	if cmd := cfg.EffectiveStartCommand(); cmd != "" {
		DefaultLogger.Info("执行启动命令: %s", cmd)
		startErr = u.runCommand(cmd)
		if startErr != nil {
			DefaultLogger.Warn("启动命令执行失败 (文件已替换): %v", startErr)
		} else {
			DefaultLogger.Info("启动命令执行完成")
		}
	}

	// 7. 清理旧备份
	cleanupOldBackups(cfg.BackupDir, cfg.BackupKeep)

	return startErr // 返回 start 命令错误，让上层决定如何上报
}

// rollback 从备份目录恢复 target_dir
func (u *Updater) rollback(backupDir string) {
	if backupDir == "" {
		return
	}
	DefaultLogger.Warn("触发回滚，从 %s 恢复", backupDir)
	_ = os.RemoveAll(u.cfg.TargetDir)
	if err := copyDir(backupDir, u.cfg.TargetDir); err != nil {
		DefaultLogger.Error("回滚失败: %v", err)
	} else {
		DefaultLogger.Info("回滚完成")
	}
}

// runCommand 通过 shell 执行完整命令字符串（支持任意形式：脚本+参数、系统命令、管道、分号、引号等）
// 超时 30s，工作目录设为 agent 自身 BaseDir
func (u *Updater) runCommand(cmdLine string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shell, shellArgs := scriptRunner()
	shellArgs = append(shellArgs, cmdLine)

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = u.cfg.BaseDir // 工作目录为 agent 自身目录，确保相对路径脚本也能找到
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (shell: %s %s), 输出:\n%s", err, shell, cmdLine, string(out))
	}
	return nil
}

// replaceDir 用 srcDir 内容原子替换 dstDir：
// 先把 src 解压内容里的每个文件对应的 dst 文件删除，再重命名
func replaceDir(dstDir, srcDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// 删除旧文件（如果存在）
		_ = os.Remove(target)
		// 父目录确保存在
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		// 原子重命名（同一文件系统下 os.Rename 原子）
		return os.Rename(path, target)
	})
}

// copyDir 完整拷贝 src → dst（包括目录结构）
func copyDir(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // 源不存在，跳过
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

// copyDirSafe 同 copyDir，但在遍历 src 时跳过任何「等于或嵌套进 skipRoot 的路径」。
// 用来防止 target_dir 包含 backup_dir 时把备份副本递归复制进备份目录自身，造成磁盘无限膨胀。
func copyDirSafe(src, dst string, skipRoots ...string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	cleaned := make([]string, 0, len(skipRoots))
	for _, s := range skipRoots {
		if s != "" {
			if a, err := filepath.Abs(s); err == nil {
				cleaned = append(cleaned, filepath.Clean(a))
			}
		}
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过任何落在 skipRoot 下的条目（把 skipRoot 自身也跳过）
		abs, err := filepath.Abs(path)
		if err == nil {
			abs = filepath.Clean(abs)
			for _, skip := range cleaned {
				if abs == skip || isSubpath(skip, abs) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

// insideOrEqualDir 判断 inner 是否等于 outer 或是 outer 的子目录（两边都会取绝对路径 + clean）。
// insideOrEqualDir("/a/b", "/a") → true
// insideOrEqualDir("/a/b", "/a/b") → true
// insideOrEqualDir("/a", "/a/b") → false（outer 反而在 inner 里）
func insideOrEqualDir(inner, outer string) bool {
	aIn, err1 := filepath.Abs(inner)
	aOut, err2 := filepath.Abs(outer)
	if err1 != nil || err2 != nil {
		return false
	}
	aIn = filepath.Clean(aIn)
	aOut = filepath.Clean(aOut)
	if aIn == aOut {
		return true
	}
	return isSubpath(aOut, aIn)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	si, _ := in.Stat()
	if si != nil {
		_ = os.Chmod(dst, si.Mode())
	}
	return nil
}

// unzip 解压 zip 到 destDir
func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		// 防止 Zip Slip 路径穿越
		if !isSubpath(destDir, target) {
			return fmt.Errorf("非法压缩路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if f.FileInfo() != nil {
			_ = os.Chmod(target, f.FileInfo().Mode())
		}
	}
	return nil
}

// isSubpath 确保 child 在 parent 内部，防止 Zip Slip
func isSubpath(parent, child string) bool {
	absP, errP := filepath.Abs(parent)
	absC, errC := filepath.Abs(child)
	if errP != nil || errC != nil {
		return false
	}
	rel, err := filepath.Rel(absP, absC)
	if err != nil {
		return false
	}
	return rel != ".." && len(rel) > 0 && rel[0] != '.'
}

// fileSHA256 计算文件 SHA256
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// cleanupOldBackups 清理 backupDir 中时间最老的文件，只保留最近 keep 个
func cleanupOldBackups(backupDir string, keep int) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}
	// 按名称（时间戳格式）排序，时间晚的在后面
	sortEntries(entries)
	if len(entries) <= keep {
		return
	}
	for i := 0; i < len(entries)-keep; i++ {
		p := filepath.Join(backupDir, entries[i].Name())
		if err := os.RemoveAll(p); err != nil {
			DefaultLogger.Warn("清理旧备份失败 %s: %v", p, err)
		} else {
			DefaultLogger.Debug("已清理旧备份 %s", p)
		}
	}
}

// sortEntries 按名字升序排序（时间戳格式天然可按字符串排序）
func sortEntries(entries []os.DirEntry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Name() < entries[j-1].Name(); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// humanSizeBytes 把字节数转成可读格式（12.3 MB / 512 KB）——给下载进度日志用，不必读文件
func humanSizeBytes(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	case size <= 0:
		return "0 B"
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// humanSize 以字节数可读格式返回文件大小
func humanSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path
	}
	size := info.Size()
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// createFile 或 os.Create 但先确保父目录存在
func createFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}
