package config

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/export"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// 用户 profile.json 里的一条普通规则，用来确认内置 Tailscale 规则排在它前面，
// 同时它自己仍排在系统保底规则前面。
const userRoutingProfile = `{
  "route": {"rules": [{"domain_suffix": ["binance.com"], "outbound": "direct"}]}
}`

func moduleNames(mods []module.ConfigModule) []string {
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		names = append(names, m.Name())
	}
	return names
}

func indexOf(names []string, want string) int {
	for i, n := range names {
		if n == want {
			return i
		}
	}
	return -1
}

// taildropDirFor 给测试用：与 state 目录同级的 taildrop 目录；stateDir 为空时不设置。
func taildropDirFor(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(stateDir), "taildrop")
}

func buildWithTailscaleModule(t *testing.T, profile string, stateDir string) *option.Options {
	t.Helper()
	opts := &option.Options{}
	if profile != "" {
		ctx := include.Context(context.Background())
		if err := singboxjson.UnmarshalContext(ctx, []byte(profile), opts); err != nil {
			t.Fatalf("profile should parse: %v", err)
		}
	}
	run := &model.RunOptions{ProxyMode: model.ProxyModeTUN, RouteMode: model.RouteModeRule, ListenAddr: "127.0.0.1"}
	bctx := module.NewBuildContext(run)
	mods := []module.ConfigModule{
		&module.EndpointModule{StateDir: stateDir, TaildropDir: taildropDirFor(stateDir)},
		&module.TUNModule{},
		&module.DNSModule{RouteMode: run.RouteMode},
		&module.RouteModule{RouteMode: run.RouteMode},
	}
	for _, m := range mods {
		if err := m.Apply(opts, bctx); err != nil {
			t.Fatalf("module %s: %v", m.Name(), err)
		}
	}
	return opts
}

func TestTailscaleModuleGeneratesEndpointDNSAndRoutes(t *testing.T) {
	opts := buildWithTailscaleModule(t, userRoutingProfile, "/run/sing-helm/tailscale")

	if len(opts.Endpoints) != 1 || opts.Endpoints[0].Type != "tailscale" || opts.Endpoints[0].Tag != "ts" {
		t.Fatalf("expected one tailscale endpoint tagged ts: %+v", opts.Endpoints)
	}
	raw, _ := singboxjson.Marshal(opts.Endpoints)
	if !strings.Contains(string(raw), `"state_directory":"/run/sing-helm/tailscale"`) {
		t.Fatalf("state_directory not set: %s", raw)
	}
	// 回归：开机由 launchd 启动时当前目录是 "/"，taildrop 目录不能留给 sing-box 的相对路径默认值。
	if !strings.Contains(string(raw), `"taildrop_directory":"/run/sing-helm/taildrop"`) {
		t.Fatalf("taildrop_directory must be an explicit absolute path: %s", raw)
	}

	foundDNS := false
	for _, s := range opts.DNS.Servers {
		if s.Tag == "ts_dns" && s.Type == "tailscale" {
			foundDNS = true
		}
	}
	if !foundDNS {
		t.Fatal("tailscale DNS server missing")
	}

	// *.ts.net 的 DNS 规则排在 fake-ip 规则之后：面向客户端的第一次解析仍拿到 fake-ip，
	// 出站拨号时的第二次解析会跳过 fake-ip 规则，落到这条规则由 MagicDNS 解析。
	dr, _ := singboxjson.Marshal(opts.DNS.Rules)
	d := string(dr)
	if iFake, iTS := strings.Index(d, "dns_fakeip"), strings.Index(d, "ts_dns"); iFake < 0 || iTS < 0 || iFake > iTS {
		t.Fatalf("ts.net DNS rule must exist after the fake-ip rule (fake=%d ts=%d)", iFake, iTS)
	}

	// 路由顺序：sniff → tailscale 规则 → 用户规则 → 系统保底规则。
	rr, _ := singboxjson.Marshal(opts.Route.Rules)
	r := string(rr)
	iSniff := strings.Index(r, `"sniff"`)
	iCGNAT := strings.Index(r, "100.64.0.0/10")
	iUser := strings.Index(r, "binance.com")
	iPrivate := strings.Index(r, "ip_is_private")
	if !(iSniff >= 0 && iSniff < iCGNAT && iCGNAT < iUser && iUser < iPrivate) {
		t.Fatalf("route rule order wrong: sniff=%d cgnat=%d user=%d private=%d", iSniff, iCGNAT, iUser, iPrivate)
	}
}

// state_directory 为空（例如 iOS 导出前的构建、或运行目录未初始化）时不写该字段，
// 交给 sing-box 选平台默认位置。
func TestTailscaleModuleOmitsEmptyStateDir(t *testing.T) {
	opts := buildWithTailscaleModule(t, "", "")
	raw, _ := singboxjson.Marshal(opts.Endpoints)
	if strings.Contains(string(raw), "state_directory") || strings.Contains(string(raw), "taildrop_directory") {
		t.Fatalf("empty dirs must not emit state_directory/taildrop_directory: %s", raw)
	}
}

// 试验阶段的做法是把 tailscale endpoint 写在 profile.json 里。开关打开时如果
// 还留着同名 endpoint，要给出明确的错误，而不是生成重复 tag 的配置。
func TestEndpointModuleRejectsProfileDefinedTailscaleEndpoint(t *testing.T) {
	ctx := include.Context(context.Background())
	opts := &option.Options{}
	if err := singboxjson.UnmarshalContext(ctx, []byte(tailscaleProfile), opts); err != nil {
		t.Fatalf("profile should parse: %v", err)
	}
	m := &module.EndpointModule{StateDir: "/x"}
	err := m.Apply(opts, module.NewBuildContext(&model.RunOptions{}))
	if err == nil || !strings.Contains(err.Error(), "profile.json") {
		t.Fatalf("expected a clear conflict error, got %v", err)
	}
	if len(opts.Endpoints) != 1 {
		t.Fatalf("endpoints must not be modified on conflict: %d", len(opts.Endpoints))
	}
}

// 手机导出：模块生成的桌面路径 state_directory 不能出现在 iOS 配置里，
// 但 endpoint、MagicDNS 和路由必须保留。
func TestTailscaleModuleIOSExportDropsStateDir(t *testing.T) {
	opts := buildWithTailscaleModule(t, "", "/run/sing-helm/tailscale")
	data, err := export.Export(opts, export.Target{Platform: "ios"})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	out := string(data)
	if strings.Contains(out, "state_directory") || strings.Contains(out, "/run/sing-helm/tailscale") ||
		strings.Contains(out, "taildrop_directory") || strings.Contains(out, "/run/sing-helm/taildrop") {
		t.Fatalf("iOS export must not contain the desktop state/taildrop dirs")
	}
	for _, want := range []string{`"type": "tailscale"`, "ts_dns", "100.64.0.0/10"} {
		if !strings.Contains(out, want) {
			t.Errorf("iOS export lost %q", want)
		}
	}
}

// 全局代理 / 直连模式会清空大部分规则，但 tailnet 路由必须保留，
// 否则 100.x 的流量会被送去代理或从本机直接发出，保险库连不上。
func TestTailscaleRoutesSurviveGlobalAndDirectRouteModes(t *testing.T) {
	for _, rm := range []model.RouteMode{model.RouteModeGlobal, model.RouteModeDirect} {
		run := &model.RunOptions{ProxyMode: model.ProxyModeTUN, RouteMode: rm, ListenAddr: "127.0.0.1"}
		bctx := module.NewBuildContext(run)
		opts := &option.Options{}
		mods := []module.ConfigModule{
			&module.EndpointModule{StateDir: "/x"},
			&module.TUNModule{},
			&module.DNSModule{RouteMode: rm},
			&module.RouteModule{RouteMode: rm},
		}
		for _, m := range mods {
			if err := m.Apply(opts, bctx); err != nil {
				t.Fatalf("%s: module %s: %v", rm, m.Name(), err)
			}
		}
		rr, _ := singboxjson.Marshal(opts.Route.Rules)
		r := string(rr)
		if !strings.Contains(r, "100.64.0.0/10") || !strings.Contains(r, "ts.net") {
			t.Errorf("%s mode dropped tailscale routes: %s", rm, r)
		}
		if rm == model.RouteModeDirect && strings.Contains(r, "binance") {
			t.Errorf("direct mode should still clear other rules: %s", r)
		}
	}
}
