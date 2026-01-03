package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := dispatchToDaemon(cmd.Context(), "status", nil)
			if err != nil {
				return err
			}
			running, _ := resp.Data["running"].(bool)
			fmt.Printf("Running: %v\n", running)

			if pid, ok := asInt(resp.Data["pid"]); ok && pid != 0 {
				fmt.Printf("PID: %d\n", pid)
			}
			if mode, ok := resp.Data["proxy_mode"].(string); ok && mode != "" {
				fmt.Printf("Proxy mode: %s\n", mode)
			}
			if mode, ok := resp.Data["route_mode"].(string); ok && mode != "" {
				fmt.Printf("Route mode: %s\n", mode)
			}
			if addr, ok := resp.Data["listen_addr"].(string); ok && addr != "" {
				if apiPort, ok := asInt(resp.Data["api_port"]); ok && apiPort != 0 {
					fmt.Printf("API: %s:%d\n", addr, apiPort)
				}
				if mixedPort, ok := asInt(resp.Data["mixed_port"]); ok && mixedPort != 0 {
					fmt.Printf("Mixed: %s:%d\n", addr, mixedPort)
				}
			}

			return nil
		},
	}
}

func newHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check daemon health",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := dispatchToDaemon(cmd.Context(), "health", nil); err != nil {
				return err
			}
			fmt.Println("ok")
			return nil
		},
	}
}

func newReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Reload daemon configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := dispatchToDaemon(cmd.Context(), "reload", nil); err != nil {
				return err
			}
			fmt.Println("Reloaded.")
			return nil
		},
	}
}
