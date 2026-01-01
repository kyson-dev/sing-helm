package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	unixSocketPerm = 0600
)

// ServerOptions control Serve behavior.
type ServerOptions struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Ready        chan<- struct{}
}

// Serve starts a Unix-domain socket server that dispatches CommandMessage requests.
func Serve(ctx context.Context, socketPath string, handler CommandHandler, opts *ServerOptions) error {
	if handler == nil {
		return fmt.Errorf("ipc: handler is required")
	}

	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("ipc: prepare socket: %w", err)
	}
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ipc: create socket dir: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("ipc: listen error: %w", err)
	}
	if opts != nil && opts.Ready != nil {
		select {
		case opts.Ready <- struct{}{}:
		default:
		}
	}
	if err := os.Chmod(socketPath, unixSocketPerm); err != nil {
		_ = listener.Close()
		return fmt.Errorf("ipc: chmod socket: %w", err)
	}

	var wg sync.WaitGroup
	connCtx, connCancel := context.WithCancel(ctx)
	defer func() {
		connCancel()
		listener.Close()
		wg.Wait()
	}()

	// 监听 context 取消，主动关闭 listener
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("ipc: accept error: %w", err)
			}
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			serveConn(connCtx, c, handler, opts)
		}(conn)
	}
}

func serveConn(ctx context.Context, conn net.Conn, handler CommandHandler, opts *ServerOptions) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	if opts != nil {
		if opts.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(opts.ReadTimeout))
		}
		if opts.WriteTimeout > 0 {
			_ = conn.SetWriteDeadline(time.Now().Add(opts.WriteTimeout))
		}
	}

	var cmd CommandMessage
	if err := decoder.Decode(&cmd); err != nil {
		encoder.Encode(CommandResult{Status: "error", Error: fmt.Sprintf("decode error: %v", err)})
		return
	}

	resp := handler.Handle(ctx, cmd)
	if resp.Status == "" {
		resp.Status = "ok"
	}
	encoder.Encode(resp)
}
