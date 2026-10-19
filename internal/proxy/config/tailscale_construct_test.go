//go:build with_tailscale

package config

import (
	"context"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

// 只构造不启动：确认整份生成配置（含 tailscale endpoint 和 tailscale DNS 服务器）
// 能被 sing-box 接受。没有 with_tailscale 标签时 box.New 会报
// "Tailscale is not included in this build"，所以这个测试只在带标签时编译
// （make test 默认不带标签，不受影响）。
func TestTailscaleProfileConstructs(t *testing.T) {
	opts := buildWithTailscaleProfile(t, model.ProxyModeTUN)
	// 模块之外由 OutboundModule 提供的出站，这里用 direct 占位。
	opts.Outbounds = append(opts.Outbounds,
		option.Outbound{Type: "direct", Tag: "direct"},
		option.Outbound{Type: "direct", Tag: "proxy"},
	)
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	b, err := box.New(box.Options{Context: ctx, Options: *opts})
	if err != nil {
		t.Fatalf("box.New: %v", err)
	}
	_ = b // 故意不 Start：不创建网卡，不改路由
}

// 内置模块生成的整份配置（TUN + DNS + Route + Tailscale）同样能被 sing-box 接受。
func TestTailscaleModuleConstructs(t *testing.T) {
	opts := buildWithTailscaleModule(t, "", t.TempDir())
	opts.Outbounds = append(opts.Outbounds,
		option.Outbound{Type: "direct", Tag: "direct"},
		option.Outbound{Type: "direct", Tag: "proxy"},
	)
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	b, err := box.New(box.Options{Context: ctx, Options: *opts})
	if err != nil {
		t.Fatalf("box.New: %v", err)
	}
	_ = b // 故意不 Start
}
