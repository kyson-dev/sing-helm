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
