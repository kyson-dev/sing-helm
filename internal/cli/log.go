package cli

import (
	"fmt"

	"github.com/kysonzou/sing-helm/internal/logger"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
)

func newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Stream application logs",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := dispatchToDaemon(cmd.Context(), "log", nil)
			if err != nil {
				logger.Error("Failed to resolve log path", "error", err)
				return
			}
			logPath, ok := resp.Data["path"].(string)
			if !ok || logPath == "" {
				logger.Error("Missing log path from daemon")
				return
			}

			t, err := tail.TailFile(logPath, tail.Config{
				Follow: true,
				ReOpen: true, // 支持日志轮转后继续读
				// 从文件末尾开始读
				Location: &tail.SeekInfo{Offset: 0, Whence: 2},
			})
			if err != nil {
				logger.Error("Failed to tail file", "error", err)
				return
			}

			for line := range t.Lines {
				fmt.Println(line.Text)
			}
		},
	}
}
