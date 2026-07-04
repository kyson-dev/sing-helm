package sysnet

import (
	"os/exec"
	"strings"
)

// SetSystemDNS configures macOS system DNS on all active network services to
// the given server address. Requires admin privileges (prompted by macOS if not root).
func SetSystemDNS(server string) error {
	services, err := listNetworkServices()
	if err != nil {
		return err
	}
	for _, service := range services {
		// Non-fatal per service: inactive services will fail silently.
		_ = exec.Command("networksetup", "-setdnsservers", service, server).Run()
	}
	return nil
}

// RestoreSystemDNS resets all active macOS network services back to automatic
// (DHCP) DNS. Best-effort: errors on individual services are ignored.
func RestoreSystemDNS() error {
	services, err := listNetworkServices()
	if err != nil {
		return err
	}
	for _, service := range services {
		_ = exec.Command("networksetup", "-setdnsservers", service, "Empty").Run()
	}
	return nil
}

func listNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	var services []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		// First line is a header note; skip blank and asterisk-prefixed lines.
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}
