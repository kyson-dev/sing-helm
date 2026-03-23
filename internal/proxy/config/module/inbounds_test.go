package module

import (
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

func TestMixedApply_RemovesTunAndCreatesMixed(t *testing.T) {
	opts := &option.Options{}
	var tun option.Inbound
	if err := moduleUtils.ApplyMapToInbound(&tun, map[string]any{
		"type": "tun", "tag": "tun-in", "address": []string{"172.19.0.1/30"},
	}); err != nil {
		t.Fatalf("build tun: %v", err)
	}
	opts.Inbounds = []option.Inbound{tun}

	run := &model.RunOptions{ProxyMode: model.ProxyModeTUN}
	ctx := NewBuildContext(run)
	mod := &MixedModule{SetSystemProxy: true, ListenAddr: "127.0.0.1", Port: 7890}
	if err := mod.Apply(opts, ctx); err != nil {
		t.Fatalf("apply mixed: %v", err)
	}

	for _, in := range opts.Inbounds {
		if in.Type == "tun" || in.Tag == "tun-in" {
			t.Fatalf("tun inbound should be removed in mixed mode")
		}
	}
	if run.ListenAddr != "127.0.0.1" || run.MixedPort != 7890 {
		t.Fatalf("run options not backfilled correctly: %+v", run)
	}
	if run.ProxyMode != model.ProxyModeTUN {
		t.Fatalf("ProxyMode must not be backfilled, got %q", run.ProxyMode)
	}
}

func TestMixedApply_UserMixedIncompleteMustFail(t *testing.T) {
	opts := &option.Options{}
	var mixed option.Inbound
	if err := moduleUtils.ApplyMapToInbound(&mixed, map[string]any{
		"type": "mixed", "tag": "mixed-in", "set_system_proxy": false,
	}); err != nil {
		t.Fatalf("build mixed: %v", err)
	}
	opts.Inbounds = []option.Inbound{mixed}

	err := (&MixedModule{SetSystemProxy: true}).Apply(opts, NewBuildContext(&model.RunOptions{}))
	if err == nil {
		t.Fatalf("expected error for incomplete user mixed config")
	}
}

func TestMixedApply_UserMixedCompleteForceSetSystemProxyAndBackfill(t *testing.T) {
	opts := &option.Options{}
	var mixed option.Inbound
	if err := moduleUtils.ApplyMapToInbound(&mixed, map[string]any{
		"type":             "mixed",
		"tag":              "mixed-in",
		"listen":           "127.0.0.9",
		"listen_port":      19090,
		"set_system_proxy": false,
	}); err != nil {
		t.Fatalf("build mixed: %v", err)
	}
	opts.Inbounds = []option.Inbound{mixed}

	run := &model.RunOptions{ProxyMode: model.ProxyModeDefault}
	ctx := NewBuildContext(run)
	err := (&MixedModule{SetSystemProxy: true, ListenAddr: "127.0.0.1", Port: 7890}).Apply(opts, ctx)
	if err != nil {
		t.Fatalf("apply mixed: %v", err)
	}

	mixedOpts := opts.Inbounds[0].Options.(*option.HTTPMixedInboundOptions)
	if !mixedOpts.SetSystemProxy {
		t.Fatalf("expected set_system_proxy forced to true")
	}
	if run.ListenAddr != "127.0.0.9" || run.MixedPort != 19090 {
		t.Fatalf("expected backfill from user mixed, got %+v", run)
	}
	if run.ProxyMode != model.ProxyModeDefault {
		t.Fatalf("ProxyMode must not be backfilled, got %q", run.ProxyMode)
	}
}

func TestTUNApply_RemovesMixedLikeInbounds(t *testing.T) {
	opts := &option.Options{}
	var mixed option.Inbound
	if err := moduleUtils.ApplyMapToInbound(&mixed, map[string]any{
		"type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 7890,
	}); err != nil {
		t.Fatalf("build mixed: %v", err)
	}
	var httpIn option.Inbound
	if err := moduleUtils.ApplyMapToInbound(&httpIn, map[string]any{
		"type": "http", "tag": "http-in", "listen": "127.0.0.1", "listen_port": 8080,
	}); err != nil {
		t.Fatalf("build http: %v", err)
	}
	opts.Inbounds = []option.Inbound{mixed, httpIn}

	if err := (&TUNModule{}).Apply(opts, NewBuildContext(&model.RunOptions{})); err != nil {
		t.Fatalf("apply tun: %v", err)
	}
	hasTun := false
	for _, in := range opts.Inbounds {
		if in.Type == "mixed" || in.Type == "http" || in.Type == "socks" || in.Tag == "mixed-in" {
			t.Fatalf("mixed-like inbound should be removed in tun mode: %s/%s", in.Type, in.Tag)
		}
		if in.Type == "tun" || in.Tag == "tun-in" {
			hasTun = true
		}
	}
	if !hasTun {
		t.Fatalf("expected tun inbound created")
	}
}
