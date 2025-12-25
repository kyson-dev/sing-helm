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

func newStartCommand() *cobra.Command {
	var dMode string
	var dRule string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start minibox in background",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 检查是否已经运行
			if err := config.CheckLock(); err == nil {
				fmt.Println("Minibox is already running (lock file exists). Please stop it first.")
				os.Exit(1)
			}

			// 2. 准备启动参数
			exePath, _ := os.Executable()

			// 使用 env 获取路径
			paths := env.Get()
			logFile := paths.LogFile

			// 传递 --home 给子进程，确保子进程使用相同的目录
			runArgs := []string{"--home", paths.HomeDir}
			if GlobalDebug {
				runArgs = append(runArgs, "--debug")
			}
			if LogFile == "" {
				LogFile = env.Get().LogFile
			}
			runArgs = append(runArgs, "--log", LogFile)
			runArgs = append(runArgs, "run")

			// 添加 mode 参数
			if dMode != "" && dMode != "default" {
				runArgs = append(runArgs, "--mode", dMode)
			}
			if dRule != "" && dRule != "rule" {
				runArgs = append(runArgs, "--route", dRule)
			}

			// 3. 创建命令对象
			command := exec.Command(exePath, runArgs...)

			// 关键：将子进程的 stdout/stderr 重定向到 /dev/null
			// 这样子进程的 isTerminal() 会返回 false，日志会写入文件
			// 注意：设置为 nil 会继承父进程的 stdout，所以必须显式打开 /dev/null
			devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err == nil {
				command.Stdout = devNull
				command.Stderr = devNull
				defer devNull.Close()
			}
			command.Stdin = nil

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

	cmd.Flags().StringVarP(&dMode, "mode", "m", "system", "Proxy mode: system, tun, or default")
	cmd.Flags().StringVarP(&dRule, "route", "r", "rule", "Route mode: rule, global, or direct")

	return cmd
}
