package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/spf13/cobra"
)

const (
	systemdUnitPath   = "/etc/systemd/system/sing-helm.service"
	launchdPlistPath  = "/Library/LaunchDaemons/com.kyson.sing-helm.plist"
	launchdPlistLabel = "com.kyson.sing-helm"
)

func newAutostartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autostart",
		Short: "Manage system autostart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("autostart requires root, please run with sudo")
			}
			switch runtime.GOOS {
			case "linux":
				return showSystemdStatus(cmd)
			case "darwin":
				return showLaunchdStatus(cmd)
			default:
				return fmt.Errorf("autostart not supported on %s", runtime.GOOS)
			}
		},
	}
	cmd.AddCommand(newAutostartOnCommand(), newAutostartOffCommand(), newAutostartStatusCommand())
	return cmd
}

func newAutostartOnCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Enable and start system autostart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("autostart requires root, please run with sudo")
			}
			switch runtime.GOOS {
			case "linux":
				return enableSystemd()
			case "darwin":
				return enableLaunchd()
			default:
				return fmt.Errorf("autostart not supported on %s", runtime.GOOS)
			}
		},
	}
}

func newAutostartOffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Disable and stop system autostart",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("autostart requires root, please run with sudo")
			}
			switch runtime.GOOS {
			case "linux":
				return disableSystemd()
			case "darwin":
				return disableLaunchd()
			default:
				return fmt.Errorf("autostart not supported on %s", runtime.GOOS)
			}
		},
	}
}

func newAutostartStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system autostart status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("autostart requires root, please run with sudo")
			}
			switch runtime.GOOS {
			case "linux":
				return showSystemdStatus(cmd)
			case "darwin":
				return showLaunchdStatus(cmd)
			default:
				return fmt.Errorf("autostart not supported on %s", runtime.GOOS)
			}
		},
	}
}

func showSystemdStatus(cmd *cobra.Command) error {
	out, err := cmdOutput("systemctl", "is-enabled", "sing-helm.service")
	if err != nil {
		return err
	}
	cmd.Printf("Enabled: %s\n", strings.TrimSpace(out))
	return nil
}

func showLaunchdStatus(cmd *cobra.Command) error {
	out, err := cmdOutput("launchctl", "print-disabled", "system")
	if err != nil {
		return err
	}
	if !fileExists(launchdPlistPath) {
		cmd.Printf("Enabled: false\n")
		return nil
	}
	if launchdDisabled(out) {
		cmd.Printf("Enabled: false\n")
		return nil
	}
	cmd.Printf("Enabled: true\n")
	return nil
}

func enableSystemd() error {
	unitContent, err := getSystemdUnitContent()
	if err != nil {
		return fmt.Errorf("failed to generate systemd unit content: %w", err)
	}

	if err := os.WriteFile(systemdUnitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	if err := runCmd("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := runCmd("systemctl", "enable", "sing-helm.service"); err != nil {
		return err
	}
	return runCmd("systemctl", "restart", "sing-helm.service")
}

// getSystemdUnitContent dynamically generates the systemd unit content using the current executable path.
//
// Log Strategy (systemd):
// - systemd automatically manages logs via journald (use 'journalctl -u sing-helm' to view)
// - Application log (/var/log/sing-helm/sing-helm.log): Main business logic logs
//
// This ensures compatibility regardless of installation method (apt, manual, brew, etc.)
func getSystemdUnitContent() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Use the environment settings for consistent path handling
	appHome := env.Get().HomeDir
	appLog := env.Get().LogFile

	return `[Unit]
Description=SingHelm daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Restart=on-failure
RestartSec=2
ExecStart=` + exe + ` run --home ` + appHome + ` --log ` + appLog + `
RuntimeDirectory=sing-helm
RuntimeDirectoryMode=0755
LogsDirectory=sing-helm
LogsDirectoryMode=0755
UMask=022

[Install]
WantedBy=multi-user.target
`, nil
}

func disableSystemd() error {
	if err := runCmd("systemctl", "disable", "--now", "sing-helm.service"); err != nil {
		return err
	}
	return nil
}

// getLaunchdPlistContent dynamically generates the plist content using the current executable path.
//
// Log Strategy:
//   - StandardOutPath/StandardErrorPath: Captures system-level issues (startup failures, panics, env errors)
//     These logs are ONLY written when something goes wrong before the app logger initializes,
//     or when third-party code writes directly to stdout/stderr.
//   - Application log: Main business logic logs (path determined by logger.ResolveLogDir)
//     Written by internal/logger, this is where you should look for normal operation logs.
//
// This separation ensures we never miss critical startup failures while keeping business logs clean.
func getLaunchdPlistContent() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Check if we are running via a symlink, resolve it if possible, or just use the path as is if valid.
	// For autostart, using the absolute path to the binary is safest.

	// Use the environment settings directly, as env.Setup() now handles sudo users correctly
	appHome := env.Get().HomeDir
	appLog := env.Get().LogFile

	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + launchdPlistLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + exe + `</string>
		<string>run</string>
		<string>--home</string>
		<string>` + appHome + `</string>
		<string>--log</string>
		<string>` + appLog + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>` + appLog + `</string>
	<key>StandardErrorPath</key>
	<string>` + appLog + `</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
</dict>
</plist>
`, nil
}
func enableLaunchd() error {

	plistContent, err := getLaunchdPlistContent()
	if err != nil {
		return fmt.Errorf("failed to generate plist content: %w", err)
	}

	if err := os.WriteFile(launchdPlistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("write launchd plist: %w", err)
	}
	_ = runCmd("launchctl", "bootout", "system/"+launchdPlistLabel)
	_ = runCmd("launchctl", "bootout", "system", launchdPlistPath)
	if err := runCmd("launchctl", "enable", "system/"+launchdPlistLabel); err != nil {
		return err
	}
	return runCmd("launchctl", "bootstrap", "system", launchdPlistPath)
}

func disableLaunchd() error {
	_ = runCmd("launchctl", "disable", "system/"+launchdPlistLabel)
	_ = runCmd("launchctl", "bootout", "system/"+launchdPlistLabel)
	_ = runCmd("launchctl", "bootout", "system", launchdPlistPath)
	_ = runCmd("launchctl", "unload", launchdPlistPath)
	_ = os.Remove(launchdPlistPath)
	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%s failed: %s", name, msg)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func cmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("%s failed: %s", name, msg)
		}
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	return string(out), nil
}

func launchdDisabled(output string) bool {
	needle := launchdPlistLabel + " => true"
	if strings.Contains(output, "\""+needle+"\"") {
		return true
	}
	return strings.Contains(output, needle)
}
