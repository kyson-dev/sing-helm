package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kysonzou/sing-helm/internal/config"
	"github.com/kysonzou/sing-helm/internal/logger"
	"github.com/kysonzou/sing-helm/internal/runtime"
	"github.com/kysonzou/sing-helm/internal/tools/exporter"
	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	var (
		port          int
		platform      string
		targetVersion string
		output        string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Generate config file and start local HTTP server",
		Long:  `Generates a sing-box configuration file locally and starts a HTTP server to share it via LAN.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 生成并写入本地文件
			runops := runtime.DefaultRunOptions()
			runops.ProxyMode = runtime.ProxyModeTUN
			runops.RouteMode = runtime.RouteModeRule

			logger.Info("Building options...")
			opts, err := config.BuildOptions(&runops)
			if err != nil {
				return err
			}

			logger.Info("Exporting config...", "version", targetVersion, "platform", platform)
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
			fmt.Printf("✅ Config exported to: %s\n", output)

			// 2. 启动 HTTP 服务
			bestIP, allIPs, err := getLanIPs()
			if err != nil {
				return fmt.Errorf("failed to detect LAN IP: %w", err)
			}

			url := fmt.Sprintf("http://%s:%d/config", bestIP.String(), port)

			fmt.Printf("\n🚀 SingHelm LAN Server Running on :%d\n", port)
			fmt.Printf("Primary Subscription URL: %s\n", url)

			if len(allIPs) > 1 {
				fmt.Println("\nAlternative IPs detected:")
				for _, ip := range allIPs {
					if ip.String() != bestIP.String() {
						fmt.Printf("  - http://%s:%d/config\n", ip.String(), port)
					}
				}
			}

			http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if rec := recover(); rec != nil {
						logger.Error("Panic in HTTP handler: %v", rec)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
				}()

				logger.Info("Received subscription request", "remote", r.RemoteAddr)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("Content-Disposition", "attachment; filename=sing-box.json")

				w.Write(data)
			})
			go func() {
				addr := fmt.Sprintf("0.0.0.0:%d", port)
				if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server failed: %v", err)
					os.Exit(1)
				}
			}()

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig
			fmt.Println("\nServer stopped.")
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8090, "HTTP server port")
	cmd.Flags().StringVar(&platform, "platform", "ios", "Target platform (e.g. ios)")
	cmd.Flags().StringVar(&targetVersion, "target-version", "1.11.4", "Target sing-box version")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output local file path")

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

// getLanIPs returns the "best guess" LAN IP and a list of all non-loopback IPs
func getLanIPs() (net.IP, []net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	var allIPs []net.IP
	var bestIP net.IP

	// score: 192.168 > 10.x > 172.x > others
	bestScore := -1

	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// only ipv4
			ip = ip.To4()
			if ip == nil {
				continue
			}

			allIPs = append(allIPs, ip)

			score := 0
			if strings.HasPrefix(ip.String(), "192.168.") {
				score = 3
			} else if strings.HasPrefix(ip.String(), "10.") {
				score = 2
			} else if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
				score = 1
			}

			if score > bestScore {
				bestScore = score
				bestIP = ip
			}
		}
	}

	if bestIP == nil {
		if len(allIPs) > 0 {
			return allIPs[0], allIPs, nil
		}
		return nil, nil, fmt.Errorf("no valid IP found")
	}

	return bestIP, allIPs, nil
}
