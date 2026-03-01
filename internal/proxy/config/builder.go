package config

import (
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	"github.com/kyson-dev/sing-helm/internal/core/model"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/sagernet/sing-box/option"
)

// Builder 配置构建器
// 支持链式调用添加模块，灵活组装配置
type Builder struct {
	opts    *model.RunOptions // 运行时参数
	modules []module.ConfigModule    // 配置模块列表
	ctx     *module.BuildContext     // 构建上下文
}

// NewBuilder 创建配置构建器（从已加载的配置）
func NewBuilder(opts *model.RunOptions) *Builder {
	if opts == nil {
		defaultOpts := model.DefaultRunOptions()
		opts = &defaultOpts
	}
	return &Builder{
		opts:    opts,
		modules: []module.ConfigModule{},
		ctx:     module.NewBuildContext(opts),
	}
}

// With 添加一个模块（链式调用）
func (b *Builder) With(m module.ConfigModule) *Builder {
	b.modules = append(b.modules, m)
	return b
}

// Build 构建完整的 sing-box 配置
func (b *Builder) Build() (*option.Options, error) {
	// 1. 复制用户配置作为基础
	result := &option.Options{}

	// 2. 依次应用各模块
	for _, m := range b.modules {
		logger.Debug("Applying config module", "name", m.Name())
		if err := m.Apply(result, b.ctx); err != nil {
			return nil, fmt.Errorf("module %s failed: %w", m.Name(), err)
		}
	}

	return result, nil
}


