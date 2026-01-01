package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

type instance struct {
	mu   sync.Mutex
	once sync.Once
	box  *box.Box
}

func NewInstance() *instance {
	return &instance{}
}

func (s *instance) Stop() {
	if s.box != nil {
		if err := s.box.Close(); err != nil {
			logger.Error("Failed to close box instance", "error", err)
			return 
		}
		logger.Info("Sing-box instance closed successfully")
		s.box = nil
	}
}	

// ReloadFromFile 从配置文件重新加载 sing-box
func (s *instance) ReloadFromFile(ctx context.Context, configPath string) error {
	if s.box != nil {
		if err := s.box.Close(); err != nil {
			// 忽略 "file already closed" 错误
			if !isAlreadyClosedError(err) {
				return &ReloadError{
					Stage: ReloadStageStop,
					Err:   fmt.Errorf("failed to close box instance: %w", err),
				}
			}
			logger.Info("Box instance already closed, continuing reload")
		}
		s.box = nil
	}
	if err := s.StartFromFile(ctx, configPath); err != nil {
		return &ReloadError{
			Stage: ReloadStageStart,
			Err:   err,
		}
	}
	return nil
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

	// 2. Start sing-box core
	if err := newBox.Start(); err != nil {
		return fmt.Errorf("failed to start sing-box core: %w", err)
	}
	s.box = newBox
	logger.Info("Sing-box core started, launching cleanup goroutine")
	return nil
}
