package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/client"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/core/runtime"
	"github.com/kyson/minibox/internal/core/service"
	"github.com/kyson/minibox/internal/core/updater"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
)

// Daemon handles long-running sing-box operations and responds to IPC commands.
type ServiceRunner interface {
	StartFromFile(context.Context, string) error
	ReloadFromFile(context.Context, string) error
}

type Daemon struct {
	mu             sync.Mutex
	cancelFunc     context.CancelFunc // 用于取消 daemon context
	service        ServiceRunner
	serviceFactory func() ServiceRunner
	lock           *env.DaemonLock
	running        bool
	state          *config.RuntimeState
}

// NewDaemon builds a daemon controller.
func NewDaemon() *Daemon {
	return &Daemon{
		serviceFactory: func() ServiceRunner {
			return service.NewInstance()
		},
	}
}

// SetServiceFactory overrides the service factory (useful for tests).
func (d *Daemon) SetServiceFactory(factory func() ServiceRunner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if factory == nil {
		d.serviceFactory = func() ServiceRunner {
			return service.NewInstance()
		}
		return
	}
	d.serviceFactory = factory
}

// Serve starts the IPC server. Blocks until ctx is cancelled.
// Use "run" IPC command to start sing-box service.
func (d *Daemon) Serve(ctx context.Context) error {
	// 检查是否已有实例在运行（通过尝试获取锁）
	lock, err := env.AcquireLock(env.Get().HomeDir)
	if err != nil {
		return fmt.Errorf("another instance is already running: %w", err)
	}
	d.lock = lock
	d.loadState()

	// 创建可取消的 context，用于控制所有子服务的生命周期
	ctx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel
	defer func() {
		logger.Info("Daemon shutting down defer")
		cancel()
		d.cleanup()
	}()

	logger.Info("Daemon started, listening for IPC commands")

	// 启动 IPC 服务器（阻塞，直到 ctx 取消）
	if err := ipc.Serve(ctx, env.Get().SocketFile, d, &ipc.ServerOptions{}); err != nil {
		return err
	}

	logger.Info("Daemon shutting down")
	return nil
}

// cleanup 清理资源
func (d *Daemon) cleanup() {
	d.mu.Lock()
	state := d.state
	defer d.mu.Unlock()

	if d.cancelFunc != nil {
		d.cancelFunc()
		d.cancelFunc = nil
	}

	d.running = false

	if d.lock != nil {
		d.lock.Release()
		d.lock = nil
	}
	if d.service != nil {
		d.service = nil
	}
	if state != nil {
		state.PID = 0
		if err := config.SaveState(state); err != nil {
			logger.Error("Failed to save runtime state", "error", err)
		}
	}
}

// Handle routes the CLI commands to the proper handlers.
func (d *Daemon) Handle(ctx context.Context, cmd ipc.CommandMessage) ipc.CommandResult {
	switch cmd.Name {
	case "run":
		return d.handleRun(ctx, cmd.Payload)
	case "update":
		return d.handleUpdate(ctx)
	case "stop":
		return d.handleStop()
	case "status":
		return d.handleStatus()
	case "mode":
		return d.handleMode(ctx, cmd.Payload)
	case "route":
		return d.handleRoute(ctx, cmd.Payload)
	case "node.list":
		return d.handleNodeList(cmd.Payload)
	case "node.use":
		return d.handleNodeUse(cmd.Payload)
	case "log":
		return d.handleLog()
	case "health":
		return d.handleHealth()
	case "reload":
		return d.handleReload(ctx)
	default:
		return ipc.CommandResult{Status: "error", Error: fmt.Sprintf("unknown command: %s", cmd.Name)}
	}
}

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
	if err := runtime.BuildConfig(env.Get().ConfigFile, env.Get().RawConfigFile, &runops); err != nil {
		return ipc.CommandResult{Status: "error", Error: fmt.Errorf("failed to build config: %w", err).Error()}
	}

	// 3. 启动 sing-box 服务
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
	d.state.RunOptions = runops
	d.mu.Unlock()

	logger.Info("Sing-box started successfully")
	return ipc.CommandResult{Status: "ok", Data: map[string]any{
		"proxy_mode": string(runops.ProxyMode),
		"route_mode": string(runops.RouteMode),
	}}
}

// parseRunOptions 解析 run 命令的参数
func (d *Daemon) parseRunOptions(payload map[string]any) (config.RunOptions, error) {
	runops := config.DefaultRunOptions()
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
		proxyMode, err := config.ParseProxyMode(mode)
		if err != nil {
			return runops, err
		}
		runops.ProxyMode = proxyMode
	}
	if route, ok := payload["route"].(string); ok && route != "" {
		routeMode, err := config.ParseRouteMode(route)
		if err != nil {
			return runops, err
		}
		runops.RouteMode = routeMode
	}
	if port, ok := asInt(payload["api_port"]); ok && port > 0 {
		runops.APIPort = port
	}
	if port, ok := asInt(payload["mixed_port"]); ok && port > 0 {
		runops.MixedPort = port
	}
	return runops, nil
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

func (d *Daemon) handleStatus() ipc.CommandResult {
	running := d.isRunning()
	state, err := d.currentState()
	if err != nil && running {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	data := map[string]any{
		"running": running,
	}
	if state != nil {
		data["proxy_mode"] = state.RunOptions.ProxyMode
		data["route_mode"] = state.RunOptions.RouteMode
		data["pid"] = state.PID
		data["api_port"] = state.RunOptions.APIPort
		data["mixed_port"] = state.RunOptions.MixedPort
		data["listen_addr"] = state.RunOptions.ListenAddr
	}
	return ipc.CommandResult{Status: "ok", Data: data}
}

func (d *Daemon) handleMode(ctx context.Context, payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	modeStr, ok := payload["mode"].(string)
	if !ok || modeStr == "" {
		return ipc.CommandResult{Status: "error", Error: "missing mode"}
	}
	proxyMode, err := config.ParseProxyMode(modeStr)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	state, err := d.currentState()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	if (proxyMode == config.ProxyModeTUN || state.RunOptions.ProxyMode == config.ProxyModeTUN) && os.Geteuid() != 0 {
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
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	routeStr, ok := payload["route"].(string)
	if !ok || routeStr == "" {
		return ipc.CommandResult{Status: "error", Error: "missing route"}
	}
	routeMode, err := config.ParseRouteMode(routeStr)
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

func (d *Daemon) applyRunOptions(ctx context.Context, state *config.RuntimeState) error {
	if err := runtime.BuildConfig(env.Get().ConfigFile, env.Get().RawConfigFile, &state.RunOptions); err != nil {
		return err
	}
	if d.service == nil {
		return errors.New("service not available")
	}
	// TODO: 这里有优化空间，可以设计如果出错，就恢复到之前的状态重启，保证服务可用
	if err := d.service.ReloadFromFile(ctx, env.Get().RawConfigFile); err != nil {
		var reloadErr *service.ReloadError
		if errors.As(err, &reloadErr) && reloadErr.Stage == service.ReloadStageStart {
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
		}
		return err
	}
	d.mu.Lock()
	d.state = state
	d.mu.Unlock()
	return nil
}

func (d *Daemon) handleUpdate(ctx context.Context) ipc.CommandResult {
	logger.Info("Daemon running rule update")
	if err := runUpdate(ctx); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok"}
}

func (d *Daemon) handleStop() ipc.CommandResult {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	// 取消 daemon context 会触发所有子服务退出
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	return ipc.CommandResult{Status: "ok"}
}

func runUpdate(ctx context.Context) error {
	dir := env.Get().AssetDir
	if err := updater.Download(ctx, updater.GeoIPURL, dir, updater.GeoIPFilename, nil); err != nil {
		return err
	}
	return updater.Download(ctx, updater.GeoSiteURL, dir, updater.GeoSiteFilename, nil)
}

func (d *Daemon) newService() ServiceRunner {
	if d.serviceFactory != nil {
		return d.serviceFactory()
	}
	return service.NewInstance()
}

func (d *Daemon) currentState() (*config.RuntimeState, error) {
	d.mu.Lock()
	state := d.state
	d.mu.Unlock()

	if state != nil {
		copyState := *state
		return &copyState, nil
	}
	return nil, nil
}

func (d *Daemon) handleNodeList(payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	apiAddr, err := d.resolveAPIAddr(payload)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	c := client.New(apiAddr)
	proxies, err := c.GetProxies()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"proxies": proxies}}
}

func (d *Daemon) handleNodeUse(payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	group, ok := payload["group"].(string)
	if !ok || group == "" {
		return ipc.CommandResult{Status: "error", Error: "missing group"}
	}
	node, ok := payload["node"].(string)
	if !ok || node == "" {
		return ipc.CommandResult{Status: "error", Error: "missing node"}
	}
	apiAddr, err := d.resolveAPIAddr(payload)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	c := client.New(apiAddr)
	if err := c.SelectProxy(group, node); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"group": group, "node": node}}
}

func (d *Daemon) handleLog() ipc.CommandResult {
	logPath := env.Get().LogFile
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"path": logPath}}
}

func (d *Daemon) handleHealth() ipc.CommandResult {
	running := d.isRunning()
	data := map[string]any{"running": running}
	state, err := d.currentState()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	if state != nil {
		data["pid"] = state.PID
	}
	return ipc.CommandResult{Status: "ok", Data: data}
}

func (d *Daemon) handleReload(ctx context.Context) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	state, err := d.currentState()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	if state == nil {
		return ipc.CommandResult{Status: "error", Error: "missing state"}
	}
	if err := d.applyRunOptions(ctx, state); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok"}
}

func (d *Daemon) resolveAPIAddr(payload map[string]any) (string, error) {
	if payload != nil {
		if api, ok := payload["api"].(string); ok && api != "" {
			return api, nil
		}
	}
	state, err := d.currentState()
	if err != nil {
		return "", err
	}
	if state == nil {
		return "", errors.New("missing state")
	}
	if state.RunOptions.APIPort == 0 {
		return "", errors.New("api port unavailable")
	}
	listenAddr := state.RunOptions.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", listenAddr, state.RunOptions.APIPort), nil
}

func (d *Daemon) isRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

func (d *Daemon) loadState() {

	state, err := config.LoadState()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Error("Failed to load runtime state", "error", err)
		return
	}
	d.mu.Lock()
	d.state = state
	d.state.PID = os.Getpid()
	d.mu.Unlock()
}
