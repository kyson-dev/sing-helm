package cli

import (
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/spf13/cobra"
)

const (
	checkCommandMixedPort = 10808
	checkCommandAPIPort   = 18080
)

func newCheckCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check configuration file",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// 不需要环境检查，或者以后实现特定的检查逻辑
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Check configuration file.....", "path", configPath)

			base, err := config.LoadProfile(configPath)
			if err != nil {
				logger.Error("Config check failed", "error", err)
				return err
			}

			runops := config.DefaultRunOptions()
			if runops.MixedPort == 0 {
				runops.MixedPort = checkCommandMixedPort
			}
			if runops.APIPort == 0 {
				runops.APIPort = checkCommandAPIPort
			}
			builder := config.NewConfigBuilder(base, &runops)
			for _, m := range config.DefaultModules(&runops) {
				builder.With(m)
			}

			opts, err := builder.Build()
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
