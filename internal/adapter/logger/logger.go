package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	instance *slog.Logger
	once     sync.Once
)

type Config struct {
	Debug    bool
	FilePath string // 如果为空，则只输出到 stdout
}

// Set up logger level
func Setup(cfg Config) {
	once.Do(func() {
		var writer io.Writer = os.Stdout

		// 如果指定了文件路径，则使用 MultiWriter (同时写文件和屏幕)
		// 或者仅写文件（取决于你的需求，后台模式通常只写文件）
		if cfg.FilePath != "" {
			_ = os.MkdirAll(filepath.Dir(cfg.FilePath), 0755)
			fileLogger := &lumberjack.Logger{
				Filename:   cfg.FilePath,
				MaxSize:    10,   // 每个日志文件最大 10MB
				MaxBackups: 3,    // 保留最近 3 个文件
				MaxAge:     28,   // 保留 28 天
				Compress:   true, // 压缩旧日志
			}
			
			// 如果是前台运行，可能希望同时看到；如果是后台，通常只写文件
			// 这里我们为了通用，如果配置了文件，就只写文件（避免后台运行时 stdout 满）
			// 或者你可以用 io.MultiWriter(os.Stdout, fileLogger)
			// writer = io.MultiWriter(os.Stdout, fileLogger)
			writer = fileLogger
		}
		
		ops := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
		}
		if cfg.Debug {
			ops.Level = slog.LevelDebug
			ops.AddSource = true // Debug 模式显示源码位置
		}
		handel := slog.NewTextHandler(writer, ops)
		instance = slog.New(handel)
		slog.SetDefault(instance)
	})
}

// Get logger instance
func Get() *slog.Logger {
	if instance == nil {
		Setup(Config{})
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
