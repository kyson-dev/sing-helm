package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kyson-dev/sing-helm/internal/core/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/engine"
	"github.com/kyson-dev/sing-helm/internal/sys/env"
	"github.com/kyson-dev/sing-helm/internal/sys/ipc"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
)

// handleRun 处理 IPC run 命令，启动 sing-box 服务
func (d *Daemon) handleRun(ctx context.Context, payload map[string]any) ipc.CommandResult {
	runops, err := d.parseRunOptions(payload)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}

	// 检查并设置运行状态（原子操作）
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return ipc.CommandResult{Status: "error", Error: "sing-box is already running"}
	}
	// 立即设置为 running，防止并发请求
	d.running = true
	d.mu.Unlock()

	// 如果后续启动失败，需要重置 running 状态
	startFailed := true
	defer func() {
		if startFailed {
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
		}
	}()

	// 1. 构建配置
	logger.Info("Building configuration", "mode", runops.ProxyMode, "route", runops.RouteMode)
	if err := engine.BuildConfig(env.Get().RawConfigFile, &runops); err != nil {
		return ipc.CommandResult{Status: "error", Error: fmt.Errorf("failed to build config: %w", err).Error()}
	}

	// 2. 启动 sing-box 服务
	svc := d.newService()
	rawPath := env.Get().RawConfigFile
	logger.Info("Starting sing-box", "config", rawPath)
	if err := svc.StartFromFile(ctx, rawPath); err != nil {
		return ipc.CommandResult{Status: "error", Error: fmt.Errorf("failed to start sing-box: %w", err).Error()}
	}

	// 启动成功，更新状态
	startFailed = false
	d.mu.Lock()
	d.service = svc
	if d.state == nil {
		d.state = &model.RuntimeState{}
	}
	d.state.RunOptions = runops
	d.mu.Unlock()

	logger.Info("Sing-box started successfully")
	return ipc.CommandResult{Status: "ok", Data: map[string]any{
		"proxy_mode": string(runops.ProxyMode),
		"route_mode": string(runops.RouteMode),
	}}
}

// parseRunOptions 解析 run 命令的参数
func (d *Daemon) parseRunOptions(payload map[string]any) (model.RunOptions, error) {
	runops := model.DefaultRunOptions()
	d.mu.Lock()
	state := d.state
	d.mu.Unlock()
	if state != nil {
		logger.Info("Using state from file", "proxy_mode", state.RunOptions.ProxyMode, "route_mode", state.RunOptions.RouteMode)
		runops = state.RunOptions
	} else {
		logger.Info("No state file, using defaults")
	}
	if payload == nil {
		return runops, nil
	}
	if mode, ok := payload["mode"].(string); ok && mode != "" {
		proxyMode, err := model.ParseProxyMode(mode)
		if err != nil {
			return runops, err
		}
		runops.ProxyMode = proxyMode
	}
	if route, ok := payload["route"].(string); ok && route != "" {
		routeMode, err := model.ParseRouteMode(route)
		if err != nil {
			return runops, err
		}
		runops.RouteMode = routeMode
	}
	if port, ok := ipc.AsInt(payload["api_port"]); ok && port > 0 {
		runops.APIPort = port
	}
	if port, ok := ipc.AsInt(payload["mixed_port"]); ok && port > 0 {
		runops.MixedPort = port
	}
	return runops, nil
}

// applyRunOptions 重新构建配置并 reload sing-box
func (d *Daemon) applyRunOptions(ctx context.Context, state *model.RuntimeState) error {
	// 检查并设置 reloading 标志，防止并发 reload
	d.mu.Lock()
	if d.reloading {
		d.mu.Unlock()
		return errors.New("reload already in progress")
	}
	d.reloading = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.reloading = false
		d.mu.Unlock()
	}()

	backupPath, _ := backupConfig(env.Get().RawConfigFile)
	if err := engine.BuildConfig(env.Get().RawConfigFile, &state.RunOptions); err != nil {
		return err
	}
	if d.service == nil {
		err := errors.New("service not available")
		return err
	}
	if err := d.service.ReloadFromFile(ctx, env.Get().RawConfigFile); err != nil {
		var reloadErr *engine.ReloadError
		if errors.As(err, &reloadErr) && reloadErr.Stage == engine.ReloadStageStart {
			if backupPath != "" {
				if retryErr := d.service.StartFromFile(ctx, backupPath); retryErr == nil {
					if restoreErr := restoreConfig(backupPath, env.Get().RawConfigFile); restoreErr != nil {
						return restoreErr
					}
					d.setRunning(true)
					_ = os.Remove(backupPath)
				} else {
					d.setRunning(false)
				}
			} else {
				d.setRunning(false)
			}
		}
		return err
	}
	d.mu.Lock()
	d.state = state
	d.mu.Unlock()
	return nil
}

func backupConfig(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	backup := path + ".bak"
	input, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(backup, input, 0644); err != nil {
		return "", err
	}
	return backup, nil
}

func restoreConfig(backupPath, targetPath string) error {
	if backupPath == "" || targetPath == "" {
		return nil
	}
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0644)
}
