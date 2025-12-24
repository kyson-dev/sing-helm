package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/config"
	"github.com/kyson/minibox/internal/ui/monitor"
	"github.com/spf13/cobra"
)

func newMonitorCommand() *cobra.Command{
	var host string
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor Sing-box traffic",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("run monitor command", "host", host)
			if host == "" {
				state, err := config.LoadState()
				if err != nil {
					logger.Error("Failed to load state", "error", err)
					os.Exit(1)
				}
				host = fmt.Sprintf("%s:%d",state.ListenAddr,state.APIPort)
			}
			model := monitor.NewModel(host)
			p := tea.NewProgram(model, tea.WithAltScreen())

			if _, err := p.Run(); err != nil {
				logger.Error("run monitor command failed", "error", err)
				return 
			}
			logger.Info("run monitor command success")
		},
	}
	cmd.Flags().StringVarP(&host, "host", "H", "", "Sing-box API host")
	return cmd
}