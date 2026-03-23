package module

import (
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	nodeProvider "github.com/kyson-dev/sing-helm/internal/proxy/config/module/node"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

type stubNodeProvider struct {
	name  string
	nodes []model.Node
	err   error
}

func (p *stubNodeProvider) Name() string { return p.name }
func (p *stubNodeProvider) GetNodes() ([]model.Node, error) {
	return p.nodes, p.err
}

func TestOutboundApply_GeneratedOverridesUserAndEmptyGroupsAutoFill(t *testing.T) {
	opts := &option.Options{}

	var userNode option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&userNode, map[string]any{
		"type":        "vless",
		"tag":         "user-node",
		"server":      "1.1.1.1",
		"server_port": 443,
		"uuid":        "11111111-1111-1111-1111-111111111111",
	}); err != nil {
		t.Fatalf("user node: %v", err)
	}
	var userProxy option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&userProxy, map[string]any{
		"type":      "selector",
		"tag":       moduleUtils.TagProxy,
		"outbounds": []string{"user-node"},
		"default":   "user-node",
	}); err != nil {
		t.Fatalf("user proxy group: %v", err)
	}
	var emptySelector option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&emptySelector, map[string]any{
		"type":      "selector",
		"tag":       "my-empty-group",
		"outbounds": []string{},
	}); err != nil {
		t.Fatalf("empty group: %v", err)
	}
	var keepSelector option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&keepSelector, map[string]any{
		"type":      "selector",
		"tag":       "my-keep-group",
		"outbounds": []string{"{all}"},
	}); err != nil {
		t.Fatalf("keep group: %v", err)
	}

	opts.Outbounds = []option.Outbound{userNode, userProxy, emptySelector, keepSelector}

	provider := &stubNodeProvider{
		name: "sub",
		nodes: []model.Node{
			{
				Name:   "sub-node",
				Type:   "vless",
				Source: "sub",
				Outbound: map[string]any{
					"server":      "2.2.2.2",
					"server_port": 443,
					"uuid":        "22222222-2222-2222-2222-222222222222",
				},
			},
		},
	}

	mod := NewOutboundModule(provider)
	if err := mod.Apply(opts, NewBuildContext(&model.RunOptions{})); err != nil {
		t.Fatalf("apply outbound: %v", err)
	}

	var proxy *option.SelectorOutboundOptions
	var auto *option.URLTestOutboundOptions
	var empty *option.SelectorOutboundOptions
	var keep *option.SelectorOutboundOptions
	for i := range opts.Outbounds {
		out := &opts.Outbounds[i]
		switch out.Tag {
		case moduleUtils.TagProxy:
			proxy = out.Options.(*option.SelectorOutboundOptions)
		case moduleUtils.TagAuto:
			auto = out.Options.(*option.URLTestOutboundOptions)
		case "my-empty-group":
			empty = out.Options.(*option.SelectorOutboundOptions)
		case "my-keep-group":
			keep = out.Options.(*option.SelectorOutboundOptions)
		}
	}

	if proxy == nil || auto == nil || empty == nil || keep == nil {
		t.Fatalf("missing expected groups after apply")
	}
	if proxy.Default != moduleUtils.TagAuto {
		t.Fatalf("expected generated proxy default=auto, got %q", proxy.Default)
	}
	if len(auto.Outbounds) != 2 {
		t.Fatalf("expected auto contains user+sub nodes, got %v", auto.Outbounds)
	}
	if len(empty.Outbounds) != 2 {
		t.Fatalf("expected empty selector auto-filled with all nodes, got %v", empty.Outbounds)
	}
	if len(keep.Outbounds) != 1 || keep.Outbounds[0] != "{all}" {
		t.Fatalf("expected non-empty selector untouched, got %v", keep.Outbounds)
	}
}

func TestOutboundApply_UserActualOutboundsAreNotDuplicated(t *testing.T) {
	opts := &option.Options{}
	var userNode option.Outbound
	if err := moduleUtils.ApplyMapToOutbound(&userNode, map[string]any{
		"type":        "vless",
		"tag":         "user-node",
		"server":      "1.1.1.1",
		"server_port": 443,
		"uuid":        "11111111-1111-1111-1111-111111111111",
	}); err != nil {
		t.Fatalf("user node: %v", err)
	}
	opts.Outbounds = []option.Outbound{userNode}

	mod := NewOutboundModule(&stubNodeProvider{name: "sub"})
	if err := mod.Apply(opts, NewBuildContext(&model.RunOptions{})); err != nil {
		t.Fatalf("apply outbound: %v", err)
	}

	count := 0
	for _, out := range opts.Outbounds {
		if out.Tag == "user-node" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected user node kept once, got %d", count)
	}
}

var _ nodeProvider.NodeProvider = (*stubNodeProvider)(nil)
