package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
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
	for _, r := range rules {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if rm["action"] == "hijack-dns" {
			hasHijackDNS = true
		}
	}
	if !hasHijackDNS {
		t.Fatalf("global route must keep hijack-dns rule to prevent DNS leaks")
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
		if stringListContains(rm["rule_set"], "geosite-google") && builtinWhitelistIdx < 0 {
			builtinWhitelistIdx = i
		}
	}

	if userRuleIdx < 0 {
		t.Fatalf("user rule not found in generated route rules")
	}
	if builtinWhitelistIdx < 0 {
		t.Fatalf("builtin geosite-google whitelist rule not found")
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

func TestRouteApply_IPv6Block(t *testing.T) {
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

	hasIPv6Block := false
	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if ipVer, _ := rm["ip_version"].(float64); ipVer == 6 && rm["action"] == "reject" {
			hasIPv6Block = true
			break
		}
	}

	if !hasIPv6Block {
		t.Fatalf("expected IPv6 reject rule, but not found")
	}
}

func TestRouteApply_RuleDirect_FinalIsDirectWithFullRules(t *testing.T) {
	opts := &option.Options{}
	m := &RouteModule{RouteMode: model.RouteModeRuleDirect}
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

	// 验证 final = direct
	if routeMap["final"] != "direct" {
		t.Fatalf("rule-direct route final = %v, want direct", routeMap["final"])
	}

	rules, ok := routeMap["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("rule-direct route rules must not be empty")
	}

	// 验证完整规则链存在：sniff、hijack-dns、geosite-gfw → proxy，以及 geoip-telegram/google → proxy
	hasSniff := false
	hasHijackDNS := false
	hasGFWProxy := false
	hasCNDirect := false
	hasTelegramIPProxy := false
	hasGoogleIPProxy := false
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
		if stringListContains(rm["rule_set"], "geosite-gfw") && rm["outbound"] == "proxy" {
			hasGFWProxy = true
		}
		if stringListContains(rm["rule_set"], "geosite-cn") {
			hasCNDirect = true
		}
		if stringListContains(rm["rule_set"], "geoip-telegram") && rm["outbound"] == "proxy" {
			hasTelegramIPProxy = true
		}
		if stringListContains(rm["rule_set"], "geoip-google") && rm["outbound"] == "proxy" {
			hasGoogleIPProxy = true
		}
	}

	if !hasSniff {
		t.Fatalf("rule-direct mode must keep sniff rule")
	}
	if !hasHijackDNS {
		t.Fatalf("rule-direct mode must keep hijack-dns rule")
	}
	if !hasGFWProxy {
		t.Fatalf("rule-direct mode must keep geosite-gfw → proxy rule")
	}
	if hasCNDirect {
		t.Fatalf("rule-direct mode must not have geosite-cn rule")
	}
	if !hasTelegramIPProxy {
		t.Fatalf("rule-direct mode must route geoip-telegram to proxy")
	}
	if !hasGoogleIPProxy {
		t.Fatalf("rule-direct mode must route geoip-google to proxy")
	}
}

func TestRouteApply_RuleDirect_RulesNotCleared(t *testing.T) {
	// rule-direct 应保留完整规则链，而 direct 模式应清空规则
	optsDirect := &option.Options{}
	mDirect := &RouteModule{RouteMode: model.RouteModeDirect}
	if err := mDirect.Apply(optsDirect, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply direct route: %v", err)
	}

	optsRuleDirect := &option.Options{}
	mRuleDirect := &RouteModule{RouteMode: model.RouteModeRuleDirect}
	if err := mRuleDirect.Apply(optsRuleDirect, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply rule-direct route: %v", err)
	}

	// direct 模式：rules 应为 nil
	if optsDirect.Route.Rules != nil {
		t.Fatalf("direct mode should have nil rules, got %d rules", len(optsDirect.Route.Rules))
	}

	// rule-direct 模式：rules 应非空
	if len(optsRuleDirect.Route.Rules) == 0 {
		t.Fatalf("rule-direct mode should preserve rules, but rules are empty")
	}
}
