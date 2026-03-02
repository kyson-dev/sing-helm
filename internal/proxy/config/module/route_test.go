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
