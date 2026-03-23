package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const runtimeDirEnv = "SINGHELM_RUNTIME_DIR"

var runtimeDirOverride string

// ResolveRuntimeDir returns the system-level runtime directory for sockets/locks/logs/state.
func ResolveRuntimeDir() string {
	if runtimeDirOverride != "" {
		return runtimeDirOverride
	}
	if dir := os.Getenv(runtimeDirEnv); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "linux":
		if dirExists("/run") {
			return filepath.Join("/run", "sing-helm")
		}
		return filepath.Join("/var/run", "sing-helm")
	case "darwin":
		// Use /usr/local/var/run instead of /var/run because /var/run is tmpfs on macOS
		// and gets cleared on reboot
		return filepath.Join("/usr/local/var/run", "sing-helm")
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = os.TempDir()
		}
		return filepath.Join(base, "sing-helm")
	default:
		return filepath.Join(os.TempDir(), "sing-helm")
	}
}

// EnsureRuntimeDirs ensures runtime and log directories exist and are writable.
func ensureRuntimeDirs(runtimeDir, logFile string) error {
	if runtimeDir != "" {
		if err := os.MkdirAll(runtimeDir, 0755); err != nil {
			return err
		}
	}
	if logFile == "" {
		return nil
	}
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	if err := ensureWritableLogFile(logFile); err != nil {
		return err
	}
	return nil
}

func ensureWritableLogFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(path, 0644); err != nil {
		return err
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}



// SetRuntimeDir overrides runtime directory resolution (tests only).
func ForTestSetRuntimeDir(dir string) {
	runtimeDirOverride = dir
}

// ResetRuntimeDir clears the runtime directory override.
func ForTestResetRuntimeDir() {
	runtimeDirOverride = ""
}
