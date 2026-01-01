package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// CommandSender dispatches command messages to the daemon.
type CommandSender interface {
	Send(ctx context.Context, cmd CommandMessage) (CommandResult, error)
}

// UnixSender dials the unix socket each time Send is invoked.
type UnixSender struct {
	Socket  string
	Dial    func(network, address string) (net.Conn, error)
	Timeout time.Duration
}

// NewUnixSender returns a CommandSender that communicates over a unix socket.
func NewUnixSender(socket string) *UnixSender {
	return &UnixSender{
		Socket:  socket,
		//Dial:    func(network, address string) (net.Conn, error) { return net.Dial("unix", address) },
		Timeout: 2 * time.Second,
	}
}

func (s *UnixSender) Send(ctx context.Context, cmd CommandMessage) (CommandResult, error) {
	if s == nil || s.Socket == "" {
		return CommandResult{}, fmt.Errorf("ipc: invalid unix sender")
	}

	dialCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(dialCtx, "unix", s.Socket)
	if err != nil {
		return CommandResult{}, fmt.Errorf("ipc: connect failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(s.Timeout)); err != nil {
		return CommandResult{}, err
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(cmd); err != nil {
		return CommandResult{}, fmt.Errorf("ipc: encode command: %w", err)
	}

	var resp CommandResult
	if err := decoder.Decode(&resp); err != nil {
		return CommandResult{}, fmt.Errorf("ipc: decode response: %w", err)
	}
	return resp, nil
}

// FakeSender allows CLI tests to inject deterministic responses.
type FakeSender struct {
	Response CommandResult
	Err      error
}

func (f *FakeSender) Send(ctx context.Context, cmd CommandMessage) (CommandResult, error) {
	if f.Err != nil {
		return CommandResult{}, f.Err
	}
	return f.Response, nil
}
