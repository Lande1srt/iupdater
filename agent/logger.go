package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogEntry 单条日志
type LogEntry struct {
	Time time.Time `json:"time"`
	Level string   `json:"level"`
	Msg   string   `json:"msg"`
}

// RingLogger 内存环形日志 + 标准输出双写
type RingLogger struct {
	mu    sync.Mutex
	buf   []LogEntry
	cap   int
	index int
	out   io.Writer
}

// DefaultLogger 全局实例
var DefaultLogger = NewRingLogger(2000, os.Stdout)

// NewRingLogger 创建一个环形日志器
func NewRingLogger(capacity int, out io.Writer) *RingLogger {
	if capacity <= 0 {
		capacity = 2000
	}
	return &RingLogger{
		buf: make([]LogEntry, 0, capacity),
		cap: capacity,
		out: out,
	}
}

// Log 写入一条日志到环形缓冲 + 标准输出
func (l *RingLogger) Log(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{Time: time.Now(), Level: level, Msg: msg}

	l.mu.Lock()
	if len(l.buf) < l.cap {
		l.buf = append(l.buf, entry)
	} else {
		l.buf[l.index] = entry
		l.index = (l.index + 1) % l.cap
	}
	l.mu.Unlock()

	// 同时输出到控制台（方便直接运行时调试）
	fmt.Fprintf(l.out, "%s [%s] %s\n", entry.Time.Format("2006-01-02 15:04:05"), level, msg)
}

func (l *RingLogger) Info(format string, args ...any)  { l.Log("INFO", format, args...) }
func (l *RingLogger) Warn(format string, args ...any)  { l.Log("WARN", format, args...) }
func (l *RingLogger) Error(format string, args ...any)  { l.Log("ERROR", format, args...) }
func (l *RingLogger) Debug(format string, args ...any) { l.Log("DEBUG", format, args...) }

// Snapshot 拷贝最近 N 条日志供面板查看（N=0 返回全部）
func (l *RingLogger) Snapshot(n int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	total := len(l.buf)
	if n <= 0 || n >= total {
		n = total
	}
	start := total - n
	// 由于是环形，需要按时间顺序重排
	sorted := make([]LogEntry, 0, n)
	for i := 0; i < total; i++ {
		sorted = append(sorted, l.buf[(l.index+i)%l.cap])
	}
	// 截取尾部 n 条
	if start < 0 {
		start = 0
	}
	return sorted[total-n:]
}
