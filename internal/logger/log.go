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
	case "linux", "darwin":
		candidate = filepath.Join("/var", "log", "minibox")
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = os.TempDir()
		}
		candidate = filepath.Join(base, "minibox", "logs")
	default:
		candidate = filepath.Join(os.TempDir(), "minibox", "logs")
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
	f, err := os.CreateTemp(dir, "minibox-perm-")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
