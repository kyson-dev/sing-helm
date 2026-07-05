package export

import "testing"

func TestApplyCompatForV1114_KeepTunAddress(t *testing.T) {
	root := map[string]any{
		"inbounds": []any{
			map[string]any{
				"type":    "tun",
				"tag":     "tun-in",
				"address": "172.19.0.1/30",
			},
		},
	}

	applyCompatForV1114(root)

	tun := firstInboundAsMap(t, root)
	if _, ok := tun["address"]; !ok {
		t.Fatalf("expected address to be kept for v1.11.4")
	}
	if _, ok := tun["inet4_address"]; ok {
		t.Fatalf("expected inet4_address not to be set for v1.11.4")
	}
}

func TestApplyCompatForV1114_StripsDefaultDomainResolver(t *testing.T) {
	root := map[string]any{
		"route": map[string]any{
			"final":                   "proxy",
			"default_domain_resolver": "local_dns",
		},
	}

	applyCompatForV1114(root)

	route, ok := root["route"].(map[string]any)
	if !ok {
		t.Fatalf("route missing")
	}
	if _, ok := route["default_domain_resolver"]; ok {
		t.Fatalf("expected default_domain_resolver to be stripped for v1.11.4")
	}
	if route["final"] != "proxy" {
		t.Fatalf("expected other route fields to be preserved, got %#v", route)
	}
}

func firstInboundAsMap(t *testing.T, root map[string]any) map[string]any {
	t.Helper()

	inbounds, ok := root["inbounds"].([]any)
	if !ok || len(inbounds) == 0 {
		t.Fatalf("inbounds missing")
	}
	tun, ok := inbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("inbound[0] is not an object")
	}
	return tun
}
