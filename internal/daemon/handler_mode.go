package daemon

import (
	"context"
	"os"

	"github.com/kyson-dev/sing-helm/internal/ipc"
	"github.com/kyson-dev/sing-helm/internal/model"
)

func (d *Daemon) handleMode(ctx context.Context, payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "sing-box not running"}
	}
	modeStr, ok := payload["mode"].(string)
	if !ok || modeStr == "" {
		return ipc.CommandResult{Status: "error", Error: "missing mode"}
	}
	proxyMode, err := model.ParseProxyMode(modeStr)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	state, err := d.currentState()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	if (proxyMode == model.ProxyModeTUN || state.RunOptions.ProxyMode == model.ProxyModeTUN) && os.Geteuid() != 0 {
		return ipc.CommandResult{Status: "error", Error: "operating with TUN mode requires root permission"}
	}
	if state.RunOptions.ProxyMode == proxyMode {
		return ipc.CommandResult{Status: "ok", Data: map[string]any{"proxy_mode": string(proxyMode)}}
	}
	state.RunOptions.ProxyMode = proxyMode
	if err := d.applyRunOptions(ctx, state); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"proxy_mode": string(proxyMode)}}
}

func (d *Daemon) handleRoute(ctx context.Context, payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "sing-box not running"}
	}
	routeStr, ok := payload["route"].(string)
	if !ok || routeStr == "" {
		return ipc.CommandResult{Status: "error", Error: "missing route"}
	}
	routeMode, err := model.ParseRouteMode(routeStr)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	state, err := d.currentState()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	if state.RunOptions.RouteMode == routeMode {
		return ipc.CommandResult{Status: "ok", Data: map[string]any{"route_mode": string(routeMode)}}
	}
	state.RunOptions.RouteMode = routeMode
	if err := d.applyRunOptions(ctx, state); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"route_mode": string(routeMode)}}
}
