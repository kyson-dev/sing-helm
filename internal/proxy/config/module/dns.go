package module

import (
	"context"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// DNSModule TUN DNS 模块
// TUN 模式需要特殊的 DNS 配置
type DNSModule struct{}

func (m *DNSModule) Name() string {
	return "dns"
}

func (m *DNSModule) Apply(opts *option.Options, ctx *BuildContext) error {
	if opts.DNS == nil {
		opts.DNS = &option.DNSOptions{}
	}   

	// 使用 IP 地址直接连接 DoH 服务器，避免引导解析（resolver_dns）发出明文 UDP DNS 查询。
	// 223.5.5.5 = AliDNS DoH IP，8.8.8.8 = Google DNS DoH IP，两者均支持 IP 直连。
	dnsMap := map[string]any{
		"servers": []map[string]any{
			{
				"tag":    "local_dns",
				"type":   "https",
				"server": "223.5.5.5", // IP 直连，无需 domain_resolver，无明文 UDP 引导查询
			},
			{
				"tag":    "proxy_dns",
				"type":   "https",
				"server": "8.8.8.8", // IP 直连，无需 domain_resolver
				"detour": moduleUtils.TagProxy,
			},
		},
		"rules": []map[string]any{
			{
				"rule_set": []string{"geosite-ads", "anti-ad"},
				"action":   "reject",
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

	var defaultDnsOpts option.DNSOptions
	// 必须使用 include.Context 来正确解析 DNS 类型
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, data, &defaultDnsOpts); err != nil {
		return err
	}

	// 1. 合并 Servers: 系统硬编码优先，同 tag 的用户定义会被丢弃；用户其它 server 保留。
	systemServerTags := make(map[string]bool, len(defaultDnsOpts.Servers))
	mergedServers := make([]option.DNSServerOptions, 0, len(defaultDnsOpts.Servers)+len(opts.DNS.Servers))
	for _, ds := range defaultDnsOpts.Servers {
		systemServerTags[ds.Tag] = true
		mergedServers = append(mergedServers, ds)
	}
	for _, us := range opts.DNS.Servers {
		if us.Tag != "" && systemServerTags[us.Tag] {
			continue
		}
		mergedServers = append(mergedServers, us)
	}
	opts.DNS.Servers = mergedServers

	// 2. 合并 Rules: 用户规则前置，系统规则后置。
	userRules := append([]option.DNSRule(nil), opts.DNS.Rules...)
	opts.DNS.Rules = append(userRules, defaultDnsOpts.Rules...)

	// 3. 基础设置: 强制使用系统硬编码，不能被用户覆盖。
	opts.DNS.Final = defaultDnsOpts.Final
	opts.DNS.Strategy = defaultDnsOpts.Strategy
	return nil
}
