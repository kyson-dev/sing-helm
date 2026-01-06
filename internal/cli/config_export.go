package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyson/minibox/internal/config"
	"github.com/kyson/minibox/internal/runtime"
	"github.com/kyson/minibox/internal/tools/exporter"
	"github.com/spf13/cobra"
)

func newConfigExportCommand() *cobra.Command {
	var (
		mode          string
		route         string
		listenAddr    string
		apiPort       int
		mixedPort     int
		targetVersion string
		platform      string
		output        string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export sing-box config for a target version/platform",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runops := runtime.DefaultRunOptions()

			if mode != "" {
				parsed, err := runtime.ParseProxyMode(mode)
				if err != nil {
					return err
				}
				runops.ProxyMode = parsed
			}

			if route != "" {
				parsed, err := runtime.ParseRouteMode(route)
				if err != nil {
					return err
				}
				runops.RouteMode = parsed
			}
			if listenAddr != "" {
				runops.ListenAddr = listenAddr
			}
			if apiPort != 0 {
				runops.APIPort = apiPort
			}
			if mixedPort != 0 {
				runops.MixedPort = mixedPort
			}

			opts, err := config.BuildOptions(&runops)
			if err != nil {
				return err
			}

			data, err := exporter.Export(opts, exporter.Target{Version: targetVersion, Platform: platform})
			if err != nil {
				return err
			}

			if output == "" {
				output = defaultExportPath(targetVersion, platform)
			}
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(output, data, 0644); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Exported: %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&mode, "mode", "m", "tun", "Proxy mode: system, tun, or default")
	cmd.Flags().StringVarP(&route, "route", "r", "rule", "Route mode: rule, global, or direct")
	cmd.Flags().StringVar(&listenAddr, "listen-addr", "", "Listen address")
	cmd.Flags().IntVar(&apiPort, "api-port", 0, "Fixed API port")
	cmd.Flags().IntVar(&mixedPort, "mixed-port", 0, "Fixed Mixed port")
	cmd.Flags().StringVar(&targetVersion, "target-version", "1.11.4", "Target sing-box version (e.g. 1.11.4)")
	cmd.Flags().StringVar(&platform, "platform", "ios", "Target platform (e.g. ios)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

func defaultExportPath(version, platform string) string {
	name := "singbox-config"
	ver := strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if ver != "" {
		name += "-" + ver
	}
	plat := strings.TrimSpace(strings.ToLower(platform))
	if plat != "" {
		name += "-" + plat
	}
	return filepath.Join("bin", name+".json")
}
