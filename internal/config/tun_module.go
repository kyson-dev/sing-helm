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
		"type":         "tun",
		"tag":          "tun-in",
		"mtu":          mtu,
		"auto_route":   true,
		"strict_route": true,
		//"stack":                      stack,
		"address": []string{"172.19.0.1/30"},
		//"inet6_address":              "fd00::1/126",
		"sniff":                      true,
		"sniff_override_destination": true,
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
				"tag":             "local_dns",
				"type":            "https",
				"server":          "dns.alidns.com",
				"domain_resolver": "resolver_dns",
			},
			{
				"tag":             "proxy_dns",
				"type":            "https",
				"server":          "dns.google",
				"domain_resolver": "resolver_dns",
				"detour":          "proxy",
			},
			{
				"tag":    "resolver_dns",
				"type":   "udp",
				"server": "223.5.5.5",
			},
		},
		"rules": []map[string]any{
			{
				"rule_set": []string{"geosite-ads", "anti-ad"},
				"action":   "reject",
			},
			{
				"domain_suffix": []string{"wise.com", "schwab.com", "interactivebrokers.com", "cloudflare.com"},
				"action":        "route",
				"server":        "local_dns",
			},
			{
				"rule_set": []string{"geosite-cn", "geoip-cn"},
				"action":   "route",
				"server":   "local_dns",
			},
		},
		"final":    "proxy_dns",
		"strategy": "ipv4_only",
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
