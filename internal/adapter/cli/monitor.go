package cli

import (

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyson/minibox/internal/adapter/logger"
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
			model := monitor.NewModel(host)
			p := tea.NewProgram(model, tea.WithAltScreen())

			if _, err := p.Run(); err != nil {
				logger.Error("run monitor command failed", "error", err)
				return 
			}
			logger.Info("run monitor command success")
		},
	}
	cmd.Flags().StringVarP(&host, "host", "H", "127.0.0.1:19090", "Sing-box API host")
	return cmd
}