package module

import (
	"context"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// RouteModule 路由模块
// 负责组装和构建动态路由协议栈，通过拼装不同的 RouteFragment 来实现灵活的规则扩展
type RouteModule struct {
	RouteMode model.RouteMode
}

func (m *RouteModule) Name() string {
	return "route"
}

func (m *RouteModule) Apply(opts *option.Options, ctx *BuildContext) error {
	if opts.Route == nil {
		opts.Route = &option.RouteOptions{}
	}

	// 1. 如果用户没有自定义 final 出站，设置默认
	if opts.Route.Final == "" {
		opts.Route.Final = moduleUtils.TagProxy
	}

	// 1.5 默认域名解析器：解析"出站节点自身 server 字段的域名"（如订阅节点用域名
	// 而非 IP）。必须指向 local_dns（直连），不能指向 proxy_dns——proxy_dns 的
	// detour 是 proxy selector，若代理节点自己的域名也要经它解析，会形成"先连上
	// 代理才能解析域名、先解析域名才能连上代理"的自引用死循环，在冷启动或
	// urltest 切换到未连接节点时必然卡死。sing-box 1.13.x 下 dns.servers 数量
	// >= 2 时不设置本项只是 deprecated 警告，1.14.0 起会成为硬性报错
	// （domain_resolver missing for domain server address）。
	if opts.Route.DefaultDomainResolver == nil {
		opts.Route.DefaultDomainResolver = &option.DomainResolveOptions{Server: moduleUtils.TagLocalDNS}
	}

	// 2. 将全局/直连模式转化为更高级别的劫持
	switch m.RouteMode {
	case model.RouteModeRuleDirect:
		// 按规则路由，默认直连（白名单模式）：规则链与 rule 一致，仅 final 不同
		opts.Route.Final = moduleUtils.TagDirect
	case model.RouteModeGlobal:
		// 全局代理：覆盖前面的默认 Final
		opts.Route.Final = moduleUtils.TagProxy
		// 但我们需要保留 DNS 和局域网绕过的规则，因此我们仍然应用 default 规则
	case model.RouteModeDirect:
		// 全局直连：所有流量默认直连
		opts.Route.Final = moduleUtils.TagDirect
	}

	// 3. 构建并应用默认扩展拼图 (作为无条件兜底)
	// 我们将生成的保底规则直接追加到用户自定义规则的后面。
	// 这样用户在 profile.json 中配置的分流优先级最高。
	if err := m.applyDefaultFragments(opts); err != nil {
		return err
	}

	// 4. 清空特定模式下的所有非必要规则
	// 全局代理仍保留 sniff，否则新版本 sing-box 不会用探测出的域名覆盖目标地址。
	if m.RouteMode == model.RouteModeGlobal {
		opts.Route.Rules = keepSniffRules(opts.Route.Rules)
	}
	if m.RouteMode == model.RouteModeDirect {
		opts.Route.Rules = nil
	}

	return nil
}

// buildRouteFragment 将 rule_set/rules 的 map 描述通过 sing-box 的 context-aware
// JSON 反序列化，转成带类型的 RuleSet/Rule 切片（action 等多态字段需要 include.Context
// 才能正确解析出具体类型，不能手写结构体字面量）。
func buildRouteFragment(ruleSetSpecs, ruleSpecs []map[string]any) ([]option.RuleSet, []option.Rule, error) {
	data, err := singboxjson.Marshal(map[string]any{
		"rule_set": ruleSetSpecs,
		"rules":    ruleSpecs,
	})
	if err != nil {
		return nil, nil, err
	}

	var parsed option.RouteOptions
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, data, &parsed); err != nil {
		return nil, nil, err
	}
	return parsed.RuleSet, parsed.Rules, nil
}

// applyDefaultFragments 组装默认的开箱即用路由规则
func (m *RouteModule) applyDefaultFragments(opts *option.Options) error {
	// 协议嗅探 (Sniffing) 和 DNS 劫持必须排在包括用户 profile.json 规则在内的所有域名
	// 规则之前才有意义：sniff 补全 metadata.Domain，hijack-dns 必须在 ip_is_private 之前
	// 拦下 DNS 包，否则会被提前放行到 direct 导致 DNS 劫持失效。这两条单独生成、单独插入
	// 用户规则之前，不与下面的保底规则混在一份列表里切片，避免下标错位。
	leadingRuleSpecs := []map[string]any{
		{"action": "sniff"},
		{"protocol": []string{"dns"}, "action": "hijack-dns"},
	}

	var ruleSets []map[string]any
	var rules []map[string]any

	// 针对 ali dns 放行（因为在 DNS 模块中配置了国内直接去 ali 解析，避免循环）
	// 字面量 IP 匹配 metadata.Destination.Addr，不依赖 resolve，放在哪里都一样。
	rules = append(rules, map[string]any{"ip_cidr": []string{"223.5.5.5/32", "223.6.6.6/32", "2400:3200::/32"}, "outbound": moduleUtils.TagDirect})

	// 片段 2: NTP 直连
	rules = append(rules, map[string]any{"protocol": []string{"ntp"}, "outbound": moduleUtils.TagDirect})

	// 片段 2.1: ICMP 直连（避免代理出站不支持 ICMP 导致报错）
	rules = append(rules, map[string]any{"protocol": []string{"icmp"}, "outbound": moduleUtils.TagDirect})

	// ============ 片段 3: 域名分流规则（根据路由模式选择不同规则集）============
	// rule-direct（白名单模式）：final=direct，仅被 GFW 封锁的域名走代理。
	// 不需要 geosite-cn/apple -> direct 规则，因为默认出站就是 direct。
	ruleSets = append(ruleSets, metaGeositeRuleSet("gfw"), metaGeositeRuleSet("google"), metaGeositeRuleSet("github"))
	rules = append(rules, map[string]any{
		"rule_set": []string{"geosite-gfw", "geosite-google", "geosite-github"},
		"outbound": moduleUtils.TagProxy,
	})

	// 国内/苹果直连白名单 -> 直连。
	ruleSets = append(ruleSets, metaGeositeRuleSet("apple"), metaGeositeRuleSet("cn"))
	rules = append(rules, map[string]any{
		"rule_set": []string{"geosite-apple", "geosite-cn"},
		"outbound": moduleUtils.TagDirect,
	})

	// ============ 片段 5: 去广告（放在白名单之后，避免误伤上面已确认的服务）============
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-ads",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/category-ads-all.srs",
		"download_detour": moduleUtils.TagProxy,
	})

	// 用 reject 而非 block outbound：block 在 macOS TUN 下会以 EPERM 的形式
	// 抛出给客户端（而不是干净的 TCP RST/ICMP 不可达），部分 App 对 EPERM 的
	// 处理比对正常连接失败更差，实测日志中出现上千次 EPERM。这里复用 IPv6
	// 字面量拦截规则（片段 3 末尾）已经验证过的做法，保持处理方式一致。
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-ads"}, "action": "reject"})

	// ============ 片段 6: resolve + IP 兜底 ============
	// 只有上面所有域名规则集都没命中的连接才会走到这里。resolve 曾经放在
	// 片段 1 最前面，导致每一条连接（包括本来靠域名就能零延迟分流的
	// google/github/apple/cn 流量）都要先付出一次真实 DNS 往返延迟——实测日志
	// 中 clients2.google.com 等域名的建连曾因此被拖慢 700ms~1.7s。挪到这里后，
	// 只有域名匹配不到的流量（未知域名、局域网域名等）才需要这次真实解析。
	//
	// fake-ip 流量到这里时 metadata.Destination 已经被 matchRule 还原成域名，
	// ip_is_private / ip_cidr（含下面的 geoip-google）这类基于 IP 的规则只会在
	// metadata.Destination.Addr 无效时退回检查 metadata.DestinationAddresses——
	// 而这个字段只有 resolve 动作才会填充，不加这一步这些规则会直接失效。
	// 对字面量 IP 流量（未走 DNS）是完全无操作、零开销的。
	rules = append(rules, map[string]any{"action": "resolve"})

	// 局域网域名 (nas.local、router.lan 等) 不在上面任何 geosite 白名单里，靠这里
	// resolve 出的真实私网 IP 兜底直连；不会因为 resolve 挪到后面而失效。
	rules = append(rules, map[string]any{"ip_is_private": true, "outbound": moduleUtils.TagDirect})

	// IPv6 字面量兜底拦截：DNS 模块已启用 fake-ip，凡是经过域名解析的流量，
	// matchRule 会先把目的地址还原成域名再匹配规则，这条规则根本碰不到；
	// 它只会命中极少数绕过 DNS、直接连字面量 IPv6 地址的流量。用 reject
	// 返回 TCP RST（而不是静默丢包），让这类流量能尽快失败/回退，避免 EPERM 错误。
	rules = append(rules, map[string]any{"ip_version": 6, "action": "reject"})

	// ============ 片段 6.5: IP 兜底规则（根据路由模式选择）============
	ruleSets = append(ruleSets, metaGeoipRuleSet("telegram"),
		metaGeoipRuleSet("google"),
		metaGeoipRuleSet("netflix"),
		metaGeoipRuleSet("facebook"),
		metaGeoipRuleSet("twitter"))
	rules = append(rules, map[string]any{"rule_set": []string{"geoip-telegram", "geoip-google", "geoip-netflix", "geoip-facebook", "geoip-twitter"}, "outbound": moduleUtils.TagProxy})

	// geoip-cn 兜底：sniff 顺序修复 + dns.reverse_mapping 上线后，只有真正拿不到域名的
	// 连接（未走 DNS 的裸 IP、UDP 首包等）才会落到这里；直连一个国内 IP 段判断出错，
	ruleSets = append(ruleSets, metaGeoipRuleSet("cn"))
	rules = append(rules, map[string]any{"rule_set": []string{"geoip-cn"}, "outbound": moduleUtils.TagDirect})

	// ============ 片段 7: 默认出站 ============
	// 未命中以上任何规则的流量，落到 RouteModule.Apply 里设置的 final。

	_, leadingRules, err := buildRouteFragment(nil, leadingRuleSpecs)
	if err != nil {
		return err
	}
	generatedRuleSets, trailingRules, err := buildRouteFragment(ruleSets, rules)
	if err != nil {
		return err
	}

	// leading（sniff/hijack-dns）和 trailing（保底规则）是两次独立生成的结果，
	// 用户规则原样插在两者中间：不再依赖对同一份列表切片取下标，以后往 trailing
	// 开头加规则也不会误伤 leading 的内容。
	opts.Route.RuleSet = append(opts.Route.RuleSet, generatedRuleSets...)
	opts.Route.Rules = append(append(append([]option.Rule{}, leadingRules...), opts.Route.Rules...), trailingRules...)
	opts.Route.AutoDetectInterface = true

	return nil
}

// metaGeositeRuleSet 引用 meta-rules-dat 按公司/产品维护的域名规则集
func metaGeositeRuleSet(name string) map[string]any {
	return map[string]any{
		"tag":             "geosite-" + name,
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/" + name + ".srs",
		"download_detour": moduleUtils.TagProxy,
	}
}

// metaGeoipRuleSet 引用 meta-rules-dat 按公司/产品维护的 IP 规则集
func metaGeoipRuleSet(name string) map[string]any {
	return map[string]any{
		"tag":             "geoip-" + name,
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/" + name + ".srs",
		"download_detour": moduleUtils.TagProxy,
	}
}

func keepSniffRules(rules []option.Rule) []option.Rule {
	kept := rules[:0]
	for _, rule := range rules {
		raw, err := singboxjson.Marshal(rule)
		if err != nil {
			continue
		}
		var rm map[string]any
		if err := singboxjson.Unmarshal(raw, &rm); err != nil {
			continue
		}
		action, _ := rm["action"].(string)
		if action == "sniff" || action == "hijack-dns" {
			kept = append(kept, rule)
			continue
		}
		// Preserve AliDNS direct-bypass so bootstrap DoH to 223.5.5.5 stays direct.
		if _, hasCIDR := rm["ip_cidr"]; hasCIDR {
			kept = append(kept, rule)
		}
		// 保留局域网私网直连规则，防止全局代理模式下局域网设备断连
		if isPrivate, _ := rm["ip_is_private"].(bool); isPrivate {
			kept = append(kept, rule)
		}
	}
	return kept
}
