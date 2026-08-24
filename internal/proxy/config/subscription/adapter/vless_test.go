package adapter

import (
	"testing"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

func TestVLessAdapter_FromURI_TCPTransportIsOmitted(t *testing.T) {
	node, err := (&VLessAdapter{}).FromURI("id@example.com:443?type=tcp&security=tls&sni=example.com#tcp-node")
	if err != nil {
		t.Fatalf("FromURI failed: %v", err)
	}
	if _, exists := node.Outbound["transport"]; exists {
		t.Fatalf("TCP must use sing-box's default transport, got %#v", node.Outbound["transport"])
	}

	var outbound option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&outbound, node.Outbound); err != nil {
		t.Fatalf("converted node must be accepted by sing-box: %v", err)
	}
}
