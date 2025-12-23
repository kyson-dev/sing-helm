package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/core/service"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sing-box",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runService(cmd.Context(), configPath)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "Config file")
	return cmd
}

// runService 抽取出来的核心逻辑，便于测试
func runService(ctx context.Context, configPath string) error {
	//1. 加载配置文件
	logger.Info("Loading config file", "path", configPath)
	opts, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	//2. 初始化服务
	svc := service.NewInstance()

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//3. 启动服务
	if err := svc.Start(ctx, opts); err != nil {
		return fmt.Errorf("failed to start sing-box: %w", err)
	}
	defer func() {
		if err := svc.Close(ctx); err != nil {
			logger.Error("Failed to close sing-box", "error", err)
		}
	}()

	//4. 等待服务退出
	// 监听信号或 context 取消
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		logger.Info("Received signal, shutting down...", "signal", sig)
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down...")
	}

	logger.Info("Service stopped gracefully")
	return nil
}
