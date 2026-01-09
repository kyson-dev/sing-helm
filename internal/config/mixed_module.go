package config

import (
	"github.com/kyson-dev/sing-helm/internal/pkg/netutil"
	"github.com/sagernet/sing-box/option"
)

const testMixedPortEnv = "MINIBOX_TEST_MIXED_PORT"

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
	// 确定监听地址
	listenAddr := m.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	// 确定端口
	port := m.Port
	if port == 0 {
		if override, ok := getPortOverride(testMixedPortEnv); ok {
			port = override
		} else {
			var err error
			port, err = netutil.GetFreePort()
			if err != nil {
				return err
			}
		}
	}

	// 更新 context 中的端口信息
	ctx.RunOptions.MixedPort = port
	ctx.RunOptions.ListenAddr = listenAddr

	// 创建 Mixed 入站配置
	mixedInbound := option.Inbound{}
	mixedMap := map[string]any{
		"type":             "mixed",
		"tag":              "mixed-in",
		"listen":           listenAddr,
		"listen_port":      port,
		"set_system_proxy": m.SetSystemProxy,
	}
	applyMapToInbound(&mixedInbound, mixedMap)

	// 添加到配置
	opts.Inbounds = append(opts.Inbounds, mixedInbound)

	return nil
}
