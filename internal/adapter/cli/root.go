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
func NewRootCommand() *cobra.Command {
	var debug bool
	var homeDir string 
	cmd := &cobra.Command{
		Use:   "minibox",
		Short: "Small and beautiful sing-box client",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 1. [关键] 初始化环境
			if err := env.Init(homeDir); err != nil {
				fmt.Println("Failed to initialize environment", "error", err)
				os.Exit(1)
			}

			logger.Setup(logger.Config{
				Debug: debug,
				FilePath: env.Get().LogFile,
			})
			logger.Debug("Logger initialized")
		},
	}

	// bind global flags
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	cmd.PersistentFlags().StringVar(&homeDir, "home", "", "Custom working directory (default: ~/.minibox)")

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
