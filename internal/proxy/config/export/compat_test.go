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

func TestApplyCompatForV1114_StripsCertificatePublicKeySHA256(t *testing.T) {
	root := map[string]any{
		"outbounds": []any{
			map[string]any{
				"type": "hysteria2",
				"tag":  "proxy",
				"tls": map[string]any{
					"enabled":                       true,
					"certificate_public_key_sha256": "dGVzdGtleQ==",
				},
			},
		},
	}

	applyCompatForV1114(root)

	outbounds, ok := root["outbounds"].([]any)
	if !ok || len(outbounds) == 0 {
		t.Fatalf("outbounds missing")
	}
	outbound, ok := outbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("outbound[0] is not an object")
	}
	tls, ok := outbound["tls"].(map[string]any)
	if !ok {
		t.Fatalf("tls missing")
	}
	if _, ok := tls["certificate_public_key_sha256"]; ok {
		t.Fatalf("expected certificate_public_key_sha256 to be stripped for v1.11.4")
	}
	if tls["insecure"] != true {
		t.Fatalf("expected insecure fallback to be set, got %#v", tls["insecure"])
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
