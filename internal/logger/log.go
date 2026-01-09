package logger

import (
	"os"
	"path/filepath"
	"runtime"
)

// ResolveLogDir returns the preferred log directory, falling back to runtimeDir if needed.
func ResolveLogDir(runtimeDir string) string {
	candidate := ""
	switch runtime.GOOS {
	case "linux":
		candidate = filepath.Join("/var", "log", "sing-helm")
	case "darwin":
		candidate = filepath.Join("/var", "log", "sing-helm")
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = os.TempDir()
		}
		candidate = filepath.Join(base, "sing-helm", "logs")
	default:
		candidate = filepath.Join(os.TempDir(), "sing-helm", "logs")
	}

	if ensureWritableDir(candidate) {
		return candidate
	}
	if ensureWritableDir(runtimeDir) {
		return runtimeDir
	}
	return ""
}

func ensureWritableDir(dir string) bool {
	if dir == "" {
		return false
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, "sing-helm-perm-")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
