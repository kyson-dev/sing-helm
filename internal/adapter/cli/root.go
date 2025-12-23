package cli

import (
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/version"
	"github.com/spf13/cobra"
)

// RootCmd 是为了让 main 能访问，但实际不建议直接暴露全局变量
// 这里演示"依赖注入"式的构建
func NewRootCommand() *cobra.Command {
	var debug bool
	cmd := &cobra.Command{
		Use:   "minibox",
		Short: "Small and beautiful sing-box client",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Setup(debug)
			logger.Debug("Logger initialized")
		},
	}

	// bind global flags
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")

	// register sub commands
	cmd.AddCommand(newVersionCommand(),
		newCheckCommand(),
		newRunCommand(),
		newUpdateCommand())

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Get().Info((&version.Info{}).String())
		},
	}
}

// execute command
func Execute() error {
	return NewRootCommand().Execute()
}
