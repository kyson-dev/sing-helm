package controller

import (
	"context"
	"fmt"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
)

// SwitchProxyMode 切换代理模式
func SwitchProxyMode(modeStr string) (string, error) {
	resp, err := sendCommand(context.Background(), "mode", map[string]any{"mode": modeStr})
	if err != nil {
		return "", err
	}
	if mode, ok := resp.Data["proxy_mode"].(string); ok && mode != "" {
		logger.Debug("Proxy mode switched successfully", "mode", mode)
		return mode, nil
	}
	logger.Debug("Proxy mode switched successfully", "mode", modeStr)
	return modeStr, nil
}

// SwitchRouteMode 切换路由模式
func SwitchRouteMode(modeStr string) (string, error) {
	resp, err := sendCommand(context.Background(), "route", map[string]any{"route": modeStr})
	if err != nil {
		return "", err
	}
	if mode, ok := resp.Data["route_mode"].(string); ok && mode != "" {
		logger.Debug("Route mode switched successfully", "mode", mode)
		return mode, nil
	}
	logger.Debug("Route mode switched successfully", "mode", modeStr)
	return modeStr, nil
}

type Status struct {
	ProxyMode  string
	RouteMode  string
	ListenAddr string
	APIPort    int
	MixedPort  int
	PID        int
	Running    bool
}

func FetchStatus(ctx context.Context) (*Status, error) {
	resp, err := sendCommand(ctx, "status", nil)
	if err != nil {
		return nil, err
	}
	status := &Status{}
	if mode, ok := resp.Data["proxy_mode"].(string); ok {
		status.ProxyMode = mode
	}
	if mode, ok := resp.Data["route_mode"].(string); ok {
		status.RouteMode = mode
	}
	if addr, ok := resp.Data["listen_addr"].(string); ok {
		status.ListenAddr = addr
	}
	if port, ok := asInt(resp.Data["api_port"]); ok {
		status.APIPort = port
	}
	if port, ok := asInt(resp.Data["mixed_port"]); ok {
		status.MixedPort = port
	}
	if pid, ok := asInt(resp.Data["pid"]); ok {
		status.PID = pid
	}
	if running, ok := resp.Data["running"].(bool); ok {
		status.Running = running
	}
	return status, nil
}

func sendCommand(ctx context.Context, name string, payload map[string]any) (ipc.CommandResult, error) {
	sender := ipc.NewUnixSender(env.Get().SocketFile)
	resp, err := sender.Send(ctx, ipc.CommandMessage{Name: name, Payload: payload})
	if err != nil {
		return ipc.CommandResult{}, fmt.Errorf("ipc send failed: %w", err)
	}
	if resp.Status == "" {
		resp.Status = "ok"
	}
	if resp.Status != "ok" {
		if resp.Error != "" {
			return resp, fmt.Errorf("daemon error: %s", resp.Error)
		}
		return resp, fmt.Errorf("daemon responded with status %s", resp.Status)
	}
	return resp, nil
}

func asInt(val any) (int, bool) {
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}
