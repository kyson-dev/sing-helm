package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	systemdUnitPath   = "/etc/systemd/system/minibox.service"
	launchdPlistPath  = "/Library/LaunchDaemons/com.kyson.minibox.plist"
	launchdPlistLabel = "com.kyson.minibox"
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
	out, err := cmdOutput("systemctl", "is-enabled", "minibox.service")
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
	if err := os.WriteFile(systemdUnitPath, []byte(systemdUnitContent), 0644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	if err := runCmd("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := runCmd("systemctl", "enable", "minibox.service"); err != nil {
		return err
	}
	return runCmd("systemctl", "restart", "minibox.service")
}

func disableSystemd() error {
	if err := runCmd("systemctl", "disable", "--now", "minibox.service"); err != nil {
		return err
	}
	return nil
}

func enableLaunchd() error {
	if err := os.WriteFile(launchdPlistPath, []byte(launchdPlistContent), 0644); err != nil {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const systemdUnitContent = `[Unit]
Description=Minibox daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/minibox run
Restart=on-failure
RestartSec=2
RuntimeDirectory=minibox
RuntimeDirectoryMode=0755
LogsDirectory=minibox
LogsDirectoryMode=0755
UMask=022

[Install]
WantedBy=multi-user.target
`

const launchdPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + launchdPlistLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/local/bin/minibox</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`
