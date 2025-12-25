package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyson/minibox/internal/adapter/logger"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

type instance struct {
	mu  sync.Mutex
	box *box.Box
}

func NewInstance() *instance {
	return &instance{}
}

func (s *instance) Start(ctx context.Context, opts *option.Options) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	logger.Info("Initializing sing-box core...")

	// 参数校验
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	// 1. Initialize sing-box core
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
	// TODO: 这里缺少了sing-box异常退出的处理

	// 3. Wait for sing-box core to exit
	go func() {
		<-ctx.Done()
		logger.Info("receiving stop signal, closing sing-box core...")
		s.Close(ctx)
		logger.Info("sing-box core closed successfully")
	}()

	logger.Info("Sing-box core started successfully")
	return nil
}

func (s *instance) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.box == nil {
		return nil
	}
	logger.Info("Stopping sing-box core...")
	err := s.box.Close()
	s.box = nil // 设置为 nil，确保幂等性
	return err
}

func (s *instance) IsRunning() bool {
	return s.box != nil
}
