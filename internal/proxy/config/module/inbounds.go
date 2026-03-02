package module

import (
	"fmt"
	"net/netip"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

// MixedModule Mixed 入站模块
// 支持设置系统代理
type MixedModule struct {
	SetSystemProxy bool
	ListenAddr     string
	Port           int
}

func (m *MixedModule) Name() string {
	return "mixed"
}

func (m *MixedModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// mixed 模式下，先清理 tun 相关入站，避免模式切换后冲突残留。
	opts.Inbounds = filterInbounds(opts.Inbounds, func(in option.Inbound) bool {
		return !isTUNInbound(in)
	})

	// 如用户已有 mixed 入站，复用并强制修正 set_system_proxy，然后回填 RunOptions。
	for _, in := range opts.Inbounds {
		if isMixedInbound(in) {
			if mixedOpts, ok := in.Options.(*option.HTTPMixedInboundOptions); ok {
				mixedOpts.SetSystemProxy = m.SetSystemProxy
				if err := backfillRunOptionsFromMixed(ctx, mixedOpts); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid mixed inbound options type for tag %q", in.Tag)
			}
			return nil
		}
	}

	// 仅在需要新建 mixed 时，才解析默认监听参数。
	resolvedListenAddr := m.ListenAddr
	if resolvedListenAddr == "" {
		resolvedListenAddr = "127.0.0.1"
	}
	resolvedPort := m.Port
	if resolvedPort == 0 {
		var err error
		resolvedPort, err = moduleUtils.GetFreePort()
		if err != nil {
			return err
		}
	}

	// 更新 context 中的端口信息
	backfillRunOptionsFromValues(ctx, resolvedListenAddr, resolvedPort)

	// 创建 Mixed 入站配置
	mixedInbound := option.Inbound{}
	mixedMap := map[string]any{
		"type":             "mixed",
		"tag":              "mixed-in",
		"listen":           resolvedListenAddr,
		"listen_port":      resolvedPort,
		"set_system_proxy": m.SetSystemProxy,
	}
	if err := moduleUtils.ApplyMapToInbound(&mixedInbound, mixedMap); err != nil {
		return err
	}

	// 添加到配置
	opts.Inbounds = append(opts.Inbounds, mixedInbound)

	return nil
}

// TUNModule TUN 入站模块
type TUNModule struct {
	MTU   int
	Stack string
}

func (m *TUNModule) Name() string {
	return "tun"
}

func (m *TUNModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// tun 模式下，先清理 mixed/socks/http，避免模式切换后冲突残留。
	opts.Inbounds = filterInbounds(opts.Inbounds, func(in option.Inbound) bool {
		return !isMixedLikeInbound(in)
	})

	// 如果用户已经在 profile 中配了 tun 设备入站，复用并回填 RunOptions。
	for _, in := range opts.Inbounds {
		if isTUNInbound(in) {
			return nil
		}
	}

	// 默认值
	mtu := m.MTU
	if mtu == 0 {
		mtu = 1500
	}

	stack := m.Stack
	if stack == "" {
		stack = "mixed" // mixed 兼顾性能和兼容性
	}

	// 创建 TUN 入站配置
	tunInbound := option.Inbound{}
	tunMap := map[string]any{
		"type":         "tun",
		"tag":          "tun-in",
		"mtu":          mtu,
		"auto_route":   true,
		"strict_route": true,
		//"stack":                      stack,
		"address": []string{"172.19.0.1/30"},
		//"inet6_address":              "fd00::1/126",
		"sniff":                      true,
		"sniff_override_destination": true,
	}
	if err := moduleUtils.ApplyMapToInbound(&tunInbound, tunMap); err != nil {
		return err
	}

	// 添加到配置
	opts.Inbounds = append(opts.Inbounds, tunInbound)

	return nil
}

func filterInbounds(inbounds []option.Inbound, keep func(option.Inbound) bool) []option.Inbound {
	filtered := make([]option.Inbound, 0, len(inbounds))
	for _, in := range inbounds {
		if keep(in) {
			filtered = append(filtered, in)
		}
	}
	return filtered
}

func isTUNInbound(in option.Inbound) bool {
	return in.Type == "tun" || in.Tag == "tun-in"
}

func isMixedInbound(in option.Inbound) bool {
	return in.Type == "mixed" || in.Tag == "mixed-in"
}

func isMixedLikeInbound(in option.Inbound) bool {
	return in.Type == "mixed" || in.Type == "socks" || in.Type == "http" || in.Tag == "mixed-in"
}

func backfillRunOptionsFromValues(ctx *BuildContext, listenAddr string, port int) {
	if ctx == nil || ctx.RunOptions == nil {
		return
	}
	if listenAddr != "" {
		ctx.RunOptions.ListenAddr = listenAddr
	}
	if port > 0 {
		ctx.RunOptions.MixedPort = port
	}
}

func backfillRunOptionsFromMixed(ctx *BuildContext, mixedOpts *option.HTTPMixedInboundOptions) error {
	if mixedOpts == nil {
		return fmt.Errorf("mixed inbound options is nil")
	}
	if mixedOpts.Listen == nil {
		return fmt.Errorf("user mixed inbound requires explicit listen")
	}
	if mixedOpts.ListenPort == 0 {
		return fmt.Errorf("user mixed inbound requires explicit listen_port")
	}
	
	backfillRunOptionsFromValues(ctx, netip.Addr(*mixedOpts.Listen).String(), int(mixedOpts.ListenPort))
	return nil
}
