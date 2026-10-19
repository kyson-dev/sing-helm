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

// 以下三份列表是 rule 模式下 direct 分流的规则集目录，均取自 meta-rules-dat
// （sing 分支），是 DNS 模块（dns.go 的 local_dns 分流规则）和路由模块共同的
// 唯一数据源，两边同时读取，不会出现"路由判断直连、DNS 却还查代理"的不同步。
// 要新增一个直连分类，只需在对应列表里加一个规则集名字，applyDefaultFragments
// 和 DNSModule.Apply 会自动生成对应的 rule_set + rule。
//
// 没有单独的"proxy 域名白名单"：凡是没有命中下面任一直连规则集的域名，天然落到
// RouteModule.Apply 里设置的 Final（rule 模式下恒为 proxy），不需要再显式列一份
// GFW 封锁域名清单去手动分流到代理。这也避免了曾经的一个坑——geosite-gfw 这类
// 社区维护的"大陆需代理"名单里混有 futunn.com 等域名，如果显式声明了
// geosite-gfw -> proxy 规则、又排在 futu 白名单前面，会抢先命中导致明明加了
// 直连白名单却还在走代理。
var (
	// directDomainRuleSets 应当直连的域名白名单：国内站点、苹果服务，以及一批
	// "服务器本就在国内/香港、走代理反而更慢"的金融类 App —— 富途 (futu)
	// 及证券/银行域名合集。这些不在 geosite-cn 里，是这类 App 走代理体验变差
	// 的常见原因。
	directDomainRuleSets = []string{
		"apple", "cn", "futu", "category-securities-cn", "category-bank-cn", "category-finance@cn",
	}

	// proxyIPRuleSets / directIPRuleSets 是域名规则都没命中时（裸 IP 连接、UDP
	// 首包等）的 IP 段兜底。
	proxyIPRuleSets  = []string{"telegram", "google", "netflix", "facebook", "twitter"}
	directIPRuleSets = []string{"cn"}

	// directASNRuleSets 是按 AS 号收录的 IP 段兜底，用于覆盖那些客户端直接连字面量
	// IP（不带域名/SNI，域名规则打不到）、且服务器不在国内 IP 段（geoip-cn 也打不到）
	// 的场景。AS132203 是腾讯云的 ASN——富途 App 的行情/长连接通道实测走的就是这个
	// ASN 下的 IP（124.156.x/119.28.x/129.226.x/170.106.x/43.128.x/43.153.x 等），
	// 域名和国内 IP 段规则都覆盖不到，此前会误落到 final=proxy。
	directASNRuleSets = []string{"AS132203"}
)

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
	// 直连模式清空所有规则，但要保留 tailnet 规则：否则 100.x 的连接会从本机网卡
	// 直接发出去，保险库就连不上了。
	if m.RouteMode == model.RouteModeDirect {
		opts.Route.Rules = keepTailscaleRules(opts.Route.Rules)
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
	// tailnet 地址段和 *.ts.net 走 tailscale endpoint（由 EndpointModule 添加）。
	// 排在用户规则之前，避免被后面的直连/代理规则抢先命中。
	leadingRuleSpecs = append(leadingRuleSpecs,
		map[string]any{"ip_cidr": []string{"100.64.0.0/10"}, "outbound": moduleUtils.TagTailscale},
		map[string]any{"domain_suffix": []string{"ts.net"}, "outbound": moduleUtils.TagTailscale},
	)

	var ruleSets []map[string]any
	var rules []map[string]any

	// 针对 ali dns 放行（因为在 DNS 模块中配置了国内直接去 ali 解析，避免循环）
	// 字面量 IP 匹配 metadata.Destination.Addr，不依赖 resolve，放在哪里都一样。
	// 注意：只放行 dns.go 中 local_dns 实际使用的 223.5.5.5；不保留未使用的
	// 备用地址。也不要加 IPv6 版本的整段放行——曾经加过 "2400:3200::/32"，
	// 但 v6 的 /32 覆盖的是阿里云整个分配网段（不像 v4 /32 是精确单地址），
	// 会把任何部署在阿里云上、恰好走字面量 v6 地址连接的服务（如 Soul 的 CDN）
	// 一并误判为 direct，绕过后面 ip_version:6 reject 等所有判断。
	rules = append(rules, map[string]any{"ip_cidr": []string{"223.5.5.5/32"}, "outbound": moduleUtils.TagDirect})

	// 片段 1.5: Apple 自有 IP 段 (17.0.0.0/8) 直连兜底。
	// 实测客户端连接日志里能看到不少目的地是 17.248.163.x:443 的裸 UDP/QUIC
	// 连接（无法确认具体是 Apple 的哪个服务），既没有可嗅探的域名/SNI 命中
	// geosite-apple，落到片段 6.5 的 IP 兜底时 proxyIPRuleSets/geoip-cn 也都
	// 覆盖不到（目标是美国的 Apple IP，不在国内 IP 段），于是这类连接会落到
	// Final=proxy，白白把本该直连的 Apple 流量绕去美国出口，没有任何好处。
	// meta-rules-dat 没有提供 geoip-apple 规则集可引用，17.0.0.0/8 自 1992 年起
	// 整体分配给 Apple、长期稳定，直接硬编码 CIDR 兜底。
	// 仅覆盖 IPv4：Apple 的 IPv6 段（如 2620:149::/32）存在同样的裸 IP 漏洞，
	// 但目前抓到的连接样本都是 v4，暂不处理，等有 v6 命中的证据再补。
	rules = append(rules, map[string]any{"ip_cidr": []string{"17.0.0.0/8"}, "outbound": moduleUtils.TagDirect})

	// 片段 2: NTP 直连
	rules = append(rules, map[string]any{"protocol": []string{"ntp"}, "outbound": moduleUtils.TagDirect})

	// 片段 2.1: ICMP 直连（ICMP 是网络类型，不是可嗅探的应用层协议）。
	rules = append(rules, map[string]any{"network": []string{"icmp"}, "outbound": moduleUtils.TagDirect})

	// ============ 片段 3: 域名分流规则 ============
	// 国内/苹果/金融类直连白名单 -> 直连；其余域名（含 GFW 封锁站点）不再显式列
	// 规则，直接落到 Final=proxy。见上面 directDomainRuleSets 的注释。
	ruleSets = append(ruleSets, geositeRuleSets(directDomainRuleSets)...)
	rules = append(rules, map[string]any{
		"rule_set": ruleSetTags("geosite", directDomainRuleSets),
		"outbound": moduleUtils.TagDirect,
	})

	// ============ 片段 5: 去广告（放在白名单之后，避免误伤上面已确认的服务）============
	ruleSets = append(ruleSets, map[string]any{
		"tag":         "geosite-ads",
		"type":        "remote",
		"format":      "binary",
		"url":         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/category-ads-all.srs",
		"http_client": map[string]any{"detour": moduleUtils.TagProxy},
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

	// 曾经这里有条无条件的 IPv6 字面量拦截 reject 规则，前提是"字面量 v6 连接
	// 不快速失败会傻等 OS 默认 30-75s 超时"。删除原因：手机实测日志显示，本机
	// 网络真没有 v6 路由时，direct 拨号会立刻收到 "no route to host"（操作系统
	// 层面瞬间失败，不是超时挂起）；本机有 v6 时应该让它正常走通。这条规则唯一
	// 的效果其实是无差别拒绝所有字面量 v6，连本来能成功的连接（比如蜂窝网络）
	// 也一并误杀。我们没有能力在配置阶段判断本机/代理节点是否有 v6，索性不再
	// 猜——交给 direct/proxy 各自的真实拨号结果自然决定成败。若以后重新出现
	// 挂起（而非瞬间失败）的情况，说明这个假设不成立，需要重新引入某种保护。
	// 见 docs/design/fake-ip-routing.md。

	// ============ 片段 6.5: IP 兜底规则 ============
	ruleSets = append(ruleSets, geoipRuleSets(proxyIPRuleSets)...)
	rules = append(rules, map[string]any{"rule_set": ruleSetTags("geoip", proxyIPRuleSets), "outbound": moduleUtils.TagProxy})

	// geoip-cn 兜底：sniff 顺序修复 + dns.reverse_mapping 上线后，只有真正拿不到域名的
	// 连接（未走 DNS 的裸 IP、UDP 首包等）才会落到这里；直连一个国内 IP 段判断出错，
	ruleSets = append(ruleSets, geoipRuleSets(directIPRuleSets)...)
	rules = append(rules, map[string]any{"rule_set": ruleSetTags("geoip", directIPRuleSets), "outbound": moduleUtils.TagDirect})

	// 按 ASN 兜底：见 directASNRuleSets 注释，覆盖裸 IP 连接、且不在国内 IP 段
	// 但实际应直连的服务器（如托管在腾讯云香港/新加坡节点的国内 App 长连接通道）。
	ruleSets = append(ruleSets, asnRuleSets(directASNRuleSets)...)
	rules = append(rules, map[string]any{"rule_set": ruleSetTags("asn", directASNRuleSets), "outbound": moduleUtils.TagDirect})

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
		"tag":         "geosite-" + name,
		"type":        "remote",
		"format":      "binary",
		"url":         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/" + name + ".srs",
		"http_client": map[string]any{"detour": moduleUtils.TagProxy},
	}
}

// metaGeoipRuleSet 引用 meta-rules-dat 按公司/产品维护的 IP 规则集
func metaGeoipRuleSet(name string) map[string]any {
	return map[string]any{
		"tag":         "geoip-" + name,
		"type":        "remote",
		"format":      "binary",
		"url":         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/" + name + ".srs",
		"http_client": map[string]any{"detour": moduleUtils.TagProxy},
	}
}

// metaAsnRuleSet 引用 meta-rules-dat 按 AS 号维护的 IP 规则集
func metaAsnRuleSet(name string) map[string]any {
	return map[string]any{
		"tag":         "asn-" + name,
		"type":        "remote",
		"format":      "binary",
		"url":         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/asn/" + name + ".srs",
		"http_client": map[string]any{"detour": moduleUtils.TagProxy},
	}
}

// geositeRuleSets/geoipRuleSets/asnRuleSets 把规则集名字列表批量展开成 rule_set 声明。
func geositeRuleSets(names []string) []map[string]any {
	sets := make([]map[string]any, 0, len(names))
	for _, name := range names {
		sets = append(sets, metaGeositeRuleSet(name))
	}
	return sets
}

func geoipRuleSets(names []string) []map[string]any {
	sets := make([]map[string]any, 0, len(names))
	for _, name := range names {
		sets = append(sets, metaGeoipRuleSet(name))
	}
	return sets
}

func asnRuleSets(names []string) []map[string]any {
	sets := make([]map[string]any, 0, len(names))
	for _, name := range names {
		sets = append(sets, metaAsnRuleSet(name))
	}
	return sets
}

// ruleSetTags 把规则集名字列表转成对应的 tag 列表（如 "cn" -> "geosite-cn"），
// 用于生成引用这些规则集的 rule。kind 为 "geosite" 或 "geoip"。
func ruleSetTags(kind string, names []string) []string {
	tags := make([]string, 0, len(names))
	for _, name := range names {
		tags = append(tags, kind+"-"+name)
	}
	return tags
}

// isAliDNSCIDR 判断一条 ip_cidr 规则是否恰好就是 AliDNS 启动期直连例外
// (223.5.5.5/32)，用于 keepSniffRules 精确保留，不误留其他
// ip_cidr 规则（如 17.0.0.0/8 Apple 直连兜底）。
func isAliDNSCIDR(value any) bool {
	switch cidr := value.(type) {
	case string:
		return cidr == "223.5.5.5/32"
	case []any:
		if len(cidr) == 0 {
			return false
		}
		for _, item := range cidr {
			s, ok := item.(string)
			if !ok || s != "223.5.5.5/32" {
				return false
			}
		}
		return true
	default:
		return false
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
		// tailnet 流量（上面 leadingRuleSpecs 里的 tailnet 规则）在全局模式下同样必须走 endpoint。
		if outbound, _ := rm["outbound"].(string); outbound == moduleUtils.TagTailscale {
			kept = append(kept, rule)
			continue
		}
		// Preserve AliDNS direct-bypass so bootstrap DoH to 223.5.5.5 stays direct.
		// 只匹配 AliDNS 这个 /32——不能像以前那样"只要带 ip_cidr 就保留"，
		// 否则片段 1.5 新增的 17.0.0.0/8 (Apple IP 段直连兜底) 也会被无差别保留，
		// 导致全局代理模式下整个 Apple /8 都绕过代理，破坏"全局代理=一切走代理"
		// 的承诺——AliDNS 的 /32 是刻意的启动期例外，/8 不是同一类东西。
		if isAliDNSCIDR(rm["ip_cidr"]) {
			kept = append(kept, rule)
		}
		// 保留局域网私网直连规则，防止全局代理模式下局域网设备断连
		if isPrivate, _ := rm["ip_is_private"].(bool); isPrivate {
			kept = append(kept, rule)
		}
	}
	return kept
}

// keepTailscaleRules 只保留出站为 tailscale endpoint 的规则。
func keepTailscaleRules(rules []option.Rule) []option.Rule {
	var kept []option.Rule
	for _, rule := range rules {
		raw, err := singboxjson.Marshal(rule)
		if err != nil {
			continue
		}
		var rm map[string]any
		if err := singboxjson.Unmarshal(raw, &rm); err != nil {
			continue
		}
		if outbound, _ := rm["outbound"].(string); outbound == moduleUtils.TagTailscale {
			kept = append(kept, rule)
		}
	}
	return kept
}
