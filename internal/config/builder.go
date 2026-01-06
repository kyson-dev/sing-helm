package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyson/minibox/internal/logger"
	"github.com/kyson/minibox/internal/runtime"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// ConfigBuilder 配置构建器
// 支持链式调用添加模块，灵活组装配置
type ConfigBuilder struct {
	base    *option.Options     // 用户配置作为基础
	opts    *runtime.RunOptions // 运行时参数
	modules []ConfigModule      // 配置模块列表
	ctx     *BuildContext       // 构建上下文
}

// BuildConfig loads the profile, applies runtime modules, and saves raw config.
func BuildConfig(rawPath string, runops *runtime.RunOptions) error {
	// 使用新的 API，UserOutboundModule 会自动加载配置文件
	builder := NewConfigBuilder(nil, runops)
	for _, m := range defaultModules(runops) {
		builder.With(m)
	}

	if err := builder.SaveToFile(rawPath); err != nil {
		return fmt.Errorf("failed to save raw config: %w", err)
	}

	return nil
}

// BuildOptions builds a sing-box config without writing to disk.
func BuildOptions(runops *runtime.RunOptions) (*option.Options, error) {
	builder := NewConfigBuilder(nil, runops)
	for _, m := range defaultModules(runops) {
		builder.With(m)
	}
	return builder.Build()
}

// NewConfigBuilder 创建配置构建器（从已加载的配置）
// 参数:
//   - base: 已加载的用户配置（可以为 nil）
//   - opts: 运行时参数
//
// 注意: 这是向后兼容的方法，推荐使用 NewConfigBuilderFromFile
func NewConfigBuilder(base *option.Options, opts *runtime.RunOptions) *ConfigBuilder {
	if base == nil {
		base = &option.Options{}
	}
	if opts == nil {
		defaultOpts := runtime.DefaultRunOptions()
		opts = &defaultOpts
	}
	return &ConfigBuilder{
		base:    base,
		opts:    opts,
		modules: []ConfigModule{},
		ctx:     NewBuildContext(opts),
	}
}

// With 添加一个模块（链式调用）
func (b *ConfigBuilder) With(m ConfigModule) *ConfigBuilder {
	b.modules = append(b.modules, m)
	return b
}

// Build 构建完整的 sing-box 配置
func (b *ConfigBuilder) Build() (*option.Options, error) {
	// 1. 复制用户配置作为基础
	result, err := b.cloneBase()
	if err != nil {
		return nil, fmt.Errorf("failed to clone base config: %w", err)
	}

	// 2. 依次应用各模块
	for _, m := range b.modules {
		logger.Debug("Applying config module", "name", m.Name())
		if err := m.Apply(result, b.ctx); err != nil {
			return nil, fmt.Errorf("module %s failed: %w", m.Name(), err)
		}
	}

	return result, nil
}

// SaveToFile 构建配置并保存到文件
func (b *ConfigBuilder) SaveToFile(path string) error {
	opts, err := b.Build()
	if err != nil {
		return err
	}

	// 使用 sing-box 的 JSON 序列化
	data, err := singboxjson.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Re-marshal for pretty print
	var pretty interface{}
	if err := json.Unmarshal(data, &pretty); err != nil {
		return fmt.Errorf("failed to unmarshal for pretty print: %w", err)
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

// cloneBase 复制用户配置
func (b *ConfigBuilder) cloneBase() (*option.Options, error) {
	// 通过序列化/反序列化来深拷贝
	data, err := singboxjson.Marshal(b.base)
	if err != nil {
		return nil, err
	}

	var result option.Options
	ctx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(ctx, data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DefaultModules 根据 RunOptions 返回默认模块组合
func defaultModules(opts *runtime.RunOptions) []ConfigModule {
	modules := []ConfigModule{
		&UserOutboundModule{},
		&SubscriptionModule{},
		&OutboundModule{},
	}

	// 根据 ProxyMode 选择入站模块
	switch opts.ProxyMode {
	case runtime.ProxyModeTUN:
		modules = append(modules,
			&TUNModule{},
			&TUNDNSModule{},
		)
	case runtime.ProxyModeSystem:
		modules = append(modules, &MixedModule{
			SetSystemProxy: true,
			ListenAddr:     opts.ListenAddr,
			Port:           opts.MixedPort,
		})
	case runtime.ProxyModeDefault:
		modules = append(modules, &MixedModule{
			SetSystemProxy: false,
			ListenAddr:     opts.ListenAddr,
			Port:           opts.MixedPort,
		})
	}

	modules = append(modules,
		&RouteModule{RouteMode: opts.RouteMode},
		&ExperimentalModule{
			ListenAddr: opts.ListenAddr,
			APIPort:    opts.APIPort,
		},
		&LogModule{},
	)

	return modules
}
