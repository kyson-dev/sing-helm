package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

func TestDNSApply_SystemServerPriorityUserRulesFirst(t *testing.T) {
	opts := &option.Options{}
	opts.DNS = &option.DNSOptions{}

	// user local_dns should be dropped because system local_dns is authoritative.
	if err := applyDNSFromMap(opts.DNS, map[string]any{
		"servers": []map[string]any{
			{"tag": "local_dns", "type": "udp", "server": "8.8.8.8"},
			{"tag": "user_dns", "type": "udp", "server": "9.9.9.9"},
		},
		"rules": []map[string]any{
			{"domain_suffix": []string{"example.com"}, "action": "route", "server": "user_dns"},
		},
		"final":    "user_dns",
		"strategy": "prefer_ipv6",
	}); err != nil {
		t.Fatalf("build user dns: %v", err)
	}

	if err := (&DNSModule{}).Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply dns: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.DNS)
	if err != nil {
		t.Fatalf("marshal dns: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode dns: %v", err)
	}

	servers := m["servers"].([]any)
	// system servers: local_dns + proxy_dns + dns_fakeip
	if len(servers) < 4 {
		t.Fatalf("expected system(3)+user(1) servers, got %d", len(servers))
	}
	firstTag := servers[0].(map[string]any)["tag"].(string)
	if firstTag != "local_dns" {
		t.Fatalf("expected system local_dns first, got %q", firstTag)
	}
	userDNSCount := 0
	for _, s := range servers {
		if s.(map[string]any)["tag"] == "user_dns" {
			userDNSCount++
		}
	}
	if userDNSCount != 1 {
		t.Fatalf("expected user_dns kept once, got %d", userDNSCount)
	}

	rules := m["rules"].([]any)
	// user(1) + system default rules: ads-reject, HTTPS/SVCB-reject, fakeip A/AAAA,
	// PTR->local_dns, private-domain-suffix->local_dns, directDomainRuleSets->local_dns.
	if len(rules) != 7 {
		t.Fatalf("expected user rules + default rules, got %d", len(rules))
	}
	firstRule := rules[0].(map[string]any)
	if firstRule["server"] != "user_dns" {
		t.Fatalf("expected user rule first, got %v", firstRule)
	}

	if m["final"] != "proxy_dns" {
		t.Fatalf("expected final forced to proxy_dns, got %v", m["final"])
	}
	if m["strategy"] != "prefer_ipv4" {
		t.Fatalf("expected strategy forced to prefer_ipv4, got %v", m["strategy"])
	}
}

func TestDNSApply_RuleMode_GeositeRoutingAndFinal(t *testing.T) {
	opts := &option.Options{}
	opts.DNS = &option.DNSOptions{}

	m := &DNSModule{RouteMode: "rule"}
	if err := m.Apply(opts, NewBuildContext(nil)); err != nil {
		t.Fatalf("apply dns: %v", err)
	}

	raw, err := singboxjson.Marshal(opts.DNS)
	if err != nil {
		t.Fatalf("marshal dns: %v", err)
	}
	var res map[string]any
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode dns: %v", err)
	}

	if res["final"] != "proxy_dns" {
		t.Fatalf("expected final to be proxy_dns under rule mode, got %v", res["final"])
	}

	rules, ok := res["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("rules must not be empty")
	}

	hasCNLocalDNS := false
	hasFutuLocalDNS := false
	for _, rule := range rules {
		rm, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if stringListContains(rm["rule_set"], "geosite-cn") && rm["server"] == "local_dns" {
			hasCNLocalDNS = true
		}
		// futu 不是硬编码在 dns.go 里的，而是和 route.go 共用同一份
		// directDomainRuleSets；这里能查到就证明两边真的用的是单一数据源。
		if stringListContains(rm["rule_set"], "geosite-futu") && rm["server"] == "local_dns" {
			hasFutuLocalDNS = true
		}
	}

	if !hasCNLocalDNS {
		t.Fatalf("rule mode should route geosite-cn to local_dns")
	}
	if !hasFutuLocalDNS {
		t.Fatalf("dns rules should stay in sync with route.go's directDomainRuleSets (geosite-futu missing)")
	}
}


func applyDNSFromMap(d *option.DNSOptions, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	return singboxjson.UnmarshalContext(include.Context(context.Background()), data, d)
}
