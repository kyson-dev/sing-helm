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
	if len(rules) != 3 {
		t.Fatalf("expected user rules + default rules, got %d", len(rules))
	}
	firstRule := rules[0].(map[string]any)
	if firstRule["server"] != "user_dns" {
		t.Fatalf("expected user rule first, got %v", firstRule)
	}

	if m["final"] != "proxy_dns" {
		t.Fatalf("expected final forced to proxy_dns, got %v", m["final"])
	}
	if m["strategy"] != "ipv4_only" {
		t.Fatalf("expected strategy forced to ipv4_only, got %v", m["strategy"])
	}
}

func applyDNSFromMap(d *option.DNSOptions, m map[string]any) error {
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	return singboxjson.UnmarshalContext(include.Context(context.Background()), data, d)
}
