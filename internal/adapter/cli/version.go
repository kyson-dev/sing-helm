package cli

import (
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/version"
	"github.com/spf13/cobra"
)


func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Get().Info((&version.Info{}).String())
		},
	}
}