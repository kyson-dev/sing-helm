package cli

import (
	"os"
	"path/filepath"

	"github.com/kyson-dev/sing-helm/internal/app/daemon"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// SetupEnvironment initializes the global paths.
// homeFlag: --home parameter from CLI
// Logic:
// 1. If homeFlag is set, use it.
// 2. Otherwise, prioritize the daemon's configured home.
// 3. Fallback to default ~/.sing-helm
// 4. Initialize paths
func SetupEnvironment(homeFlag string) error {
	resolvedHome := ""

	if homeFlag != "" {
		resolvedHome = homeFlag
	} else {
		if runtimeHome := daemon.FindRuntimeConfigHome(); runtimeHome != "" {
			resolvedHome = runtimeHome
		} else {
			if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
				resolvedHome = filepath.Join("/Users", sudoUser, ".sing-helm")
			} else {
				userHome, _ := os.UserHomeDir()
				resolvedHome = filepath.Join(userHome, ".sing-helm")
			}
		}
	}

	return paths.Init(resolvedHome)
}
