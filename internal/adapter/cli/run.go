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
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var (
		tunMode     bool
		systemProxy bool
		apiPort     int
		mixPort     int
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sing-box",
		RunE: func(cmd *cobra.Command, args []string) error {

			runops := config.DefaultRunOptions()
			if systemProxy {
				runops.Mode = config.ModeSystem
			}
			if tunMode {
				runops.Mode = config.ModeTUN
			}
			runops.APIPort = apiPort
			runops.MixedPort = mixPort

			return runService(context.Background(), &runops)
		},
	}
	cmd.Flags().BoolVar(&tunMode, "tun", false, "Enable TUN mode")
	cmd.Flags().BoolVar(&systemProxy, "system-proxy", false, "Enable System Proxy")
	cmd.Flags().IntVar(&apiPort, "api-port", 0, "Fixed API port")
	cmd.Flags().IntVar(&mixPort, "mixed-port", 0, "Fixed Mixed port")
	return cmd
}

// runService 抽取出来的核心逻辑，便于测试
func runService(ctx context.Context, runops *config.RunOptions) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	profilePath := env.Get().ConfigFile
	//1. 加载配置文件
	logger.Info("Loading profile file", "path", profilePath)
	userConf, err := config.LoadProfile(profilePath)
	if err != nil {
		logger.Error("Failed to load profile file", "error", err)
		return fmt.Errorf("failed to load profile file: %w", err)
	}

	opts, err := config.Generate(userConf, runops)
	if err != nil {
		logger.Error("Failed to generate config", "error", err)
		return fmt.Errorf("failed to load config file: %w", err)
	}

	//2. 初始化服务
	svc := service.NewInstance()

	//3. 启动服务
	if err := svc.Start(runCtx, opts); err != nil {
		logger.Error("Failed to start sing-box", "error", err)
		return fmt.Errorf("failed to start sing-box: %w", err)
	}
	defer func() {
		if err := svc.Close(runCtx); err != nil {
			logger.Error("Failed to close sing-box", "error", err)
		}
	}()

	// 保存文件
	state := config.RuntimeState{
		RunOptions: *runops,
		PID:        os.Getpid(),
	}
	if err := config.SaveState(&state); err != nil {
		logger.Error("Failed to save state", "error", err)
		return fmt.Errorf("failed to save state: %w", err)
	}
	defer os.Remove(config.GetStatePath())

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
