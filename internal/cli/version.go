package cli

import (
	"github.com/kyson/sing-helm/internal/logger"
	"github.com/kyson/sing-helm/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 不需要任何环境检查
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info((&version.Info{}).String())
		},
	}
}
