package node

import (
	"testing"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

func TestUserNodeProvider_GetNodes_OnlyActualOutbounds(t *testing.T) {
	var userNode option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&userNode, map[string]any{
		"type":        "vless",
		"tag":         "user-vless",
		"server":      "1.1.1.1",
		"server_port": 443,
		"uuid":        "11111111-1111-1111-1111-111111111111",
	}); err != nil {
		t.Fatalf("build vless outbound: %v", err)
	}

	var selector option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&selector, map[string]any{
		"type":      "selector",
		"tag":       "group",
		"outbounds": []string{"user-vless"},
	}); err != nil {
		t.Fatalf("build selector outbound: %v", err)
	}

	var direct option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&direct, map[string]any{
		"type": "direct",
		"tag":  "direct",
	}); err != nil {
		t.Fatalf("build direct outbound: %v", err)
	}

	provider := NewUserNodeProvider([]option.Outbound{userNode, selector, direct})
	nodes, err := provider.GetNodes()
	if err != nil {
		t.Fatalf("GetNodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 actual node, got %d", len(nodes))
	}
	if nodes[0].Name != "user-vless" || nodes[0].Type != "vless" || nodes[0].Source != "user" {
		t.Fatalf("unexpected node identity: %+v", nodes[0])
	}
	if gotTag := nodes[0].Outbound["tag"]; gotTag != "user-vless" {
		t.Fatalf("expected outbound tag user-vless, got %v", gotTag)
	}
}
