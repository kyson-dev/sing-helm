package module

import (
	"context"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// ApplyMapToOutbound 将 map 配置应用到 Outbound 结构体
func ApplyMapToOutbound(out *option.Outbound, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	// 使用 context 确保类型注册
	ctx := include.Context(context.Background())
	return singboxjson.UnmarshalContext(ctx, data, out)
}

// ApplyMapToInbound 将 map 配置应用到 Inbound 结构体
func ApplyMapToInbound(in *option.Inbound, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	ctx := include.Context(context.Background())
	return singboxjson.UnmarshalContext(ctx, data, in)
}
