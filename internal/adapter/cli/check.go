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
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Check configuration file.....", "path", configPath)

			opts, err := config.Load(configPath)
			if err != nil {
				// 只是检查，不要 panic，打印错误日志即可
				logger.Error("Config check failed", "error", err)
				// 按照 UNIX 规范，失败退出码为 1
				// 但这里我们在 Run 内部，可以用 os.Exit(1) 或者返回 error 让上层处理
				return
			}

			inCount := len(opts.Inbounds)
			outCount := len(opts.Outbounds)
			logger.Info("Config is valid", "inbounds", inCount, "outbounds", outCount)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "path to config file")

	return cmd

}
