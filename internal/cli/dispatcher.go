package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/kysonzou/sing-helm/internal/env"
	"github.com/kysonzou/sing-helm/internal/ipc"
)

var errDaemonUnavailable = errors.New("daemon unavailable")
var ErrDaemonUnavailable = errDaemonUnavailable

var commandSenderFactory = defaultCommandSenderFactory

func defaultCommandSenderFactory() ipc.CommandSender {
	socket := env.Get().SocketFile
	if !pathExists(socket) {
		legacy := filepath.Join(env.Get().HomeDir, "ipc.sock")
		if legacy != socket && pathExists(legacy) {
			return ipc.NewUnixSender(legacy)
		}
	}
	return ipc.NewUnixSender(socket)
}

// SetCommandSenderFactory lets tests replace the command sender.
func SetCommandSenderFactory(factory func() ipc.CommandSender) {
	if factory == nil {
		commandSenderFactory = defaultCommandSenderFactory
		return
	}
	commandSenderFactory = factory
}

// ResetCommandSenderFactory restores the default sender.
func ResetCommandSenderFactory() {
	commandSenderFactory = defaultCommandSenderFactory
}

// dispatchToDaemon sends a command message to the daemon, returning errDaemonUnavailable when the socket is unreachable.
func dispatchToDaemon(ctx context.Context, name string, payload map[string]any) (ipc.CommandResult, error) {
	sender := commandSenderFactory()
	resp, err := sender.Send(ctx, ipc.CommandMessage{Name: name, Payload: payload})
	if err != nil {
		if isDaemonUnavailable(err) {
			return ipc.CommandResult{}, errDaemonUnavailable
		}
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

func isDaemonUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if strings.Contains(err.Error(), "connect failed") || strings.Contains(err.Error(), "no such file or directory") {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	return false
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
