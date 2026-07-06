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

	// 针对 ali dns 放行（因为在 DNS 模块中配置了国内直接去 ali 解析，避免循环）
	// 字面量 IP 匹配 metadata.Destination.Addr，不依赖 resolve，放在哪里都一样。
	rules = append(rules, map[string]any{"ip_cidr": []string{"223.5.5.5/32", "223.6.6.6/32", "2400:3200::/32"}, "outbound": moduleUtils.TagDirect})

	// 片段 2: NTP 直连
	rules = append(rules, map[string]any{"protocol": []string{"ntp"}, "outbound": moduleUtils.TagDirect})

	// 片段 2.1: ICMP 直连（避免代理出站不支持 ICMP 导致报错）
	rules = append(rules, map[string]any{"protocol": []string{"icmp"}, "outbound": moduleUtils.TagDirect})


	// ============ 片段 3: 白名单（来自 meta-rules-dat，优先于广告拦截）============
	// 不再使用 SagerNet 的 geosite-cn/geoip-cn/geolocation-!cn：经实测确认历史上
	// www.gstatic.com 被误直连，命中的是 SagerNet geosite-cn 这个域名规则集
	// （geoip-cn 对日志中出现的所有 Google/GitHub IP 均未命中，是无辜的）。改用
	// meta-rules-dat（mihomo 团队维护，原生发布 sing-box .srs，按公司/产品拆分，
	// 已实测 www.gstatic.com/google.com/github.com 在其 cn.srs 中均为干净）。
	// 以后新增名单，只应在这里逐条添加具体规则集，不允许重新引入任何"大类"。

	// 海外强制代理白名单 -> 代理。放在最前面，防止被下面的国内白名单或广告规则
	// 集意外误伤（历史事故：gstatic.com 被误直连、github.com 被广告规则集误判
	// block，均导致本该走代理的连接白白等满超时或 EPERM）。
	// 这里只用域名规则集，故意不带 geoip-google：域名匹配不需要真实 IP，可以在
	// 下面的 resolve 之前就直接命中，让已知域名零延迟分流；geoip-google 的 IP
	// 兜底放到 resolve 之后单独处理（见片段 6），避免为了它把 resolve 提前到
	// 所有连接都要付出一次真实 DNS 往返的位置。
	ruleSets = append(ruleSets, metaGeositeRuleSet("google"), metaGeositeRuleSet("github"))
	rules = append(rules, map[string]any{
		"rule_set": []string{"geosite-google", "geosite-github"},
		"outbound": moduleUtils.TagProxy,
	})

	// 国内/苹果直连白名单 -> 直连。只用域名规则集 cn（不引入对应的 geoip-cn）：
	// fake-ip 模式下规则匹配前已把目的地址还原成域名，IP 库能多覆盖的场景很小
	// （只有绕过 DNS 直拨字面量 IP 的流量），却要背上跨境 CDN/Anycast IP 误判的
	// 风险，不值得。
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

	// geoip-google 兜底：极少数不在 geosite-google 名单里的 Google 域名，或绕过
	// DNS 直接拨字面量 IP 的流量，只要目的地址落在 Google IP 段也强制代理——
	// 误判代价只是多走一次代理，没有实际损失。
	ruleSets = append(ruleSets, metaGeoipRuleSet("google"))
	rules = append(rules, map[string]any{"rule_set": []string{"geoip-google"}, "outbound": moduleUtils.TagProxy})

	// ============ 片段 7: 默认代理 ============
	// 未命中以上任何规则的流量，落到 RouteModule.Apply 里设置的 final: proxy。
	// 不再有 geoip-cn 兜底直连：这是整套规则里误判率最高的信号来源，删除之后，
	// 任何没被白名单显式认领的流量默认代理，代价只是慢一点，不会再出现
	// "误判走直连、实际连不通、白等 5 秒超时" 这类硬失败。

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
	}
	return kept
}
