package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/core/service"
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

const skipServiceEnv = "MINIBOX_TEST_SKIP_SERVICE"

func newRunCommand() *cobra.Command {
	var (
		mode    string
		rule    string
		apiPort int
		mixPort int
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sing-box",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 检查是否已经运行 (启动命令必须独占)
			if err := env.CheckLock(env.Get().HomeDir); err == nil {
				return fmt.Errorf("minibox is already running at %s", env.Get().HomeDir)
			}

			runops := config.DefaultRunOptions()
			m, err := config.ParseProxyMode(mode)
			if err != nil {
				return err
			}
			runops.ProxyMode = m
			r, err := config.ParseRouteMode(rule)
			if err != nil {
				return err
			}
			runops.RouteMode = r
			runops.APIPort = apiPort
			runops.MixedPort = mixPort

			return runService(context.Background(), &runops)
		},
	}
	cmd.Flags().StringVarP(&mode, "mode", "m", "system", "Proxy mode: system, tun, or default")
	cmd.Flags().StringVarP(&rule, "route", "r", "rule", "Route mode: rule, global, or direct")
	cmd.Flags().IntVar(&apiPort, "api-port", 0, "Fixed API port")
	cmd.Flags().IntVar(&mixPort, "mixed-port", 0, "Fixed Mixed port")
	return cmd
}

// runService 抽取出来的核心逻辑，便于测试
func runService(ctx context.Context, runops *config.RunOptions) error {
	// 获取文件锁，确保单实例运行
	lock, err := env.AcquireLock(env.Get().HomeDir)
	if err != nil {
		return fmt.Errorf("minibox is already running (failed to acquire lock): %w", err)
	}
	defer lock.Release()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	profilePath := env.Get().ConfigFile

	//1. 加载用户配置
	logger.Info("Loading profile file", "path", profilePath)
	base, err := config.LoadProfile(profilePath)
	if err != nil {
		logger.Error("Failed to load profile file", "error", err)
		return fmt.Errorf("failed to load profile file: %w", err)
	}

	// 2. 构建完整配置
	builder := config.NewConfigBuilder(base, runops)
	for _, m := range config.DefaultModules(runops) {
		builder.With(m)
	}

	// 3. 保存完整配置到 raw.json
	rawPath := env.Get().RawConfigFile
	if err := builder.SaveToFile(rawPath); err != nil {
		logger.Error("Failed to save raw config", "error", err)
		return fmt.Errorf("failed to save raw config: %w", err)
	}

	// 4. 保存运行状态
	state := config.RuntimeState{
		RunOptions: *runops,
		PID:        os.Getpid(),
	}
	if err := config.SaveState(&state); err != nil {
		logger.Error("Failed to save state", "error", err)
		return fmt.Errorf("failed to save state: %w", err)
	}
	defer os.Remove(config.GetStatePath())

	if shouldSkipService() {
		logger.Info("Skipping sing-box startup due to test mode")
		return nil
	}

	// 5. 初始化服务
	svc := service.NewInstance()

	// 6. 从配置文件启动
	if err := svc.StartFromFile(runCtx, rawPath); err != nil {
		logger.Error("Sing-box error", "error", err)
		return fmt.Errorf("sing-box error: %w", err)
	}

	// 阻塞
	if err := svc.Run(runCtx, env.Get().SocketFile); err != nil {
		logger.Error("Sing-box error", "error", err)
		return fmt.Errorf("sing-box error: %w", err)
	}

	logger.Info("Service stopped gracefully")
	return nil
}

func shouldSkipService() bool {
	return os.Getenv(skipServiceEnv) == "1"
}
