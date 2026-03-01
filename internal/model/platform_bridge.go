package model

import "github.com/kyson-dev/sing-helm/internal/platform"

// platformGetStateFile returns the state file path from the global platform config.
// This is isolated here so state.go doesn't directly import platform,
// making it easier to eventually remove this dependency.
func platformGetStateFile() string {
	return platform.Get().StateFile
}
