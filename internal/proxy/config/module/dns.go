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
			// fake-ip：客户端拿到的始终是这个池子里的占位地址，真实解析推迟到
			// 出站拨号时才发生（见下方 rules 注释），从而让 sing-box 自己的
			// dialer 对域名做真正的 Happy Eyeballs (v4/v6 并行竞速)，
			// 而不是由客户端 OS 单方面决定连哪个地址族。
			{
				"tag":         "dns_fakeip",
				"type":        "fakeip",
				"inet4_range": "198.18.0.0/15",
				"inet6_range": "fc00::/18",
			},
		},
		"rules": []map[string]any{
			{
				"rule_set": []string{"geosite-ads", "anti-ad"},
				"action":   "reject",
			},
			// 必须放在 cn/apple/!cn 分流规则之前：命中的查询在“面向客户端”的
			// 那一次解析中会直接拿到 fake-ip；而 sing-box 内部为出站拨号做的
			// 第二次解析 (allowFakeIP=false) 会自动跳过这条规则，落到下面的
			// local_dns/proxy_dns 分流规则上，拿到真实地址——两次解析互不冲突，
			// 无需额外配置。
			{
				"query_type": []string{"A", "AAAA"},
				"server":     "dns_fakeip",
			},
			// 非中国大陆域名强制代理 (防止海外域名被 IP 查表误判走直连)
			{
				"rule_set": []string{"geosite-geolocation-!cn"},
				"action":   "route",
				"server":   "proxy_dns",
			},
			// DNS解析模块不应该设置ip集，它本来就是输入域名输出ip的。
			// geosite-cn 未收录的常见国内服务（B站/爱奇艺/优酷等，见 route.go 里的详细说明）
			// 显式补充到 local_dns，避免这些域名解析退化到 final: proxy_dns 走跨境代理，
			// final 兜底策略本身保持不变。
			{
				"rule_set": []string{
					"geosite-cn", "geosite-apple",
					"geosite-bilibili", "geosite-iqiyi", "geosite-youku", "geosite-sina",
					"geosite-zhihu", "geosite-xiaohongshu", "geosite-douyin", "geosite-kuaishou",
					"geosite-sohu", "geosite-kugou", "geosite-kuwo", "geosite-acfun",
				},
				"action": "route",
				"server": "local_dns",
			},
		},
		"final": "proxy_dns",
		// prefer_ipv4 只排序不过滤 (v4 优先，v6 仍保留)：既让 fake-ip 能正常
		// 生成 AAAA 占位地址，也让出站拨号阶段的真实解析同时拿到 v4/v6，
		// 交给 dialer 的 Happy Eyeballs 去竞速，而不是在 DNS 层就切断 v6。
		"strategy": "prefer_ipv4",
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
