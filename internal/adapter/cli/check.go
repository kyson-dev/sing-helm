package cli

import (
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Check configuration file.....", "path", configPath)

			user, err := config.LoadProfile(configPath)
			if err != nil {
				logger.Error("Config check failed", "error", err)
				return err
			}
			runops := config.DefaultRunOptions()
			opts, err := config.Generate(user, &runops)
			if err != nil {
				logger.Error("Config check failed", "error", err)
				return err
			}

			inCount := len(opts.Inbounds)
			outCount := len(opts.Outbounds)
			logger.Info("Config is valid", "inbounds", inCount, "outbounds", outCount)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "path to config file")

	return cmd
}
