package daemon

import (
	"errors"
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/clashapi"
	"github.com/kyson-dev/sing-helm/internal/ipc"
)

func (d *Daemon) handleNodeList(payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "sing-box not running"}
	}
	apiAddr, err := d.resolveAPIAddr(payload)
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	c := clashapi.New(apiAddr)
	proxies, err := c.GetProxies()
	if err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"proxies": proxies}}
}

func (d *Daemon) handleNodeUse(payload map[string]any) ipc.CommandResult {
	if !d.isRunning() {
		return ipc.CommandResult{Status: "error", Error: "sing-box not running"}
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
	c := clashapi.New(apiAddr)
	if err := c.SelectProxy(group, node); err != nil {
		return ipc.CommandResult{Status: "error", Error: err.Error()}
	}
	return ipc.CommandResult{Status: "ok", Data: map[string]any{"group": group, "node": node}}
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
