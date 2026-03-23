package engine

import (
	"regexp"

	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/sagernet/sing-box/log"
)

// PlatformWriter 实现 sing-box 的 log.PlatformWriter 接口
// 将 sing-box 的日志重定向到我们的 slog logger
type PlatformWriter struct{}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func NewPlatformWriter() log.PlatformWriter {
	return &PlatformWriter{}
}

func (p *PlatformWriter) DisableColors() bool {
	// 禁用颜色，因为我们要写入文件或者由 slog 处理
	return true
}

func (p *PlatformWriter) WriteMessage(level log.Level, message string) {
	clean := ansiEscapePattern.ReplaceAllString(message, "")
	switch level {
	case log.LevelTrace, log.LevelDebug:
		logger.Debug(clean, "source", "sing-box")
	case log.LevelInfo:
		logger.Info(clean, "source", "sing-box")
	case log.LevelWarn:
		// logger doesn't have Warn exposed, using Info for now
		logger.Info("[WARN] "+clean, "source", "sing-box")
	case log.LevelError, log.LevelFatal, log.LevelPanic:
		logger.Error(clean, "source", "sing-box")
	default:
		logger.Info(clean, "source", "sing-box")
	}
}

// 确保实现了接口
var _ log.PlatformWriter = (*PlatformWriter)(nil)
