package module

import (
	"github.com/kyson-dev/sing-helm/internal/proxy/engine/config"
	"github.com/sagernet/sing-box/option"
)

// LogModule 日志模块
type LogModule struct {
	Level string
}

func (m *LogModule) Name() string {
	return "log"
}

func (m *LogModule) Apply(opts *option.Options, ctx *config.BuildContext) error {
	level := m.Level
	if level == "" {
		level = "info"
	}

	// 如果用户没有配置日志，使用默认配置
	if opts.Log == nil {
		opts.Log = &option.LogOptions{}
	}

	opts.Log.Level = level

	return nil
}
