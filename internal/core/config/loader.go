package config

import (
	"context"
	"os"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// LoadOptions 从文件加载 sing-box 配置
// 可用于加载用户的 profile.json 或生成的 raw.json
func LoadOptions(path string) (*option.Options, error) {
	return LoadOptionsWithContext(context.Background(), path)
}

// LoadOptionsWithContext 从文件加载 sing-box 配置（带 context）
func LoadOptionsWithContext(ctx context.Context, path string) (*option.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var opts option.Options
	// 使用 sing-box 的 JSON 解析器，include.Context 确保正确解析 Outbound 类型
	tx := include.Context(ctx)
	if err := singboxjson.UnmarshalContext(tx, content, &opts); err != nil {
		return nil, err
	}

	return &opts, nil
}

// LoadProfile 是 LoadOptions 的别名，用于加载用户配置
// Deprecated: 建议使用 LoadOptions
func LoadProfile(path string) (*option.Options, error) {
	return LoadOptions(path)
}
