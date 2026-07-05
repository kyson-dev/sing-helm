package adapter

import (
	"testing"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

func TestHysteria2Adapter_FromURI(t *testing.T) {
	a := &Hysteria2Adapter{}

	tests := []struct {
		name  string
		uri   string
		check func(t *testing.T, outbound map[string]any)
	}{
		{
			name: "obfs is parsed",
			uri:  "secret@example.com:443?sni=example.com&obfs=salamander&obfs-password=obfspass",
			check: func(t *testing.T, outbound map[string]any) {
				obfs, ok := outbound["obfs"].(map[string]any)
				if !ok {
					t.Fatalf("expected obfs map, got %v", outbound["obfs"])
				}
				if obfs["type"] != "salamander" {
					t.Fatalf("expected obfs type salamander, got %v", obfs["type"])
				}
				if obfs["password"] != "obfspass" {
					t.Fatalf("expected obfs password obfspass, got %v", obfs["password"])
				}
			},
		},
		{
			name: "pubKeySHA256 sets certificate pin",
			uri:  "secret@example.com:443?sni=example.com&pinSHA256=AA:BB&pubKeySHA256=dGVzdGtleQ==",
			check: func(t *testing.T, outbound map[string]any) {
				tls, ok := outbound["tls"].(map[string]any)
				if !ok {
					t.Fatalf("expected tls map, got %v", outbound["tls"])
				}
				if tls["certificate_public_key_sha256"] != "dGVzdGtleQ==" {
					t.Fatalf("expected certificate_public_key_sha256 to be set, got %v", tls["certificate_public_key_sha256"])
				}
				if tls["insecure"] == true {
					t.Fatalf("expected insecure to stay unset when pubKeySHA256 is present")
				}
			},
		},
		{
			name: "pinSHA256 without pubKeySHA256 falls back to insecure",
			uri:  "secret@example.com:443?sni=example.com&pinSHA256=AA:BB",
			check: func(t *testing.T, outbound map[string]any) {
				tls, ok := outbound["tls"].(map[string]any)
				if !ok {
					t.Fatalf("expected tls map, got %v", outbound["tls"])
				}
				if tls["insecure"] != true {
					t.Fatalf("expected insecure fallback, got %v", tls["insecure"])
				}
				if _, ok := tls["certificate_public_key_sha256"]; ok {
					t.Fatalf("did not expect certificate_public_key_sha256 to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := a.FromURI(tt.uri)
			if err != nil {
				t.Fatalf("FromURI failed: %v", err)
			}

			tt.check(t, node.Outbound)

			var outbound option.Outbound
			if err := moduleUtils.ApplyMapToOutbound(&outbound, node.Outbound); err != nil {
				t.Fatalf("ApplyMapToOutbound failed: %v", err)
			}
		})
	}
}
