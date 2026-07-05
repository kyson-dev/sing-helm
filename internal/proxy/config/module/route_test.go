package module

import (
	"encoding/json"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
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
		t.Fatalf("expected IPv6 reject rule but not found")
	}
}
