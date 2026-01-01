package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/ui/monitor"
	"github.com/spf13/cobra"
)

func newMonitorCommand() *cobra.Command {
	var host string
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor Sing-box traffic",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				resp, err := dispatchToDaemon(cmd.Context(), "status", nil)
				if err != nil {
					return fmt.Errorf("failed to fetch daemon status: %w", err)
				}
				if running, _ := resp.Data["running"].(bool); !running {
					return fmt.Errorf("minibox is not running")
				}
				listenAddr, _ := resp.Data["listen_addr"].(string)
				apiPort, ok := asInt(resp.Data["api_port"])
				if !ok || apiPort == 0 {
					return fmt.Errorf("failed to resolve API port from daemon status")
				}
				if listenAddr == "" {
					return fmt.Errorf("failed to resolve listen address from daemon status")
				}
				host = fmt.Sprintf("%s:%d", listenAddr, apiPort)
			}
			logger.Info("run monitor command", "host", host)
			model := monitor.NewModel(host)
			p := tea.NewProgram(model, tea.WithAltScreen())

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("run monitor command failed: %w", err)
			}
			logger.Info("run monitor command success")
			return nil
		},
	}
	cmd.Flags().StringVarP(&host, "host", "H", "", "Sing-box API host")
	return cmd
}

func asInt(val any) (int, bool) {
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}
