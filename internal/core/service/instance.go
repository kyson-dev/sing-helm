package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/ipc"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

type instance struct {
	mu         sync.Mutex
	once       sync.Once
	box        *box.Box
	listener   net.Listener
	socketPath string
	done       chan struct{}
}

func NewInstance() *instance {
	return &instance{}
}

func (s *instance) Run(ctx context.Context, socketPath string) error {
	if s.listener != nil {
		return fmt.Errorf("listener already exists")
	}

	// 接收ipc信号
	os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	s.listener = listener
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()

	// 启动 IPC 服务
	go func() {
		s.acceptLoop(ctx)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	s.done = make(chan struct{})
	select {
	case <-s.done:
		s.clean()
		logger.Info("sing-box core stopped")
	case sig := <-sigCh:
		s.clean()
		logger.Info("Received signal, shutting down...", "signal", sig)
	}
	return nil
}

// func (s *instance) Shutdown() {
// 	s.done <- struct{}{}
// }

func (s *instance) clean() {
	s.box.Close()
	s.listener.Close()
}

// ReloadFromFile 从配置文件重新加载 sing-box
func (s *instance) ReloadFromFile(ctx context.Context, configPath string) error {
	if s.box != nil {
		if err := s.box.Close(); err != nil {
			// 忽略 "file already closed" 错误
			if !isAlreadyClosedError(err) {
				return fmt.Errorf("failed to close box instance: %w", err)
			}
			logger.Info("Box instance already closed, continuing reload")
		}
		s.box = nil
	}
	return s.StartFromFile(ctx, configPath)
}

// isAlreadyClosedError 检查是否是 "file already closed" 错误
func isAlreadyClosedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "file already closed" || errStr == "use of closed file"
}

// StartFromFile 从配置文件启动 sing-box
func (s *instance) StartFromFile(ctx context.Context, configPath string) error {
	// 从文件加载配置
	opts, err := config.LoadOptionsWithContext(ctx, configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	return s.Start(ctx, opts)
}

// Start 启动 sing-box（接收 option.Options）
func (s *instance) Start(ctx context.Context, opts *option.Options) error {
	if s.box != nil {
		return fmt.Errorf("box instance already exists")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	logger.Info("Initializing sing-box core...")

	// 参数校验
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	tx := include.Context(ctx)
	newBox, err := box.New(box.Options{
		Context:           tx,
		Options:           *opts,
		PlatformLogWriter: logger.NewPlatformWriter(), // 将 sing-box 日志重定向到我们的 logger
	})
	if err != nil {
		return fmt.Errorf("failed to create box instance: %w", err)
	}
	s.box = newBox
	// 2. Start sing-box core
	if err := s.box.Start(); err != nil {
		return fmt.Errorf("failed to start sing-box core: %w", err)
	}
	return nil
}

func (s *instance) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				logger.Info("[DEBUG]: IPC listener closing due to context cancellation")
				return
			default:
				continue
			}
		}
		// Handle each connection in a goroutine
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single client connection
func (s *instance) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger.Info("New IPC connection accepted")

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var req ipc.Request
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				// 客户端正常关闭连接
				logger.Info("IPC connection closed by client")
				return
			}
			logger.Error("Failed to decode IPC request", "error", err)
			return
		}

		logger.Info("Received IPC request", "method", req.Method, "id", req.ID)
		resp := s.handleRequest(ctx, &req)
		logger.Info("Sending IPC response", "id", resp.ID, "hasError", resp.Error != nil)

		if err := encoder.Encode(resp); err != nil {
			logger.Error("Failed to encode IPC response", "error", err)
			return
		}
	}
}

// handleRequest routes IPC requests to appropriate handlers
func (s *instance) handleRequest(ctx context.Context, req *ipc.Request) *ipc.Response {
	switch req.Method {
	case ipc.MethodReload:
		return s.Reload(ctx, req)
	default:
		return ipc.NewErrorResponse(req.ID, ipc.ErrCodeMethodNotFound, "method not found")
	}
}
