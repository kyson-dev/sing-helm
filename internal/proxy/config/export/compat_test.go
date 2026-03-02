package export

import "testing"

func TestApplyVersionCompat_KeepTunAddressOnV111(t *testing.T) {
	root := map[string]any{
		"inbounds": []any{
			map[string]any{
				"type":    "tun",
				"tag":     "tun-in",
				"address": "172.19.0.1/30",
			},
		},
	}

	if err := applyVersionCompat(root, "1.11.4"); err != nil {
		t.Fatalf("apply version compat failed: %v", err)
	}

	tun := firstInboundAsMap(t, root)
	if _, ok := tun["address"]; !ok {
		t.Fatalf("expected address to be kept for v1.11.x")
	}
	if _, ok := tun["inet4_address"]; ok {
		t.Fatalf("expected inet4_address not to be set for v1.11.x")
	}
}

func TestApplyVersionCompat_DowngradeTunAddressOnV110(t *testing.T) {
	root := map[string]any{
		"inbounds": []any{
			map[string]any{
				"type":    "tun",
				"tag":     "tun-in",
				"address": "172.19.0.1/30",
			},
		},
	}

	if err := applyVersionCompat(root, "1.10.9"); err != nil {
		t.Fatalf("apply version compat failed: %v", err)
	}

	tun := firstInboundAsMap(t, root)
	if _, ok := tun["address"]; ok {
		t.Fatalf("expected address to be removed for v1.10.x")
	}
	if v, ok := tun["inet4_address"].(string); !ok || v != "172.19.0.1/30" {
		t.Fatalf("expected inet4_address to be set for v1.10.x, got %v", tun["inet4_address"])
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
