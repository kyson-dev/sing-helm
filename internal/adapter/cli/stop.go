package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running daemon",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 检查是否在运行 (通过锁)
			if err := env.CheckLock(env.Get().HomeDir); err != nil {
				fmt.Println("Minibox is not running.")
				return
			}

			// 读取状态以获取 PID
			state, err := config.LoadState()
			if err != nil {
				fmt.Println("Minibox is running but state file is missing or corrupted.")
				// 这种情况下很难 graceful stop，可能需要提示用户手动 kill
				return
			}

			// 2. 查找进程
			process, err := os.FindProcess(state.PID)
			if err != nil {
				fmt.Println("Process not found.")
				// 清理残留文件
				_ = os.Remove(config.GetStatePath())
				return
			}

			// 3. 发送 SIGTERM 信号
			if err := process.Signal(os.Interrupt); err != nil {
				// 如果进程已经不在了
				fmt.Println("Process already exited.")
			} else {
				fmt.Println("Stopping minibox...")
				// 轮询检查是否真的退出了
				for i := 0; i < 50; i++ {
					if _, err := os.Stat(config.GetStatePath()); os.IsNotExist(err) {
						fmt.Println("Stopped successfully.")
						return
					}
					time.Sleep(100 * time.Millisecond)
				}
				// 超时后强制清理
				fmt.Println("Stop command sent, cleaning up state file...")
				_ = os.Remove(config.GetStatePath())
				fmt.Println("Stopped.")
			}
		},
	}
}
