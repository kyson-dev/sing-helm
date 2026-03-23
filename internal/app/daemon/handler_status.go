package daemon

import (
	"context"

	"github.com/kyson-dev/sing-helm/internal/sys/ipc"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

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

func (d *Daemon) handleLog() ipc.CommandResult {
	logPath := paths.Get().LogFile
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"path": logPath}}
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

func (d *Daemon) handleStop() ipc.CommandResult {
	d.mu.Lock()
	running := d.running
	cancel := d.cancelFunc
	d.mu.Unlock()

	if cancel == nil {
		if running {
			return ipc.CommandResult{Status: "error", Error: "daemon not running"}
		}
		return ipc.CommandResult{Status: "error", Error: "daemon not running"}
	}
	// 取消 daemon context 会触发所有子服务退出
	cancel()
	return ipc.CommandResult{Status: "ok"}
}
