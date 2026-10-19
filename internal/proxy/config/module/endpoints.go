package module

import (
	"context"
	"fmt"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// EndpointModule 负责 endpoints 段。目前只有 Tailscale endpoint：让本机通过
// sing-box 内置的 Tailscale 加入 tailnet（不依赖 Tailscale 客户端，用户态网络，
// 不创建网卡）。对应的 DNS 服务器和路由规则分别由 DNSModule / RouteModule 生成。
//
// 使用它的二进制必须带 with_tailscale 编译标签（见 Makefile）。
type EndpointModule struct {
	// StateDir 是 Tailscale 节点身份（含私钥）的保存目录，空值表示由 sing-box 选默认位置。
	StateDir string
}

func (m *EndpointModule) Name() string {
	return "endpoints"
}

func (m *EndpointModule) Apply(opts *option.Options, ctx *BuildContext) error {
	for _, ep := range opts.Endpoints {
		if ep.Tag == moduleUtils.TagTailscale {
			return fmt.Errorf("endpoint %q is already defined in profile.json; remove it from profile.json (sing-helm now adds it itself)", ep.Tag)
		}
	}

	endpoint := map[string]any{
		"type": "tailscale",
		"tag":  moduleUtils.TagTailscale,
	}
	if m.StateDir != "" {
		endpoint["state_directory"] = m.StateDir
	}
	data, err := singboxjson.Marshal(map[string]any{"endpoints": []map[string]any{endpoint}})
	if err != nil {
		return err
	}
	// 必须使用 include.Context 才能解析 endpoint 的多态类型。
	var parsed option.Options
	if err := singboxjson.UnmarshalContext(include.Context(context.Background()), data, &parsed); err != nil {
		return err
	}
	opts.Endpoints = append(opts.Endpoints, parsed.Endpoints...)
	return nil
}
