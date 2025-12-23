package config

import (
	"context"
	"fmt"
	"net"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/pkg/netutil"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// Mode 代理模式
type Mode string

const (
	ModeTUN     Mode = "tun"     // TUN 虚拟网卡模式 (需要 root)
	ModeSystem  Mode = "system"  // 系统代理模式
	ModeDefault Mode = "default" // 仅开端口，手动配置代理
)

// RunOptions 运行时参数
type RunOptions struct {
	Mode       Mode   `json:"mode"`                 // 代理模式
	APIPort    int    `json:"api_port"`             // Clash API 端口，0 表示自动获取
	MixedPort  int    `json:"mixed_port,omitempty"` // Mixed 入站端口，0 表示自动获取
	ListenAddr string `json:"listen_addr,omitempty"`
}

// DefaultRunOptions 返回默认运行参数
func DefaultRunOptions() RunOptions {
	return RunOptions{
		Mode:       ModeDefault,
		ListenAddr: "127.0.0.1",
	}
}

// Generate 基于用户配置 + 运行时参数，生成最终的 sing-box 核心配置
// 返回 *option.Options 和实际使用的端口信息 (通过 state.go 保存)
func Generate(user *UserProfile, opts *RunOptions) (*option.Options, error) {
	if user == nil {
		return nil, fmt.Errorf("user profile cannot be nil")
	}

	result := &option.Options{}

	// 保留的 tag 名称，用户不能使用这些名称
	reservedTags := map[string]bool{
		"direct": true,
		"block":  true,
		"proxy":  true,
	}

	// 1. 过滤并复制用户节点（排除保留 tag）
	var userNodeTags []string
	for _, out := range user.Outbounds {
		if reservedTags[out.Tag] {
			// 忽略用户配置的保留 tag
			logger.Info("Ignoring reserved outbound tag from user config", "tag", out.Tag)
			continue
		}
		result.Outbounds = append(result.Outbounds, out)
		userNodeTags = append(userNodeTags, out.Tag)
	}

	// 2. 生成程序默认的 outbound (direct, block, proxy)
	if err := generateDefaultOutbounds(result, userNodeTags); err != nil {
		return nil, fmt.Errorf("failed to generate default outbounds: %w", err)
	}

	// 3. 复制用户路由
	result.Route = user.Route

	var err error

	// 4. 自动分配 API 端口
	if opts.APIPort, err = resolvePort(opts.APIPort); err != nil {
		return nil, fmt.Errorf("failed to allocate API port: %w", err)
	}

	// 5. 配置 Experimental (Clash API)
	result.Experimental = &option.ExperimentalOptions{
		ClashAPI: &option.ClashAPIOptions{
			ExternalController: net.JoinHostPort(opts.ListenAddr, fmt.Sprintf("%d", opts.APIPort)),
		},
	}

	// 6. 根据模式生成 Inbounds 和 DNS
	switch opts.Mode {
	case ModeTUN:
		generateTUN(result)
	case ModeSystem:
		if opts.MixedPort, err = resolvePort(opts.MixedPort); err != nil {
			return nil, fmt.Errorf("failed to allocate mixed port: %w", err)
		}
		generateMixed(result, opts.ListenAddr, opts.MixedPort, true)
	default:
		if opts.MixedPort, err = resolvePort(opts.MixedPort); err != nil {
			return nil, fmt.Errorf("failed to allocate mixed port: %w", err)
		}
		generateMixed(result, opts.ListenAddr, opts.MixedPort, false)
	}

	// 7. 配置日志
	result.Log = &option.LogOptions{
		Level: "info",
	}

	return result, nil
}

// resolvePort 解析端口，如果为 0 则自动获取空闲端口
func resolvePort(port int) (int, error) {
	if port > 0 {
		return port, nil
	}
	return netutil.GetFreePort()
}

// generateDefaultOutbounds 生成程序默认的 outbound (direct, block, proxy)
// proxy 是一个 selector，包含所有用户配置的节点
func generateDefaultOutbounds(opts *option.Options, userNodeTags []string) error {
	ctx := include.Context(context.Background())

	// 1. 生成 direct outbound
	directMap := map[string]any{
		"type": "direct",
		"tag":  "direct",
	}
	directData, err := singboxjson.Marshal(directMap)
	if err != nil {
		return fmt.Errorf("failed to marshal direct outbound: %w", err)
	}
	var direct option.Outbound
	if err := singboxjson.UnmarshalContext(ctx, directData, &direct); err != nil {
		return fmt.Errorf("failed to unmarshal direct outbound: %w", err)
	}

	// 2. 生成 block outbound
	blockMap := map[string]any{
		"type": "block",
		"tag":  "block",
	}
	blockData, err := singboxjson.Marshal(blockMap)
	if err != nil {
		return fmt.Errorf("failed to marshal block outbound: %w", err)
	}
	var block option.Outbound
	if err := singboxjson.UnmarshalContext(ctx, blockData, &block); err != nil {
		return fmt.Errorf("failed to unmarshal block outbound: %w", err)
	}

	// 3. 生成 proxy outbound (selector 类型，包含所有用户节点)
	// 如果没有用户节点，proxy 回落到 direct
	proxyOutbounds := userNodeTags
	if len(proxyOutbounds) == 0 {
		proxyOutbounds = []string{"direct"}
	}
	proxyMap := map[string]any{
		"type":      "selector",
		"tag":       "proxy",
		"outbounds": proxyOutbounds,
	}
	proxyData, err := singboxjson.Marshal(proxyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal proxy outbound: %w", err)
	}
	var proxy option.Outbound
	if err := singboxjson.UnmarshalContext(ctx, proxyData, &proxy); err != nil {
		return fmt.Errorf("failed to unmarshal proxy outbound: %w", err)
	}

	// 按顺序添加：direct, block, proxy
	opts.Outbounds = append(opts.Outbounds, direct, block, proxy)

	return nil
}

// generateTUN 生成 TUN 模式配置
// 注意：macOS 上 TUN 网卡名称由系统自动分配，不能指定
func generateTUN(opts *option.Options) {
	// 使用 JSON 方式构建 TUN 入站配置
	tunMap := map[string]any{
		"type":         "tun",
		"tag":          "tun-in",
		"address":      []string{"172.19.0.1/30"},
		"auto_route":   true,
		"strict_route": true,
		"stack":        "mixed", // TCP 用 system 性能好，UDP 用 gvisor 兼容性好
		"mtu":          9000,
	}

	data, err := singboxjson.Marshal(tunMap)
	if err != nil {
		logger.Error("failed to marshal TUN config", "error", err)
		return
	}

	var tun option.Inbound
	ctx := include.Context(context.Background())
	err = singboxjson.UnmarshalContext(ctx, data, &tun)
	if err != nil {
		logger.Error("failed to unmarshal TUN config", "error", err)
		return
	}

	opts.Inbounds = append(opts.Inbounds, tun)

	// TUN 模式需要 DNS 配置
	// 关键点：dns-proxy 的代理出站需要先解析代理服务器 IP
	// 使用直连 DNS 解析，避免死循环
	//
	// 解析流程：
	//   1. 应用请求 google.com -> dns-proxy (走代理)
	//   2. 代理出站需解析代理服务器域名 -> dns-direct (直连解析)
	//   3. dns-direct 解析完成 -> 代理连接建立
	//   4. dns-proxy 通过代理查询 -> 返回结果
	dnsMap := map[string]any{
		"servers": []map[string]any{
			{
				"tag":    "dns-direct",
				"type":   "udp",
				"server": "223.5.5.5",
				// 不指定 detour，默认使用直连
			},
			{
				"tag":    "dns-proxy",
				"type":   "udp",
				"server": "8.8.8.8",
				"detour": "proxy",
			},
		},
		"final": "dns-proxy",
	}

	dnsData, err := singboxjson.Marshal(dnsMap)
	if err != nil {
		logger.Error("failed to marshal DNS config", "error", err)
		return
	}

	var dns option.DNSOptions
	err = singboxjson.UnmarshalContext(ctx, dnsData, &dns)
	if err != nil {
		logger.Error("failed to unmarshal DNS config", "error", err)
		return
	}

	opts.DNS = &dns
}

// generateMixed 生成 Mixed (HTTP/SOCKS) 入站
func generateMixed(opts *option.Options, listenAddr string, port int, setSystemProxy bool) {
	// 使用 map 构建配置，然后通过 JSON 转换为正确的类型
	// 这样可以避免复杂的类型转换问题
	inboundMap := map[string]any{
		"type":             "mixed",
		"tag":              "mixed-in",
		"listen":           listenAddr,
		"listen_port":      port,
		"set_system_proxy": setSystemProxy,
	}

	// 通过 JSON 序列化和反序列化来创建正确的 Inbound 结构
	// 使用 sing-box 的 JSON 包，支持 context
	data, err := singboxjson.Marshal(inboundMap)
	if err != nil {
		// 这不应该失败，因为我们控制输入
		logger.Error("failed to marshal inbound config: %v", err)
	}

	var mixed option.Inbound
	// 使用 include.Context() 初始化带有 sing-box 注册表的 context
	// 这样 sing-box 可以正确解析 inbound 类型
	ctx := include.Context(context.Background())
	err = singboxjson.UnmarshalContext(ctx, data, &mixed)
	if err != nil {
		logger.Error("failed to unmarshal inbound config: %v", err)
	}

	opts.Inbounds = append(opts.Inbounds, mixed)
}
