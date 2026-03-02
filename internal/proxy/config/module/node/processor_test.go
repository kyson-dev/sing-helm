package node

import (
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
)

func TestAddNodes_DedupeKeepsAliasMapping(t *testing.T) {
	p := NewOutboundProcessor()
	p.AddNodes([]model.Node{
		{
			Name:   "first",
			Source: "s1",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "1.1.1.1",
				"server_port": 443,
				"uuid":        "u-1",
			},
		},
		{
			Name:   "alias",
			Source: "s2",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "1.1.1.1",
				"server_port": 443,
				"uuid":        "u-1",
			},
		},
	})

	outbounds := p.GetProcessedOutbounds()
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound after dedupe, got %d", len(outbounds))
	}

	firstTag := p.originalToTag["s1"]["first"]
	aliasTag := p.originalToTag["s2"]["alias"]
	if firstTag == "" || aliasTag == "" {
		t.Fatalf("expected both original names to have mappings, got first=%q alias=%q", firstTag, aliasTag)
	}
	if firstTag != aliasTag {
		t.Fatalf("expected alias to map to canonical tag %q, got %q", firstTag, aliasTag)
	}

	mapped, ok := p.resolveDetour("alias")
	if !ok || mapped != firstTag {
		t.Fatalf("expected detour alias to resolve to %q, got %q (ok=%v)", firstTag, mapped, ok)
	}
}

func TestResolveDetour_AmbiguousGlobalNameRequiresSourceQualifier(t *testing.T) {
	p := NewOutboundProcessor()
	p.AddNodes([]model.Node{
		{
			Name:   "same",
			Source: "alpha",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "1.1.1.1",
				"server_port": 443,
				"uuid":        "u-1",
			},
		},
		{
			Name:   "same",
			Source: "beta",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "2.2.2.2",
				"server_port": 443,
				"uuid":        "u-2",
			},
		},
	})

	alphaTag := p.originalToTag["alpha"]["same"]
	betaTag := p.originalToTag["beta"]["same"]
	if alphaTag == "" || betaTag == "" || alphaTag == betaTag {
		t.Fatalf("expected distinct resolved tags for qualified lookup, alpha=%q beta=%q", alphaTag, betaTag)
	}
	if resolved, ok := p.resolveDetour("same"); !ok || resolved != alphaTag {
		t.Fatalf("expected unqualified detour to resolve concrete existing tag %q, got %q (ok=%v)", alphaTag, resolved, ok)
	}

	if resolved, ok := p.resolveDetour("alpha/same"); !ok || resolved != alphaTag {
		t.Fatalf("expected alpha/same -> %q, got %q (ok=%v)", alphaTag, resolved, ok)
	}
	if resolved, ok := p.resolveDetour("beta/same"); !ok || resolved != betaTag {
		t.Fatalf("expected beta/same -> %q, got %q (ok=%v)", betaTag, resolved, ok)
	}
}

func TestFingerprint_DifferentCredentialsAreNotDeduped(t *testing.T) {
	p := NewOutboundProcessor()
	p.AddNodes([]model.Node{
		{
			Name:   "node-a",
			Source: "s1",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "3.3.3.3",
				"server_port": 443,
				"uuid":        "uuid-a",
			},
		},
		{
			Name:   "node-b",
			Source: "s2",
			Type:   "vless",
			Outbound: map[string]any{
				"server":      "3.3.3.3",
				"server_port": 443,
				"uuid":        "uuid-b",
			},
		},
	})

	outbounds := p.GetProcessedOutbounds()
	if len(outbounds) != 2 {
		t.Fatalf("expected 2 outbounds when credentials differ, got %d", len(outbounds))
	}
}
