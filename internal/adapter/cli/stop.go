package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 发送 stop 命令
			if _, err := dispatchToDaemon(cmd.Context(), "stop", nil); err != nil {
				return fmt.Errorf("failed to stop daemon: %w", err)
			}

			fmt.Println("Stop command sent, waiting for daemon to shutdown...")

			// 等待 daemon 真正停止（通过轮询 status）
			timeout := time.After(5 * time.Second)
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					return fmt.Errorf("timeout waiting for daemon to stop")
				case <-ticker.C:
					// 尝试连接 daemon，如果连接失败说明已经停止
					_, err := dispatchToDaemon(cmd.Context(), "status", nil)
					if err != nil {
						// daemon 已经停止
						fmt.Println("Daemon stopped successfully.")
						return nil
					}
				}
			}
		},
	}
}
