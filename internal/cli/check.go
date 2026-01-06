package cli

import (
	"github.com/kyson/minibox/internal/config"
	"github.com/kyson/minibox/internal/logger"
	"github.com/kyson/minibox/internal/runtime"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 配置检查是纯静态分析，不需要 daemon 运行
			return runLocalCheck(configPath)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "path to config file")

	return cmd
}

func runLocalCheck(configPath string) error {
	logger.Info("Check configuration file.....", "path", configPath)

	runops := runtime.DefaultRunOptions()
	if runops.MixedPort == 0 {
		runops.MixedPort = 10808
	}
	if runops.APIPort == 0 {
		runops.APIPort = 18089
	}

	// 使用新的 API，UserOutboundModule 会自动加载配置文件
	builder := config.NewConfigBuilderFromFile(&runops)
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
}
