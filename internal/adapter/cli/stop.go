package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running daemon",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 读取状态
			state, err := config.LoadState()
			if err != nil {
				fmt.Println("Minibox is not running.")
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
				fmt.Println("Stop command sent, but state file still exists.")
			}
			
			// 兜底清理（理论上 run 命令退出时会 defer remove state）
			defer os.Remove(config.GetStatePath())
		},
	}
}