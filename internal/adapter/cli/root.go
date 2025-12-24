package cli

import (
	"fmt"
	"os"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)
// RootCmd 是为了让 main 能访问，但实际不建议直接暴露全局变量
// 这里演示"依赖注入"式的构建
var GlobalDebug bool
var LogFile     string
func NewRootCommand() *cobra.Command {
	var homeDir string 
	cmd := &cobra.Command{
		Use:   "minibox",
		Short: "Small and beautiful sing-box client",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 1. [关键] 初始化环境
			if err := env.Init(homeDir); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if LogFile == "" {
				logger.Setup(logger.Config{Debug: GlobalDebug})
			}else {
				logger.Setup(logger.Config{Debug: GlobalDebug,FilePath: LogFile})
			}
		},
	}

	// bind global flags
	cmd.PersistentFlags().BoolVarP(&GlobalDebug, "debug", "d", false, "Enable debug mode")
	cmd.PersistentFlags().StringVar(&homeDir, "home", "", "Custom working directory (default: ~/.minibox)")
	cmd.PersistentFlags().StringVar(&LogFile, "log", "", "Custom log file (default: ~/.minibox/log)")

	// register sub commands
	cmd.AddCommand(newVersionCommand(),
		newCheckCommand(),
		newRunCommand(),
		newUpdateCommand(),
		newMonitorCommand(),
		newNodeCommand(),
		newStartCommand(),
		newStopCommand(),
		newLogCommand(),
	)

	return cmd
}
// execute command
func Execute() error {
	return NewRootCommand().Execute()
}
