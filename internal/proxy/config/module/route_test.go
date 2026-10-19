package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

func TestRouteApply_DNSHijackBeforePrivateDirect(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("route rules missing")
	}

	dnsHijackIdx := -1
	privateDirectIdx := -1

	for i, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if protocolHasDNS(rm["protocol"]) && rm["action"] == "hijack-dns" && dnsHijackIdx < 0 {
			dnsHijackIdx = i
		}
		if v, ok := rm["ip_is_private"].(bool); ok && v && privateDirectIdx < 0 {
			privateDirectIdx = i
		}
	}

	if dnsHijackIdx < 0 {
		t.Fatalf("dns hijack rule not found")
	}
	if privateDirectIdx < 0 {
		t.Fatalf("ip_is_private direct rule not found")
	}
	if dnsHijackIdx >= privateDirectIdx {
		t.Fatalf("dns hijack must be before ip_is_private: hijack=%d private=%d", dnsHijackIdx, privateDirectIdx)
	}
}

func TestRouteApply_GlobalKeepsSniff(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeGlobal}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	if routeMap["final"] != "proxy" {
		t.Fatalf("global route final = %v, want proxy", routeMap["final"])
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("global route must keep sniff rule")
	}
	// Global mode keeps: sniff, hijack-dns, and ip_cidr (AliDNS bypass).
	// Verify the first rule is sniff and hijack-dns is present.
	firstRule, ok := rules[0].(map[string]any)
	if !ok || firstRule["action"] != "sniff" {
		t.Fatalf("global route first rule = %#v, want sniff action", rules[0])
	}
	hasHijackDNS := false
	hasAliDNSBypass := false
	for _, r := range rules {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if rm["action"] == "hijack-dns" {
			hasHijackDNS = true
		}
		if stringListContains(rm["ip_cidr"], "223.6.6.6/32") {
			t.Fatalf("global route kept unused AliDNS bypass: %v", rm)
		}
		if stringListContains(rm["ip_cidr"], "223.5.5.5/32") && rm["outbound"] == "direct" {
			hasAliDNSBypass = true
		}
	}
	if !hasHijackDNS {
		t.Fatalf("global route must keep hijack-dns rule to prevent DNS leaks")
	}
	if !hasAliDNSBypass {
		t.Fatalf("global route must keep the exact 223.5.5.5/32 DoH bootstrap bypass: %#v", rules)
	}
}

func TestRouteApply_DefaultDomainResolverPointsToLocalDNS(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	if routeMap["default_domain_resolver"] != "local_dns" {
		t.Fatalf("default_domain_resolver = %v, want local_dns", routeMap["default_domain_resolver"])
	}
}

func TestRouteApply_UserDefaultDomainResolverIsPreserved(t *testing.T) {
	opts := &option.Options{
		Route: &option.RouteOptions{
			DefaultDomainResolver: &option.DomainResolveOptions{Server: "custom_dns"},
		},
	}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	if opts.Route.DefaultDomainResolver == nil || opts.Route.DefaultDomainResolver.Server != "custom_dns" {
		t.Fatalf("user default_domain_resolver was overwritten: %#v", opts.Route.DefaultDomainResolver)
	}
}

// stringListContains checks whether v (a decoded badoption.Listable[string],
// which collapses to a bare string when it has exactly one element) contains target.
func stringListContains(v any, target string) bool {
	switch p := v.(type) {
	case string:
		return p == target
	case []any:
		for _, item := range p {
			if s, ok := item.(string); ok && s == target {
				return true
			}
		}
	}
	return false
}

func protocolHasDNS(v any) bool {
	switch p := v.(type) {
	case string:
		return p == "dns"
	case []any:
		for _, item := range p {
			s, ok := item.(string)
			if ok && s == "dns" {
				return true
			}
		}
	case []string:
		for _, s := range p {
			if s == "dns" {
				return true
			}
		}
	}
	return false
}

func TestRouteApply_UserRulesBetweenLeadingAndTrailing(t *testing.T) {
	tx := include.Context(context.Background())
	userRuleData, err := singboxjson.Marshal(map[string]any{
		"domain_suffix": []string{"example.com"},
		"outbound":      "direct",
	})
	if err != nil {
		t.Fatalf("marshal user rule: %v", err)
	}
	var userRule option.Rule
	if err := singboxjson.UnmarshalContext(tx, userRuleData, &userRule); err != nil {
		t.Fatalf("unmarshal user rule: %v", err)
	}

	opts := &option.Options{
		Route: &option.RouteOptions{
			Rules: []option.Rule{userRule},
		},
	}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) < 3 {
		t.Fatalf("route rules missing or too short: %#v", rules)
	}

	first, _ := rules[0].(map[string]any)
	if first["action"] != "sniff" {
		t.Fatalf("rules[0] = %#v, want action sniff", rules[0])
	}
	second, _ := rules[1].(map[string]any)
	if !protocolHasDNS(second["protocol"]) || second["action"] != "hijack-dns" {
		t.Fatalf("rules[1] = %#v, want hijack-dns", rules[1])
	}

	userRuleIdx, builtinWhitelistIdx := -1, -1
	for i, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if stringListContains(rm["domain_suffix"], "example.com") && userRuleIdx < 0 {
			userRuleIdx = i
		}
		if stringListContains(rm["rule_set"], "geosite-cn") && builtinWhitelistIdx < 0 {
			builtinWhitelistIdx = i
		}
	}

	if userRuleIdx < 0 {
		t.Fatalf("user rule not found in generated route rules")
	}
	if builtinWhitelistIdx < 0 {
		t.Fatalf("builtin geosite-cn whitelist rule not found")
	}
	if userRuleIdx >= builtinWhitelistIdx {
		t.Fatalf("user rule must come before builtin whitelist rules: user=%d builtin=%d", userRuleIdx, builtinWhitelistIdx)
	}
	if userRuleIdx <= 1 {
		t.Fatalf("user rule must come after sniff/hijack-dns: user=%d", userRuleIdx)
	}
}

func TestRouteApply_GeoipCNFallbackPresent(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("route rules missing")
	}

	resolveIdx, geoipCNIdx := -1, -1
	for i, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if rm["action"] == "resolve" && resolveIdx < 0 {
			resolveIdx = i
		}
		if rm["outbound"] == "direct" && stringListContains(rm["rule_set"], "geoip-cn") {
			geoipCNIdx = i
		}
	}

	if resolveIdx < 0 {
		t.Fatalf("resolve action rule not found")
	}
	if geoipCNIdx < 0 {
		t.Fatalf("geoip-cn direct fallback rule not found")
	}
	if geoipCNIdx <= resolveIdx {
		t.Fatalf("geoip-cn fallback must come after resolve: geoip-cn=%d resolve=%d", geoipCNIdx, resolveIdx)
	}
}

// TestRouteApply_RuleSetHTTPClientDetourRoundTrips 验证生成的 rule_set 用
// http_client.detour（而非废弃的 download_detour）声明下载出站，且该写法能被
// sing-box 自己的严格解码器（DisallowUnknownFields）正确解析回 detour 字段——
// 只 json.Unmarshal 到 map[string]any 不会校验 http_client 是否是 sing-box
// 认识的合法字段，必须走 option.Route 才能验证。
func TestRouteApply_RuleSetHTTPClientDetourRoundTrips(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}

	var decoded option.RouteOptions
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, raw, &decoded); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	if len(decoded.RuleSet) == 0 {
		t.Fatalf("rule_set missing")
	}
	found := false
	for _, rs := range decoded.RuleSet {
		if rs.Type != "remote" {
			continue
		}
		if rs.RemoteOptions.DownloadDetour != "" {
			t.Fatalf("expected download_detour to be unused, got %q on %v", rs.RemoteOptions.DownloadDetour, rs.Tag)
		}
		if rs.RemoteOptions.HTTPClient != nil && rs.RemoteOptions.HTTPClient.Detour == moduleUtils.TagProxy {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected at least one remote rule_set with http_client.detour == %q", moduleUtils.TagProxy)
	}
}

// TestRouteApply_AppleIPRangeDirect 验证 17.0.0.0/8 (Apple 自有 IP 段) 有直连
// 兜底规则——APNs 推送等裸 IP/QUIC 连接嗅探不到域名，需要靠这条规则兜底直连，
// 否则会落到 Final=proxy 导致推送通道频繁经代理重连。
func TestRouteApply_AppleIPRangeDirect(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("route rules missing")
	}

	found := false
	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if stringListContains(rm["ip_cidr"], "17.0.0.0/8") && rm["outbound"] == "direct" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 17.0.0.0/8 -> direct fallback rule for Apple's own IP range")
	}
}

// TestRouteApply_GlobalModeDropsAppleIPRange 验证全局代理模式下 17.0.0.0/8
// 直连兜底规则不会被 keepSniffRules 误保留——只有 AliDNS 的 223.5.5.5/32
// 是刻意保留的启动期例外，/8 这种大段不该在"全局代理"下还绕过代理。
func TestRouteApply_GlobalModeDropsAppleIPRange(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeGlobal}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok {
		t.Fatalf("route rules missing")
	}

	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if stringListContains(rm["ip_cidr"], "17.0.0.0/8") {
			t.Fatalf("global mode must not keep the 17.0.0.0/8 direct bypass: %v", rm)
		}
	}
}

// TestRouteApply_NoUnconditionalIPv6Block 验证不存在无条件拦截字面量 IPv6 的
// reject 规则——删除原因见 docs/design/fake-ip-routing.md 的 IPv6 policy 一节：
// 本机没有真实 v6 路由时 direct 拨号会立刻收到 no route to host，不需要在路由层
// 提前拦截，无差别拦截反而会误杀本来能成功的连接（如蜂窝网络）。
func TestRouteApply_NoUnconditionalIPv6Block(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("route rules missing")
	}

	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if ipVer, _ := rm["ip_version"].(float64); ipVer == 6 && rm["action"] == "reject" {
			t.Fatalf("unexpected unconditional IPv6 reject rule found: %v", rm)
		}
	}
}

func TestRouteApply_ICMPUsesNetworkMatcher(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok {
		t.Fatal("route rules missing")
	}
	found := false
	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if stringListContains(rm["protocol"], "icmp") {
			t.Fatalf("ICMP must not use the protocol matcher: %v", rm)
		}
		if stringListContains(rm["network"], "icmp") && rm["outbound"] == moduleUtils.TagDirect {
			found = true
		}
	}
	if !found {
		t.Fatal("network=icmp direct rule not found")
	}
}

func TestRouteApply_RuleMode_FinalIsProxyWithFullRules(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRule}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply route: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.Route)
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}
	var routeMap map[string]any
	if err := json.Unmarshal(raw, &routeMap); err != nil {
		t.Fatalf("decode route: %v", err)
	}

	// 验证 final = proxy
	if routeMap["final"] != "proxy" {
		t.Fatalf("rule route final = %v, want proxy", routeMap["final"])
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("rule route rules must not be empty")
	}

	// 验证完整规则链存在：sniff、hijack-dns、geosite-cn/futu → direct，
	// 以及 geoip-telegram/google → proxy。不再有单独的 geosite-gfw → proxy
	// 规则——未命中直连白名单的域名（含 GFW 封锁站点）天然落到 Final=proxy。
	hasSniff := false
	hasHijackDNS := false
	hasCNDirect := false
	hasFutuDirect := false
	hasTelegramIPProxy := false
	hasGoogleIPProxy := false
	hasAsnDirect := false
	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if rm["action"] == "sniff" {
			hasSniff = true
		}
		if rm["action"] == "hijack-dns" {
			hasHijackDNS = true
		}
		if stringListContains(rm["rule_set"], "geosite-cn") && rm["outbound"] == "direct" {
			hasCNDirect = true
		}
		if stringListContains(rm["rule_set"], "geosite-futu") && rm["outbound"] == "direct" {
			hasFutuDirect = true
		}
		if stringListContains(rm["rule_set"], "geoip-telegram") && rm["outbound"] == "proxy" {
			hasTelegramIPProxy = true
		}
		if stringListContains(rm["rule_set"], "geoip-google") && rm["outbound"] == "proxy" {
			hasGoogleIPProxy = true
		}
		if stringListContains(rm["rule_set"], "asn-AS132203") && rm["outbound"] == "direct" {
			hasAsnDirect = true
		}
	}

	if !hasSniff {
		t.Fatalf("rule mode must keep sniff rule")
	}
	if !hasHijackDNS {
		t.Fatalf("rule mode must keep hijack-dns rule")
	}
	if !hasCNDirect {
		t.Fatalf("rule mode should have geosite-cn → direct rule")
	}
	if !hasFutuDirect {
		t.Fatalf("rule mode should have geosite-futu → direct rule")
	}
	if !hasTelegramIPProxy {
		t.Fatalf("rule mode must route geoip-telegram to proxy")
	}
	if !hasGoogleIPProxy {
		t.Fatalf("rule mode must route geoip-google to proxy")
	}
	if !hasAsnDirect {
		t.Fatalf("rule mode should have asn-AS132203 → direct rule")
	}
}

func TestRouteApply_DirectModeClearsRules(t *testing.T) {
	optsDirect := &option.Options{}
	mDirect := &RouteModule{RouteMode: model.RouteModeDirect}
	if err := mDirect.Apply(optsDirect, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply direct route: %v", err)
	}

	// 直连模式清空全部规则，只保留两条 tailnet 规则（100.64.0.0/10 与 *.ts.net → ts）。
	if len(optsDirect.Route.Rules) != 2 {
		t.Fatalf("direct mode should keep only the 2 tailnet rules, got %d rules", len(optsDirect.Route.Rules))
	}
}
