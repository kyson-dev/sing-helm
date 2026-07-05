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
				"tag":    moduleUtils.TagLocalDNS,
				"type":   "https",
				"server": "223.5.5.5", // IP 直连，无需 domain_resolver，无明文 UDP 引导查询
			},
			{
				"tag":    moduleUtils.TagProxyDNS,
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
				"rule_set": []string{"geosite-ads"},
				"action":   "reject",
			},
			// 现代浏览器发起 A/AAAA 查询时会并发发出 HTTPS/SVCB (type 65/64) 查询做
			// ALPN/ECH 协商。这类查询不在下面的 A/AAAA 过滤范围内，会穿透到
			// local_dns/proxy_dns 走真实网络解析；由于 A/AAAA 已经立刻拿到 fake-ip，
			// 浏览器却要等这次真实查询返回才继续握手，实测会造成几百毫秒到数秒的
			// 卡顿（网络波动时甚至 10s 超时）。直接拒绝，浏览器会退回用 fake-ip 建连。
			{
				"query_type": []string{"HTTPS", "SVCB"},
				"action":     "reject",
			},
			// 必须放在 cn/apple 分流规则之前：命中的查询在“面向客户端”的
			// 那一次解析中会直接拿到 fake-ip；而 sing-box 内部为出站拨号做的
			// 第二次解析 (allowFakeIP=false) 会自动跳过这条规则，落到下面的
			// local_dns/proxy_dns 分流规则上，拿到真实地址——两次解析互不冲突，
			// 无需额外配置。
			{
				"query_type": []string{"A", "AAAA"},
				"server":     "dns_fakeip",
			},
			// PTR（反向解析）及局域网私有域名：不在 geosite-cn/apple 名单里，若落到
			// 下面 final: proxy_dns，会拿私网地址/mDNS 域名去问公网 DNS，必然失败，
			// 实测每次都要等满 10s 超时——常见于 macOS/iOS 的 Bonjour 局域网设备发现
			// (AirDrop、打印机、NAS 等)。这里强制走本地直连 DNS。
			{
				"query_type": []string{"PTR"},
				"action":     "route",
				"server":     moduleUtils.TagLocalDNS,
			},
			{
				"domain_suffix": []string{".local", ".lan", ".home.arpa", "localhost"},
				"action":        "route",
				"server":        moduleUtils.TagLocalDNS,
			},
			// DNS解析模块不应该设置ip集，它本来就是输入域名输出ip的。
			// tag 与 route.go 的白名单规则共用同一份 rule_set 定义（meta-rules-dat 的
			// geosite-cn/geosite-apple），命中的域名交给 AliDoH 解析；未命中的一律落到
			// 下面的 final: proxy_dns，走代理侧 DoH 解析（不再需要额外的 !cn 规则，效果和
			// final 完全重复）。
			{
				"rule_set": []string{"geosite-cn", "geosite-apple"},
				"action":   "route",
				"server":   moduleUtils.TagLocalDNS,
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
