package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

// 复用之前的 run flags
var (
	dProfilePath string
	dTunMode     bool
	dSystemProxy bool
)

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start minibox in background",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 检查是否已经运行
			if _, err := config.LoadState(); err == nil {
				fmt.Println("Minibox is already running. Please stop it first.")
				os.Exit(1)
			}

			// 2. 准备启动参数
			exePath, _ := os.Executable()

			// 使用 env 获取路径
			paths := env.Get()
			logFile := paths.LogFile

			// 传递 --home 给子进程，确保子进程使用相同的目录
			runArgs := []string{"--home", paths.HomeDir, "run"}
			if dTunMode {
				runArgs = append(runArgs, "--tun")
			}
			if dSystemProxy {
				runArgs = append(runArgs, "--system-proxy")
			}

			// 3. 创建命令对象
			command := exec.Command(exePath, runArgs...)

			// 4. 启动子进程
			if err := command.Start(); err != nil {
				fmt.Printf("Failed to start daemon: %v\n", err)
				os.Exit(1)
			}

			// 5. 等待一小会儿，确保子进程没有立即崩溃 (比如配置错误)
			time.Sleep(1 * time.Second)
			if command.ProcessState != nil && command.ProcessState.Exited() {
				fmt.Println("Daemon process exited unexpectedly. Check logs.")
				os.Exit(1)
			}

			fmt.Printf("Minibox started [PID: %d]\n", command.Process.Pid)
			fmt.Printf("Log file: %s\n", logFile)
		},
	}

	// 绑定 Flags
	cmd.Flags().StringVarP(&dProfilePath, "config", "c", "config.json", "config file")
	cmd.Flags().BoolVar(&dTunMode, "tun", false, "Enable TUN mode")
	cmd.Flags().BoolVar(&dSystemProxy, "system-proxy", false, "Enable System Proxy")

	return cmd
}
