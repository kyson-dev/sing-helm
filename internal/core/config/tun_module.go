package config

import (
	"context"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// TUNModule TUN 入站模块
type TUNModule struct {
	MTU   int
	Stack string
}

func (m *TUNModule) Name() string {
	return "tun"
}

func (m *TUNModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 默认值
	mtu := m.MTU
	if mtu == 0 {
		mtu = 9000
	}

	stack := m.Stack
	if stack == "" {
		stack = "mixed" // mixed 兼顾性能和兼容性
	}

	// 创建 TUN 入站配置
	tunInbound := option.Inbound{}
	tunMap := map[string]any{
		"type":                       "tun",
		"tag":                        "tun-in",
		"mtu":                        mtu,
		"auto_route":                 true,
		"strict_route":               true,
		"stack":                      stack,
		"inet4_address":              "172.19.0.1/30",
		"inet6_address":              "fd00::1/126",
		"sniff":                      true,
		"sniff_override_destination": false,
	}
	applyMapToInbound(&tunInbound, tunMap)

	// 添加到配置
	opts.Inbounds = append(opts.Inbounds, tunInbound)

	return nil
}

// TUNDNSModule TUN DNS 模块
// TUN 模式需要特殊的 DNS 配置
type TUNDNSModule struct{}

func (m *TUNDNSModule) Name() string {
	return "tun_dns"
}

func (m *TUNDNSModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 使用 map 方式创建 DNS 配置
	// local_dns 不需要 detour，默认就是直连
	dnsMap := map[string]any{
		"servers": []map[string]any{
			{
				"tag":              "proxy_dns",
				"address":          "https://dns.google/dns-query",
				"address_resolver": "local_dns",
				"detour":           "proxy",
			},
			{
				"tag":     "local_dns",
				"address": "223.5.5.5",
				// 不设置 detour，默认直连
			},
			{
				"tag":     "block_dns",
				"address": "rcode://success",
			},
		},
		"rules": []map[string]any{
			{
				"outbound": "any",
				"server":   "proxy_dns",
			},
		},
		"strategy":          "prefer_ipv4",
		"disable_cache":     false,
		"disable_expire":    false,
		"independent_cache": false,
	}

	data, err := singboxjson.Marshal(dnsMap)
	if err != nil {
		return err
	}

	var dnsOpts option.DNSOptions
	// 必须使用 include.Context 来正确解析 DNS 类型
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, data, &dnsOpts); err != nil {
		return err
	}

	opts.DNS = &dnsOpts
	return nil
}
