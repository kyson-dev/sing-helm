package config

import (
	"context"
	"strings"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// 用户在 profile.json 里声明 Tailscale endpoint、tailscale 类型的 DNS 服务器，
// 以及对应的 DNS / 路由规则。这些测试确认现有的 TUN、DNS、Route 模块不会丢掉
// 或打乱它们（不需要 with_tailscale 标签，解析阶段不检查标签）。
//
// 刻意不调用 paths.Init（它会探测真实的 /var/log/sing-helm），而是模拟
// TemplateModule 的做法，直接把 profile 读进 Options。
const tailscaleProfile = `{
  "endpoints": [
    {"type": "tailscale", "tag": "ts", "state_directory": "/usr/local/var/run/sing-helm/tailscale"}
  ],
  "dns": {
    "servers": [{"type": "tailscale", "tag": "ts_dns", "endpoint": "ts"}],
    "rules": [{"domain_suffix": ["ts.net"], "server": "ts_dns"}]
  },
  "route": {
    "rules": [
      {"ip_cidr": ["100.64.0.0/10"], "outbound": "ts"},
      {"domain_suffix": ["ts.net"], "outbound": "ts"}
    ]
  }
}`

func buildWithTailscaleProfile(t *testing.T, mode model.ProxyMode) *option.Options {
	t.Helper()
	ctx := include.Context(context.Background())
	opts := &option.Options{}
	if err := singboxjson.UnmarshalContext(ctx, []byte(tailscaleProfile), opts); err != nil {
		t.Fatalf("profile should parse: %v", err)
	}
	run := &model.RunOptions{ProxyMode: mode, RouteMode: model.RouteModeRule, ListenAddr: "127.0.0.1"}
	bctx := module.NewBuildContext(run)
	mods := []module.ConfigModule{
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

func TestTailscaleProfileSurvivesModules(t *testing.T) {
	opts := buildWithTailscaleProfile(t, model.ProxyModeTUN)

	if len(opts.Endpoints) != 1 || opts.Endpoints[0].Tag != "ts" {
		t.Fatalf("endpoint lost: %+v", opts.Endpoints)
	}
	foundDNS := false
	for _, s := range opts.DNS.Servers {
		if s.Tag == "ts_dns" {
			foundDNS = true
		}
	}
	if !foundDNS {
		t.Fatal("tailscale DNS server dropped")
	}

	// 用户 DNS 规则必须排在 fake-ip 的 A/AAAA 规则之前，否则 *.ts.net 会拿到假地址。
	raw, _ := singboxjson.Marshal(opts.DNS.Rules)
	s := string(raw)
	iTS := strings.Index(s, "ts_dns")
	iFake := strings.Index(s, "dns_fakeip")
	if iTS < 0 || iFake < 0 || iTS > iFake {
		t.Fatalf("ts.net DNS rule must precede fake-ip rule (ts=%d fake=%d)", iTS, iFake)
	}

	// 用户路由规则必须在 sniff/hijack-dns 之后、保底规则（含 ip_is_private/geoip）之前。
	rr, _ := singboxjson.Marshal(opts.Route.Rules)
	r := string(rr)
	iSniff := strings.Index(r, `"sniff"`)
	iCGNAT := strings.Index(r, "100.64.0.0/10")
	iPrivate := strings.Index(r, "ip_is_private")
	if !(iSniff >= 0 && iSniff < iCGNAT && iCGNAT < iPrivate) {
		t.Fatalf("route rule order wrong: sniff=%d cgnat=%d private=%d", iSniff, iCGNAT, iPrivate)
	}
}
