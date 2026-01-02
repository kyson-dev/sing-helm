package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyson/minibox/internal/adapter/logger"
	coredaemon "github.com/kyson/minibox/internal/core/daemon"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
	"github.com/spf13/cobra"
)

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
			payload := map[string]any{}
			if cmd.Flags().Changed("mode") {
				payload["mode"] = mode
			}
			if cmd.Flags().Changed("route") {
				payload["route"] = rule
			}
			if cmd.Flags().Changed("api-port") {
				payload["api_port"] = apiPort
			}
			if cmd.Flags().Changed("mixed-port") {
				payload["mixed_port"] = mixPort
			}

			// 无论是前台还是后台运行，都启动 Daemon + IPC + Sing-box
			// 前台运行：阻塞终端
			// 后台运行：由 start 命令触发，这里也是阻塞（但在后台进程中）
			return runAsDaemon(cmd.Context(), payload)
		},
	}
	cmd.Flags().StringVarP(&mode, "mode", "m", "", "Proxy mode: system, tun, or default")
	cmd.Flags().StringVarP(&rule, "route", "r", "", "Route mode: rule, global, or direct")
	cmd.Flags().IntVar(&apiPort, "api-port", 0, "Fixed API port")
	cmd.Flags().IntVar(&mixPort, "mixed-port", 0, "Fixed Mixed port")
	return cmd
}

// runAsDaemon 以 daemon 模式运行：启动 sing-box 和 IPC 服务器
func runAsDaemon(ctx context.Context, payload map[string]any) error {
	logger.Info("Starting in daemon mode")

	// 创建独立的 context，不依赖于命令的 context
	// 这样 daemon 的生命周期完全由信号和 IPC 控制
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 监听系统信号（Ctrl+C, kill）
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// 在后台监听信号，收到信号时取消 context
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, initiating shutdown...", "signal", sig)
		cancel()
	}()

	// 创建 daemon 实例
	d := coredaemon.NewDaemon()

	// 在后台启动 sing-box 服务（通过 IPC run 命令）
	go func() {
		// 等待 IPC 服务器启动
		time.Sleep(200 * time.Millisecond)

		// 现在（正确）
		sender := ipc.NewUnixSender(env.Get().SocketFile)
		resp, err := sender.Send(context.Background(), ipc.CommandMessage{
			Name:    "run",
			Payload: payload,
		})
		if err != nil {
			logger.Error("Failed to send run command", "error", err)
			return
		}
		if resp.Status != "" && resp.Status != "ok" {
			if resp.Error != "" {
				logger.Error("Run command failed", "error", resp.Error)
			} else {
				logger.Error("Run command failed", "status", resp.Status)
			}
			return
		}
		logger.Info("Run command sent successfully")
	}()

	// 启动 IPC 服务器（阻塞直到 context 取消或发生错误）
	err := d.Serve(runCtx)

	if err != nil {
		logger.Error("Daemon stopped with error", "error", err)
		return err
	}

	logger.Info("Daemon stopped gracefully")
	return nil
}
