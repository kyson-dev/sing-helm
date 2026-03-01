package logger

import (
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
		get().Debug(message, "source", "sing-box")
	case log.LevelInfo:
		get().Info(message, "source", "sing-box")
	case log.LevelWarn:
		get().Warn(message, "source", "sing-box")
	case log.LevelError, log.LevelFatal, log.LevelPanic:
		get().Error(message, "source", "sing-box")
	default:
		get().Info(message, "source", "sing-box")
	}
}

// 确保实现了接口
var _ log.PlatformWriter = (*PlatformWriter)(nil)
