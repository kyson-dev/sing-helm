package cli

import (
	"context"
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/app"
	"github.com/kyson-dev/sing-helm/internal/sys/env"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/spf13/cobra"
)

// appKey is the context key for the Application instance.
type appKey struct{}

// AppFromContext retrieves the Application from a command's context.
// Returns nil if not set (should not happen after PersistentPreRunE).
func AppFromContext(ctx context.Context) *app.Application {
	if v := ctx.Value(appKey{}); v != nil {
		return v.(*app.Application)
	}
	return nil
}

func NewRootCommand() *cobra.Command {
	var homeDir string
	var globalDebug bool
	var logFile string
	cmd := &cobra.Command{
		Use:   "sing-helm",
		Short: "Small and beautiful sing-box client",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			home, _ := cmd.Flags().GetString("home")

			// 使用 setup 初始化环境，支持智能探测和注册
			if err := env.Setup(home); err != nil {
				return fmt.Errorf("environment setup failed: %w", err)
			}

			if logFile == "" {
				logger.Setup(logger.Config{Debug: globalDebug})
			} else {
				logger.Setup(logger.Config{Debug: globalDebug, FilePath: logFile})
			}

			// Build the Application and attach to context
			paths := env.Get()
			application := app.New(paths, logger.GetInstance())
			ctx := context.WithValue(cmd.Context(), appKey{}, application)
			cmd.SetContext(ctx)

			return nil
		},
	}

	// 启用命令建议（当输入错误时会提示相似的命令）
	cmd.SuggestionsMinimumDistance = 2

	// bind global flags
	cmd.PersistentFlags().BoolVarP(&globalDebug, "debug", "d", false, "Enable debug mode")
	cmd.PersistentFlags().StringVar(&homeDir, "home", "", "Custom working directory (default: ~/.sing-helm)")
	cmd.PersistentFlags().StringVar(&logFile, "log", "", "Custom log file (default: system runtime path)")

	// register sub commands
	cmd.AddCommand(newVersionCommand(),
		newConfigCommand(),
		newRunCommand(),
		newStatusCommand(),
		newHealthCommand(),
		newReloadCommand(),
		newMonitorCommand(),
		newNodeCommand(),
		newModeCommand(),
		newRouteCommand(),
		newStartCommand(),
		newStopCommand(),
		newLogCommand(),
		newAutostartCommand(),
		newServeCommand(),
	)

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return NewRootCommand().Execute()
}
