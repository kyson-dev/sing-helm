package logger

import (
	"log/slog"
	"os"
	"sync"
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
		//TOTO: Use Text First
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

func Info(msg string, args ...any)  { Get().Info(msg, args...) }
func Error(msg string, args ...any) { Get().Error(msg, args...) }
func Debug(msg string, args ...any) { Get().Debug(msg, args...) }
