package cli

import (
	"fmt"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/core/controller"
	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

func newModeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode [system|tun|default]",
		Short: "Switch proxy mode",
		Long: `Switch the proxy mode:
  system  - Use system proxy (default)
  tun     - Use TUN virtual network interface (requires root)
  default - Only open port, configure proxy manually

Note: This will restart sing-box to apply the new mode.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 检查是否在运行
			if err := env.CheckLock(env.Get().HomeDir); err != nil {
				return fmt.Errorf("minibox is not running: %w", err)
			}

			// 如果没有参数，显示当前模式
			if len(args) == 0 {
				state, err := config.LoadState()
				if err != nil {
					return fmt.Errorf("daemon not running: %w", err)
				}
				proxyMode := state.ProxyMode
				if proxyMode == "" {
					proxyMode = config.ProxyModeSystem
				}
				fmt.Printf("Current proxy mode: %s\n", proxyMode)
				return nil
			}

			mode := args[0]
			newMode, err := controller.SwitchProxyMode(mode)
			if err != nil {
				return err
			}

			fmt.Printf("Proxy mode switched to: %s\n", newMode)
			return nil
		},
	}

	return cmd
}
