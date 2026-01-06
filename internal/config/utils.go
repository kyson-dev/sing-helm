package config

import (
	"context"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// LoadOptionsWithContext 从配置文件加载 sing-box 配置
func LoadOptionsWithContext(ctx context.Context, configPath string) (*option.Options, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var opts option.Options
	includeCtx := include.Context(ctx)
	if err := singboxjson.UnmarshalContext(includeCtx, data, &opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &opts, nil
}

// applyMapToOutbound 将 map 配置应用到 Outbound 结构体
func applyMapToOutbound(out *option.Outbound, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	// 使用 context 确保类型注册
	ctx := include.Context(context.Background())
	return singboxjson.UnmarshalContext(ctx, data, out)
}

// applyMapToInbound 将 map 配置应用到 Inbound 结构体
func applyMapToInbound(in *option.Inbound, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	ctx := include.Context(context.Background())
	return singboxjson.UnmarshalContext(ctx, data, in)
}
