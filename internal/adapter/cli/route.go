package cli

import (
	"fmt"
	"strings"

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
			if len(args) == 0 {
				resp, err := dispatchToDaemon(cmd.Context(), "status", nil)
				if err != nil {
					return err
				}

				if mode, ok := resp.Data["route_mode"].(string); ok && mode != "" {
					cmd.Printf("Current route mode: %s\n", mode)
					return nil
				}
				return fmt.Errorf("missing route mode in daemon status")
			}

			mode := args[0]
			resp, err := dispatchToDaemon(cmd.Context(), "route", map[string]any{"route": mode})
			if err != nil {
				if strings.Contains(err.Error(), "sing-box not running") {
					return fmt.Errorf("sing-box is not running")
				}
				return err
			}

			if newMode, ok := resp.Data["route_mode"].(string); ok && newMode != "" {
				mode = newMode
			}
			cmd.Printf("Route mode switched to: %s\n", mode)
			return nil
		},
	}

	return cmd
}
