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

	// 2. 将全局/直连模式转化为更高级别的劫持
	switch m.RouteMode {
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

// applyDefaultFragments 组装默认的开箱即用路由规则
func (m *RouteModule) applyDefaultFragments(opts *option.Options) error {
	var ruleSets []map[string]any
	var rules []map[string]any

	// 协议嗅探 (Sniffing) - 必须放在第一位进行协议和域名嗅探
	rules = append(rules, map[string]any{"action": "sniff"})

	// 片段 1: DNS 流量专门劫持 (在 TUN/Mixed 模式中，由 sing-box 本地解析)
	// 必须在 ip_is_private 之前，否则会把 172.19.0.2:53 等 DNS 包提前放行到 direct，导致 DNS 劫持失效。
	rules = append(rules, map[string]any{"protocol": []string{"dns"}, "action": "hijack-dns"})

	// fake-ip 流量到这里时 metadata.Destination 已经被 matchRule 还原成域名，
	// ip_is_private / ip_cidr（含下面的 geoip-cn）这类基于 IP 的规则只会在
	// metadata.Destination.Addr 无效时退回检查 metadata.DestinationAddresses——
	// 而这个字段只有 resolve 动作才会填充，不加这一步这些规则会直接失效。
	// 对字面量 IP 流量（未走 DNS）是完全无操作、零开销的。
	rules = append(rules, map[string]any{"action": "resolve"})

	// 针对 ali dns 放行（因为在 DNS 模块中配置了国内直接去 ali 解析，避免循环）
	rules = append(rules, map[string]any{"ip_cidr": []string{"223.5.5.5/32", "223.6.6.6/32", "2400:3200::/32"}, "outbound": moduleUtils.TagDirect})

	// 片段 2: 局域网直连
	rules = append(rules, map[string]any{"ip_is_private": true, "outbound": moduleUtils.TagDirect})

	// 片段 3: NTP 直连
	rules = append(rules, map[string]any{"protocol": []string{"ntp"}, "outbound": moduleUtils.TagDirect})

	// IPv6 字面量兜底拦截：DNS 模块已启用 fake-ip，凡是经过域名解析的流量，
	// matchRule 会先把目的地址还原成域名再匹配规则，这条规则根本碰不到；
	// 它只会命中极少数绕过 DNS、直接连字面量 IPv6 地址的流量。用 reject
	// 返回 TCP RST（而不是静默丢包），让这类流量能尽快失败/回退，避免 EPERM 错误。
	rules = append(rules, map[string]any{"ip_version": 6, "action": "reject"})

	// 片段 4: 去广告模块
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-ads",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "anti-ad",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/privacy-protection-tools/anti-ad.github.io/master/docs/anti-ad-sing-box.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-ads", "anti-ad"}, "outbound": moduleUtils.TagBlock})


	// 片段 5.5: 非中国大陆域名强制代理 (防止海外域名被 IP 查表误判走直连)
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-geolocation-!cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-geolocation-!cn"}, "outbound": moduleUtils.TagProxy})

	// 片段 6: Apple 流量直连
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-apple",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-apple.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-apple"}, "outbound": moduleUtils.TagDirect})


	// 片段 7: 国内直连 (CN 路由分流)
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geoip-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
		"download_detour": moduleUtils.TagProxy,
	})

	// 片段 7.5: geosite-cn 未收录的常见国内服务补充名单。
	// geosite-cn 只聚合了 v2fly domain-list-community 的 tld-cn + geolocation-cn 两个分类，
	// 哔哩哔哩、爱奇艺、优酷等站点根本不在这两个分类里（上游单独维护了对应分类，但没被 include
	// 进 cn/geolocation-cn）。这些站点又大量使用动态命名的 CDN 节点域名，域名规则命中不到时，
	// 只能靠下面 geoip-cn 兜底：连接本身仍会正确走 direct，但 DNS 模块里同名域名解析会先落到
	// proxy_dns（见 dns.go），导致每次换 CDN 节点都要多一次跨境代理 DNS 往返，实测造成明显卡顿。
	// 这里按域名显式补上，配合 dns.go 里的同名规则，从根源避免这次多余的跨境解析。
	commonCNServices := []string{"bilibili", "iqiyi", "youku", "sina", "zhihu", "xiaohongshu", "douyin", "kuaishou", "sohu", "kugou", "kuwo", "acfun"}
	directRuleSetTags := []string{"geosite-cn", "geoip-cn"}
	for _, tag := range commonCNServices {
		ruleSetTag := "geosite-" + tag
		ruleSets = append(ruleSets, map[string]any{
			"tag":             ruleSetTag,
			"type":            "remote",
			"format":          "binary",
			"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/" + ruleSetTag + ".srs",
			"download_detour": moduleUtils.TagProxy,
		})
		directRuleSetTags = append(directRuleSetTags, ruleSetTag)
	}
	rules = append(rules, map[string]any{"rule_set": directRuleSetTags, "outbound": moduleUtils.TagDirect})

	// 此时组合成一个整体 map 进行反序列化 (为了兼容 sing-box 的 rule 抽象类型)
	routeMap := map[string]any{
		"rule_set":              ruleSets,
		"rules":                 rules,
		"auto_detect_interface": true,
	}

	data, err := singboxjson.Marshal(routeMap)
	if err != nil {
		return err
	}

	var generatedRoute option.RouteOptions
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, data, &generatedRoute); err != nil {
		return err
	}

	opts.Route.RuleSet = append(opts.Route.RuleSet, generatedRoute.RuleSet...)
	opts.Route.Rules = append(opts.Route.Rules, generatedRoute.Rules...)
	opts.Route.AutoDetectInterface = generatedRoute.AutoDetectInterface

	return nil
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
	}
	return kept
}
