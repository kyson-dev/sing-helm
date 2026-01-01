package cli

import (
	"fmt"

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
			if len(args) == 0 {
				resp, err := dispatchToDaemon(cmd.Context(), "status", nil)
				if err != nil {
					return err
				}

				if mode, ok := resp.Data["proxy_mode"].(string); ok && mode != "" {
					fmt.Printf("Current proxy mode: %s\n", mode)
					return nil
				}
				return fmt.Errorf("missing proxy mode in daemon status")
			}

			mode := args[0]
			resp, err := dispatchToDaemon(cmd.Context(), "mode", map[string]any{"mode": mode})
			if err != nil {
				return err
			}

			if newMode, ok := resp.Data["proxy_mode"].(string); ok && newMode != "" {
				mode = newMode
			}
			fmt.Printf("Proxy mode switched to: %s\n", mode)
			return nil
		},
	}

	return cmd
}
