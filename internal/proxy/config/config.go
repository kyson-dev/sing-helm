package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/core/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	nodeProvider "github.com/kyson-dev/sing-helm/internal/proxy/config/module/node"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// BuildConfig loads the profile, applies runtime modules, and saves raw config.
func BuildConfig(rawPath string, runops *model.RunOptions) error {
	builder := NewBuilder(runops)
	for _, m := range DefaultModules(runops) {
		builder.With(m)
	}

	opts, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	if err := SaveToFile(rawPath, opts); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
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
		module.NewOutboundModule(
			&nodeProvider.UserNodeProvider{},
			&nodeProvider.SubscriptionNodeProvider{},
		),
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

// SaveToFile 构建配置并保存到文件
func SaveToFile(path string, opts *option.Options) error {
	// 使用 sing-box 的 JSON 序列化
	data, err := singboxjson.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Re-marshal for pretty print
	// Re-marshal for pretty print
	var pretty map[string]any
	if err := json.Unmarshal(data, &pretty); err != nil {
		return fmt.Errorf("failed to unmarshal for pretty print: %w", err)
	}

	// Hotfix for sing-box v1.13.0-rc.7 removing 'format' from output JSON.
	// We inject 'format' back inside 'rule_set' to maintain compat with v1.11.x
	if routeStruct, ok := pretty["route"].(map[string]any); ok {
		if ruleSets, ok := routeStruct["rule_set"].([]any); ok {
			for idx, rs := range ruleSets {
				if rsMap, ok := rs.(map[string]any); ok {
					if _, hasFormat := rsMap["format"]; !hasFormat {
						// guess format by url/path extension
						urlStr, _ := rsMap["url"].(string)
						if urlStr == "" {
							urlStr, _ = rsMap["path"].(string)
						}
						if urlStr != "" {
							if strings.HasSuffix(urlStr, ".srs") {
								rsMap["format"] = "binary"
							} else if strings.HasSuffix(urlStr, ".json") {
								rsMap["format"] = "source"
							}
							ruleSets[idx] = rsMap
						}
					}
				}
			}
			routeStruct["rule_set"] = ruleSets
		}
	}

	data, err = json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal indent: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("Config saved", "path", path)
	return nil
}
