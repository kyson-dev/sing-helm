package cli

import (
	"fmt"

	"github.com/kyson/minibox/internal/env"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
)

func newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Stream application logs",
		Run: func(cmd *cobra.Command, args []string) {
			logPath := env.Get().LogFile

			t, err := tail.TailFile(logPath, tail.Config{
				Follow: true,
				ReOpen: true, // 支持日志轮转后继续读
				// 从文件末尾开始读
				Location: &tail.SeekInfo{Offset: 0, Whence: 2}, 
			})
			if err != nil {
				fmt.Println(err)
				return
			}

			for line := range t.Lines {
				fmt.Println(line.Text)
			}
		},
	}
}