package engine

import (
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/sagernet/sing-box/log"
)

// PlatformWriter 实现 sing-box 的 log.PlatformWriter 接口
// 将 sing-box 的日志重定向到我们的 slog logger
type PlatformWriter struct{}

func NewPlatformWriter() log.PlatformWriter {
	return &PlatformWriter{}
}

func (p *PlatformWriter) DisableColors() bool {
	// 禁用颜色，因为我们要写入文件或者由 slog 处理
	return true
}

func (p *PlatformWriter) WriteMessage(level log.Level, message string) {
	switch level {
	case log.LevelTrace, log.LevelDebug:
		logger.Debug(message, "source", "sing-box")
	case log.LevelInfo:
		logger.Info(message, "source", "sing-box")
	case log.LevelWarn:
		// logger doesn't have Warn exposed, using Info for now
		logger.Info("[WARN] "+message, "source", "sing-box")
	case log.LevelError, log.LevelFatal, log.LevelPanic:
		logger.Error(message, "source", "sing-box")
	default:
		logger.Info(message, "source", "sing-box")
	}
}

// 确保实现了接口
var _ log.PlatformWriter = (*PlatformWriter)(nil)
