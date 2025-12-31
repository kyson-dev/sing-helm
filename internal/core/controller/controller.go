package controller

import (
	"context"
	"fmt"

	"os"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
)

// SwitchProxyMode 切换代理模式
func SwitchProxyMode(modeStr string) (string, error) {
	// 1. 验证模式
	proxyMode, err := config.ParseProxyMode(modeStr)
	if err != nil {
		return "", err
	}

	// 2. 加载状态
	state, err := config.LoadState()
	if err != nil {
		return "", fmt.Errorf("daemon not running: %w", err)
	}

	// 3. 检查是否需要切换
	if state.ProxyMode == proxyMode {
		return string(state.ProxyMode), nil // 已经处于该模式
	}

	// 4. 权限检查：涉及 TUN 模式的操作需要 root 权限
	if (proxyMode == config.ProxyModeTUN || state.ProxyMode == config.ProxyModeTUN) && os.Geteuid() != 0 {
		return "", fmt.Errorf("operating with TUN mode requires root permission")
	}

	// 5. 更新状态
	state.ProxyMode = proxyMode
	if err := applyConfigAndReload(state); err != nil {
		return "", err
	}

	logger.Debug("Proxy mode switched successfully", "mode", proxyMode)
	return string(proxyMode), nil
}

// SwitchRouteMode 切换路由模式
func SwitchRouteMode(modeStr string) (string, error) {
	// 1. 验证模式
	routeMode, err := config.ParseRouteMode(modeStr)
	if err != nil {
		return "", err
	}

	// 2. 加载状态
	state, err := config.LoadState()
	if err != nil {
		return "", fmt.Errorf("daemon not running: %w", err)
	}

	// 3. 检查是否需要切换
	if state.RouteMode == routeMode {
		return string(state.RouteMode), nil
	}

	// 4. 更新状态
	state.RouteMode = routeMode
	if err := applyConfigAndReload(state); err != nil {
		return "", err
	}

	logger.Debug("Route mode switched successfully", "mode", routeMode)
	return string(routeMode), nil
}

// applyConfigAndReload 重新生成配置并通知 daemon 重载
func applyConfigAndReload(state *config.RuntimeState) error {
	// 1. 加载用户配置
	base, err := config.LoadOptions(env.Get().ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// 2. 重新构建配置
	builder := config.NewConfigBuilder(base, &state.RunOptions)
	for _, m := range config.DefaultModules(&state.RunOptions) {
		builder.With(m)
	}

	// 3. 保存 raw.json
	if err := builder.SaveToFile(env.Get().RawConfigFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// 4. 保存状态
	if err := config.SaveState(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// 5. 通知 daemon
	ipcClient := ipc.NewClient(env.Get().SocketFile)
	ctx := context.Background()
	if err := ipcClient.Call(ctx, ipc.MethodReload, nil, nil); err != nil {
		return fmt.Errorf("failed to reload: %w", err)
	}

	return nil
}
