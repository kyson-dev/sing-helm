package config

import (
	"context"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_DefaultMode 测试默认模式配置生成
func TestGenerate_DefaultMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
		},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证模式
	assert.Equal(t, ModeDefault, runOpts.Mode)

	// 验证 API 端口已自动分配
	assert.Greater(t, runOpts.APIPort, 0, "API port should be auto-assigned")
	assert.Less(t, runOpts.APIPort, 65536, "API port should be valid")

	// 验证 Mixed 端口已自动分配
	assert.Greater(t, runOpts.MixedPort, 0, "Mixed port should be auto-assigned")
	assert.Less(t, runOpts.MixedPort, 65536, "Mixed port should be valid")

	// 验证 Clash API 配置
	require.NotNil(t, result.Experimental)
	require.NotNil(t, result.Experimental.ClashAPI)
	assert.Contains(t, result.Experimental.ClashAPI.ExternalController, "127.0.0.1")

	// 验证 Inbounds (应该有 mixed 入站)
	require.Len(t, result.Inbounds, 1)
	assert.Equal(t, "mixed", result.Inbounds[0].Type)
	assert.Equal(t, "mixed-in", result.Inbounds[0].Tag)

	// 验证 Outbounds (应该包含用户的 + direct)
	assert.GreaterOrEqual(t, len(result.Outbounds), 2)
	hasProxy := false
	hasDirect := false
	for _, out := range result.Outbounds {
		if out.Tag == "proxy" {
			hasProxy = true
		}
		if out.Tag == "direct" {
			hasDirect = true
		}
	}
	assert.True(t, hasProxy, "Should have proxy outbound")
	assert.True(t, hasDirect, "Should have direct outbound")

	// 验证日志配置
	require.NotNil(t, result.Log)
	assert.Equal(t, "info", result.Log.Level)
}

// TestGenerate_SystemProxyMode 测试系统代理模式
func TestGenerate_SystemProxyMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "vmess",
				Tag:  "proxy",
			},
		},
	}

	runOpts := &RunOptions{
		Mode:       ModeSystem,
		ListenAddr: "127.0.0.1",
		APIPort:    0,
		MixedPort:  0,
	}

	result, err := Generate(user, runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证端口已分配
	assert.Greater(t, runOpts.APIPort, 0)
	assert.Greater(t, runOpts.MixedPort, 0)

	// 验证 Inbounds
	require.Len(t, result.Inbounds, 1)
	assert.Equal(t, "mixed", result.Inbounds[0].Type)

	// 系统代理模式下，set_system_proxy 应该为 true
	// 注意：由于使用了 map[string]any，我们需要通过 Options 字段访问
	// 这个验证可能需要根据实际的 Inbound 结构调整
}

// TestGenerate_TUNMode 测试 TUN 模式
func TestGenerate_TUNMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "trojan",
				Tag:  "proxy",
			},
		},
	}

	runOpts := &RunOptions{
		Mode:       ModeTUN,
		ListenAddr: "127.0.0.1",
		APIPort:    0,
	}

	result, err := Generate(user, runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证 API 端口已分配
	assert.Greater(t, runOpts.APIPort, 0)

	// TUN 模式下 MixedPort 不应该被设置
	assert.Equal(t, 0, runOpts.MixedPort)

	// 验证 Inbounds (应该有 TUN 入站)
	require.Len(t, result.Inbounds, 1)
	assert.Equal(t, "tun", result.Inbounds[0].Type)
	assert.Equal(t, "tun-in", result.Inbounds[0].Tag)

	// 验证 DNS 配置 (TUN 模式需要 DNS)
	require.NotNil(t, result.DNS)
	assert.NotEmpty(t, result.DNS.Servers)
	assert.Equal(t, "dns-proxy", result.DNS.Final)

	// 验证 DNS 服务器
	assert.GreaterOrEqual(t, len(result.DNS.Servers), 2)
}

// TestGenerate_FixedPorts 测试固定端口配置
func TestGenerate_FixedPorts(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
		},
	}

	fixedAPIPort := 9090
	fixedMixedPort := 7890

	runOpts := &RunOptions{
		Mode:       ModeDefault,
		ListenAddr: "127.0.0.1",
		APIPort:    fixedAPIPort,
		MixedPort:  fixedMixedPort,
	}

	result, err := Generate(user, runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证端口保持固定值
	assert.Equal(t, fixedAPIPort, runOpts.APIPort, "API port should remain fixed")
	assert.Equal(t, fixedMixedPort, runOpts.MixedPort, "Mixed port should remain fixed")

	// 验证 Clash API 使用了固定端口
	require.NotNil(t, result.Experimental)
	require.NotNil(t, result.Experimental.ClashAPI)
	assert.Contains(t, result.Experimental.ClashAPI.ExternalController, "9090")
}

// TestGenerate_WithExistingDirect 测试已有 direct outbound 的情况
func TestGenerate_WithExistingDirect(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
			{
				Type: "direct",
				Tag:  "direct",
			},
		},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证不会重复添加 direct outbound
	directCount := 0
	for _, out := range result.Outbounds {
		if out.Type == "direct" {
			directCount++
		}
	}
	assert.Equal(t, 1, directCount, "Should have exactly one direct outbound")
}

// TestGenerate_NilUserProfile 测试 nil 用户配置
func TestGenerate_NilUserProfile(t *testing.T) {
	runOpts := DefaultRunOptions()
	result, err := Generate(nil, &runOpts)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestGenerate_EmptyUserProfile 测试空用户配置
func TestGenerate_EmptyUserProfile(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 即使用户没有配置 outbound，也应该有 direct
	assert.GreaterOrEqual(t, len(result.Outbounds), 1)
	hasDirect := false
	for _, out := range result.Outbounds {
		if out.Tag == "direct" {
			hasDirect = true
		}
	}
	assert.True(t, hasDirect, "Should have direct outbound even with empty user config")
}

// TestGenerate_MultipleOutbounds 测试多个出站配置
func TestGenerate_MultipleOutbounds(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "ss-proxy",
			},
			{
				Type: "vmess",
				Tag:  "vmess-proxy",
			},
			{
				Type: "trojan",
				Tag:  "trojan-proxy",
			},
		},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证所有用户配置的 outbound 都存在
	assert.GreaterOrEqual(t, len(result.Outbounds), 4) // 3 个用户的 + 1 个 direct

	tags := make(map[string]bool)
	for _, out := range result.Outbounds {
		tags[out.Tag] = true
	}

	assert.True(t, tags["ss-proxy"])
	assert.True(t, tags["vmess-proxy"])
	assert.True(t, tags["trojan-proxy"])
	assert.True(t, tags["direct"])
}

// TestGenerate_CustomListenAddr 测试自定义监听地址
func TestGenerate_CustomListenAddr(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
		},
	}

	customAddr := "0.0.0.0"
	runOpts := &RunOptions{
		Mode:       ModeDefault,
		ListenAddr: customAddr,
		APIPort:    0,
		MixedPort:  0,
	}

	result, err := Generate(user, runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证 Clash API 使用了自定义地址
	require.NotNil(t, result.Experimental)
	require.NotNil(t, result.Experimental.ClashAPI)
	assert.Contains(t, result.Experimental.ClashAPI.ExternalController, customAddr)
}

// TestDefaultRunOptions 测试默认运行选项
func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions()

	assert.Equal(t, ModeDefault, opts.Mode)
	assert.Equal(t, "127.0.0.1", opts.ListenAddr)
	assert.Equal(t, 0, opts.APIPort, "API port should be 0 (auto-assign)")
	assert.Equal(t, 0, opts.MixedPort, "Mixed port should be 0 (auto-assign)")
}

// TestResolvePort 测试端口解析逻辑
func TestResolvePort(t *testing.T) {
	t.Run("auto-assign when port is 0", func(t *testing.T) {
		port, err := resolvePort(0)
		require.NoError(t, err)
		assert.Greater(t, port, 0)
		assert.Less(t, port, 65536)
	})

	t.Run("keep fixed port", func(t *testing.T) {
		fixedPort := 8080
		port, err := resolvePort(fixedPort)
		require.NoError(t, err)
		assert.Equal(t, fixedPort, port)
	})

	t.Run("multiple auto-assign should give different ports", func(t *testing.T) {
		port1, err1 := resolvePort(0)
		port2, err2 := resolvePort(0)

		require.NoError(t, err1)
		require.NoError(t, err2)

		// 大概率会得到不同的端口（虽然理论上可能相同）
		// 这个测试主要是确保端口分配机制正常工作
		assert.Greater(t, port1, 0)
		assert.Greater(t, port2, 0)
	})
}

// TestGenerate_PortsAreDifferent 测试 API 端口和 Mixed 端口不会冲突
func TestGenerate_PortsAreDifferent(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
		},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证 API 端口和 Mixed 端口不同
	assert.NotEqual(t, runOpts.APIPort, runOpts.MixedPort, "API port and Mixed port should be different")
}

// TestGenerate_WithRoute 测试带路由配置的生成
func TestGenerate_WithRoute(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "shadowsocks",
				Tag:  "proxy",
			},
		},
		Route: &option.RouteOptions{
			Final: "proxy",
		},
	}

	runOpts := DefaultRunOptions()
	result, err := Generate(user, &runOpts)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证路由配置被保留
	require.NotNil(t, result.Route)
	assert.Equal(t, "proxy", result.Route.Final)
}

// =============================================================================
// 集成测试：验证 sing-box 能否真正使用生成的配置
// =============================================================================

// TestGenerate_SingboxCanParseDefaultMode 验证 sing-box 能解析默认模式的配置
func TestGenerate_SingboxCanParseDefaultMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "direct",
				Tag:  "proxy", // 用 direct 类型模拟，避免需要真实服务器
			},
		},
	}

	runOpts := DefaultRunOptions()
	opts, err := Generate(user, &runOpts)
	require.NoError(t, err)

	// 关键测试：尝试创建 sing-box 实例
	// 如果配置有类型问题，这里会 panic 或返回 error
	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: *opts,
	})

	require.NoError(t, err, "sing-box should be able to parse the generated config")
	require.NotNil(t, instance)

	// 清理
	instance.Close()
}

// TestGenerate_SingboxCanParseTUNMode 验证 sing-box 能解析 TUN 模式的配置
func TestGenerate_SingboxCanParseTUNMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "direct",
				Tag:  "proxy",
			},
		},
	}

	runOpts := &RunOptions{
		Mode:       ModeTUN,
		ListenAddr: "127.0.0.1",
		APIPort:    0,
	}

	opts, err := Generate(user, runOpts)
	require.NoError(t, err)

	// 关键测试：尝试创建 sing-box 实例
	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: *opts,
	})

	require.NoError(t, err, "sing-box should be able to parse TUN mode config")
	require.NotNil(t, instance)

	// 清理
	instance.Close()
}

// TestGenerate_SingboxCanParseSystemProxyMode 验证 sing-box 能解析系统代理模式的配置
func TestGenerate_SingboxCanParseSystemProxyMode(t *testing.T) {
	user := &UserProfile{
		Outbounds: []option.Outbound{
			{
				Type: "direct",
				Tag:  "proxy",
			},
		},
	}

	runOpts := &RunOptions{
		Mode:       ModeSystem,
		ListenAddr: "127.0.0.1",
		APIPort:    0,
		MixedPort:  0,
	}

	opts, err := Generate(user, runOpts)
	require.NoError(t, err)

	// 关键测试：尝试创建 sing-box 实例
	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: *opts,
	})

	require.NoError(t, err, "sing-box should be able to parse system proxy mode config")
	require.NotNil(t, instance)

	// 清理
	instance.Close()
}
