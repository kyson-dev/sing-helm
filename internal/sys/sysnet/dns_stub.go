//go:build !darwin

package sysnet

// SetSystemDNS is a no-op on non-macOS platforms.
func SetSystemDNS(server string) error {
	return nil
}

// RestoreSystemDNS is a no-op on non-macOS platforms.
func RestoreSystemDNS() error {
	return nil
}
