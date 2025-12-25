package cli

import (
	"fmt"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/core/controller"
	"github.com/spf13/cobra"
)

func newRouteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [rule|global|direct]",
		Short: "Switch route mode",
		Long: `Switch the route mode:
  rule   - Route traffic based on rules (default)
  global - All traffic goes through proxy
  direct - All traffic goes direct

Note: This will restart sing-box to apply the new mode.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 检查是否在运行
			if err := config.CheckLock(); err != nil {
				return fmt.Errorf("minibox is not running: %w", err)
			}

			// 如果没有参数，显示当前模式
			if len(args) == 0 {
				state, err := config.LoadState()
				if err != nil {
					return fmt.Errorf("daemon not running: %w", err)
				}
				routeMode := state.RouteMode
				if routeMode == "" {
					routeMode = config.RouteModeRule
				}
				fmt.Printf("Current route mode: %s\n", routeMode)
				return nil
			}

			mode := args[0]
			newMode, err := controller.SwitchRouteMode(mode)
			if err != nil {
				return err
			}

			fmt.Printf("Route mode switched to: %s\n", newMode)
			return nil
		},
	}

	return cmd
}
