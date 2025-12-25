package config

import (
	"context"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

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
