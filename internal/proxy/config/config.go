package config

import (
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/core/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	"github.com/sagernet/sing-box/option"
)

// BuildConfig loads the profile, applies runtime modules, and saves raw config.
func BuildConfig(rawPath string, runops *model.RunOptions) error {
	builder := NewBuilder(runops)
	for _, m := range DefaultModules(runops) {
		builder.With(m)
	}

	if err := builder.SaveToFile(rawPath); err != nil {
		return fmt.Errorf("failed to save raw config: %w", err)
	}

	return nil
}

// BuildOptions builds a sing-box config without writing to disk.
func BuildOptions(runops *model.RunOptions) (*option.Options, error) {
	builder := NewBuilder(runops)
	for _, m := range DefaultModules(runops) {
		builder.With(m)
	}
	return builder.Build()
}

// DefaultModules 根据 RunOptions 返回默认模块组合
func DefaultModules(opts *model.RunOptions) []module.ConfigModule {
	if opts == nil {
		defaultOpts := model.DefaultRunOptions()
		opts = &defaultOpts
	}

	modules := []module.ConfigModule{
		&module.OutboundModule{
			Providers: []module.NodeProvider{
				&module.UserNodeProvider{},
				&module.SubscriptionNodeProvider{},
			},
		},
	}

	// 根据 ProxyMode 选择入站模块
	switch opts.ProxyMode {
	case model.ProxyModeTUN:
		modules = append(modules,
			&module.TUNModule{},
			&module.TUNDNSModule{},
		)
	case model.ProxyModeSystem:
		modules = append(modules, &module.MixedModule{
			SetSystemProxy: true,
			ListenAddr:     opts.ListenAddr,
			Port:           opts.MixedPort,
		})
	case model.ProxyModeDefault:
		modules = append(modules, &module.MixedModule{
			SetSystemProxy: false,
			ListenAddr:     opts.ListenAddr,
			Port:           opts.MixedPort,
		})
	}

	modules = append(modules,
		&module.RouteModule{RouteMode: opts.RouteMode},
		&module.ExperimentalModule{
			ListenAddr: opts.ListenAddr,
			APIPort:    opts.APIPort,
		},
		&module.LogModule{},
	)

	return modules
}
