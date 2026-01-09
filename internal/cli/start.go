package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kysonzou/sing-helm/internal/env"
	"github.com/kysonzou/sing-helm/internal/logger"
	"github.com/spf13/cobra"
)

func newStartCommand() *cobra.Command {
	var dMode string
	var dRule string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start sing-helm in background",
		RunE: func(cmd *cobra.Command, args []string) error {
			if resp, err := dispatchToDaemon(cmd.Context(), "status", nil); err == nil {
				if running, _ := resp.Data["running"].(bool); running {
					return fmt.Errorf("sing-helm is already running")
				}
			}
			// 2. 准备启动参数
			exePath, _ := os.Executable()

			// 使用 env 获取路径
			paths := env.Get()
			logFile := paths.LogFile

			// 传递 --home 给子进程，确保子进程使用相同的目录
			runArgs := []string{"--home", paths.HomeDir}
			if logger.IsDebug() {
				runArgs = append(runArgs, "--debug")
			}
			if logFile != "" {
				runArgs = append(runArgs, "--log", logFile)
			}
			runArgs = append(runArgs, "run")

			// 添加 mode 参数
			if dMode != "" {
				runArgs = append(runArgs, "--mode", dMode)
			}
			if dRule != "" {
				runArgs = append(runArgs, "--route", dRule)
			}

			// 3. 创建命令对象
			command := exec.Command(exePath, runArgs...)

			// 关键：将子进程的 stdout/stderr 重定向到 /dev/null
			// 这样子进程的 isTerminal() 会返回 false，日志会写入文件
			// 注意：设置为 nil 会继承父进程的 stdout，所以必须显式打开 /dev/null
			// 如果没有可用的日志文件，则保留 stdout/stderr 以便看到错误提示
			if logFile != "" {
				devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if err == nil {
					command.Stdout = devNull
					command.Stderr = devNull
					defer devNull.Close()
				}
			}
			command.Stdin = nil

			// 4. 启动子进程
			if err := command.Start(); err != nil {
				return fmt.Errorf("failed to start daemon: %w", err)
			}

			// 5. 等待一小会儿，确保 daemon 可用
			timeout := time.After(2 * time.Second)
			ticker := time.NewTicker(150 * time.Millisecond)
			defer ticker.Stop()
			sawStatus := false
			lastRunning := false
			for {
				select {
				case <-timeout:
					logHint := ""
					if logFile != "" {
						logHint = fmt.Sprintf(" (log: %s)", logFile)
					}
					if sawStatus && !lastRunning {
						return fmt.Errorf("sing-box is not running; check logs for details")
					}
					return fmt.Errorf("daemon failed to start; check logs%s or run with sudo", logHint)
				case <-ticker.C:
					resp, err := dispatchToDaemon(cmd.Context(), "status", nil)
					if err == nil {
						sawStatus = true
						if running, _ := resp.Data["running"].(bool); running {
							fmt.Printf("SingHelm started [PID: %d]\n", command.Process.Pid)
							if logFile != "" {
								fmt.Printf("Log file: %s\n", logFile)
							} else {
								fmt.Println("Log file: (not set)")
							}
							return nil
						}
						lastRunning = false
					}
				}
			}

			// unreachable
		},
	}

	cmd.Flags().StringVarP(&dMode, "mode", "m", "", "Proxy mode: system, tun, or default")
	cmd.Flags().StringVarP(&dRule, "route", "r", "", "Route mode: rule, global, or direct")

	return cmd
}
