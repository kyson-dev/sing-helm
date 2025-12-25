package config

import (
	"fmt"
)

// ProxyMode 代理模式（如何捕获流量）
type ProxyMode string

const (
	ProxyModeTUN     ProxyMode = "tun"     // TUN 虚拟网卡模式 (需要 root)
	ProxyModeSystem  ProxyMode = "system"  // 系统代理模式
	ProxyModeDefault ProxyMode = "default" // 仅开端口，手动配置代理
)

// RouteMode 路由模式（如何路由流量）
type RouteMode string

const (
	RouteModeRule   RouteMode = "rule"   // 按规则路由（默认）
	RouteModeGlobal RouteMode = "global" // 全局代理
	RouteModeDirect RouteMode = "direct" // 全局直连
)

// RunOptions 运行时参数
type RunOptions struct {
	ProxyMode  ProxyMode `json:"proxy_mode"`            // 代理模式
	RouteMode  RouteMode `json:"route_mode,omitempty"`  // 路由模式
	APIPort    int       `json:"api_port"`              // Clash API 端口，0 表示自动获取
	MixedPort  int       `json:"mixed_port,omitempty"`  // Mixed 入站端口，0 表示自动获取
	ListenAddr string    `json:"listen_addr,omitempty"` // 监听地址
}

// DefaultRunOptions 返回默认运行参数
func DefaultRunOptions() RunOptions {
	return RunOptions{
		ProxyMode:  ProxyModeSystem,
		RouteMode:  RouteModeRule,
		ListenAddr: "127.0.0.1",
	}
}

// ParseProxyMode 解析代理模式字符串
func ParseProxyMode(s string) (ProxyMode, error) {
	switch s {
	case "system":
		return ProxyModeSystem, nil
	case "tun":
		return ProxyModeTUN, nil
	case "default", "":
		return ProxyModeDefault, nil
	default:
		return "", fmt.Errorf("invalid proxy mode: %s", s)
	}
}

// ParseRouteMode 解析路由模式字符串
func ParseRouteMode(s string) (RouteMode, error) {
	switch s {
	case "rule", "":
		return RouteModeRule, nil
	case "global":
		return RouteModeGlobal, nil
	case "direct":
		return RouteModeDirect, nil
	default:
		return "", fmt.Errorf("invalid route mode: %s", s)
	}
}
