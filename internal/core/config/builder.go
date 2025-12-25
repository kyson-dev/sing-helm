package config

import (
	"context"
	"fmt"
	"os"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// ConfigBuilder 配置构建器
// 支持链式调用添加模块，灵活组装配置
type ConfigBuilder struct {
	base    *option.Options // 用户配置作为基础
	opts    *RunOptions     // 运行时参数
	modules []ConfigModule  // 配置模块列表
	ctx     *BuildContext   // 构建上下文
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder(base *option.Options, opts *RunOptions) *ConfigBuilder {
	if base == nil {
		base = &option.Options{}
	}
	if opts == nil {
		defaultOpts := DefaultRunOptions()
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

	// 2. 预处理：提取用户节点信息
	b.extractNodeTags(result)

	// 3. 依次应用各模块
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

// extractNodeTags 从用户配置中提取节点信息
func (b *ConfigBuilder) extractNodeTags(opts *option.Options) {
	reservedTags := map[string]bool{
		"direct": true,
		"block":  true,
		"proxy":  true,
		"auto":   true,
	}

	for _, out := range opts.Outbounds {
		tag := out.Tag
		if reservedTags[tag] {
			continue
		}

		b.ctx.UserNodeTags = append(b.ctx.UserNodeTags, tag)

		// 判断是否为实际代理节点（非 selector/urltest）
		outMap := map[string]any{}
		data, _ := singboxjson.Marshal(out)
		singboxjson.Unmarshal(data, &outMap)
		outType, _ := outMap["type"].(string)
		if outType != "selector" && outType != "urltest" {
			b.ctx.ActualNodes = append(b.ctx.ActualNodes, tag)
		}
	}
}

// DefaultModules 根据 RunOptions 返回默认模块组合
func DefaultModules(opts *RunOptions) []ConfigModule {
	modules := []ConfigModule{
		&OutboundModule{},
	}

	// 根据 ProxyMode 选择入站模块
	switch opts.ProxyMode {
	case ProxyModeTUN:
		modules = append(modules,
			&TUNModule{},
			&TUNDNSModule{},
		)
	case ProxyModeSystem:
		modules = append(modules, &MixedModule{
			SetSystemProxy: true,
			ListenAddr:     opts.ListenAddr,
			Port:           opts.MixedPort,
		})
	case ProxyModeDefault:
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
