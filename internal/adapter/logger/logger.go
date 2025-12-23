package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	instance *slog.Logger
	once     sync.Once
)

// Set up logger level
func Setup(debug bool) {
	once.Do(func() {
		ops := &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}
		if debug {
			ops.Level = slog.LevelDebug
		}
		handel := slog.NewTextHandler(os.Stdout, ops)
		instance = slog.New(handel)
		slog.SetDefault(instance)
	})
}

// Get logger instance
func Get() *slog.Logger {
	if instance == nil {
		Setup(false)
	}
	return instance
}

// log 是内部辅助函数，用于正确显示调用位置
// skip 参数指定要跳过的调用栈层数
func log(level slog.Level, msg string, args ...any) {
	l := Get()
	if !l.Enabled(context.Background(), level) {
		return
	}

	var pcs [1]uintptr
	// skip=3: runtime.Callers, log, Info/Error/Debug
	runtime.Callers(3, pcs[:])

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)

	_ = l.Handler().Handle(context.Background(), r)
}

// Info logs at Info level with correct source location
func Info(msg string, args ...any) {
	log(slog.LevelInfo, msg, args...)
}

// Error logs at Error level with correct source location
func Error(msg string, args ...any) {
	log(slog.LevelError, msg, args...)
}

// Debug logs at Debug level with correct source location
func Debug(msg string, args ...any) {
	log(slog.LevelDebug, msg, args...)
}
