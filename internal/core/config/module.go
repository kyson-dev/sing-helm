package config

import (
	"github.com/sagernet/sing-box/option"
)

// ConfigModule 配置模块接口
// 每个模块负责配置的一个部分，可以灵活组装
type ConfigModule interface {
	// Name 返回模块名称，用于日志和调试
	Name() string
	// Apply 将模块的配置应用到 opts 上
	Apply(opts *option.Options, ctx *BuildContext) error
}

// BuildContext 构建上下文，模块间共享数据
type BuildContext struct {
	// RunOptions 运行时参数
	RunOptions *RunOptions

	// UserNodeTags 用户配置的所有节点 tags（包括 selector/urltest 组）
	UserNodeTags []string

	// ActualNodes 实际代理节点的 tags（排除 selector/urltest）
	ActualNodes []string
}

// NewBuildContext 创建构建上下文
func NewBuildContext(opts *RunOptions) *BuildContext {
	return &BuildContext{
		RunOptions:   opts,
		UserNodeTags: []string{},
		ActualNodes:  []string{},
	}
}
