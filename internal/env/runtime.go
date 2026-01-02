package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const runtimeDirEnv = "MINIBOX_RUNTIME_DIR"

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
			return filepath.Join("/run", "minibox")
		}
		return filepath.Join("/var/run", "minibox")
	case "darwin":
		return filepath.Join("/var/run", "minibox")
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = os.TempDir()
		}
		return filepath.Join(base, "minibox")
	default:
		return filepath.Join(os.TempDir(), "minibox")
	}
}

// EnsureRuntimeDirs ensures runtime and log directories exist and are writable.
func EnsureRuntimeDirs(runtimeDir, logFile string) error {
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

// SetRuntimeDir overrides runtime directory resolution (tests only).
func SetRuntimeDir(dir string) {
	runtimeDirOverride = dir
}

// ResetRuntimeDir clears the runtime directory override.
func ResetRuntimeDir() {
	runtimeDirOverride = ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FindRuntimeConfigHome returns the config home from a running system daemon, if any.
func FindRuntimeConfigHome() string {
	runtimeDir := ResolveRuntimeDir()
	if runtimeDir == "" {
		return ""
	}
	if err := CheckLock(runtimeDir); err != nil {
		return ""
	}

	meta, err := LoadRuntimeMeta(runtimeDir)
	if err != nil || meta == nil {
		return ""
	}
	if meta.ConfigHome == "" {
		return ""
	}
	if !fileExists(filepath.Join(meta.ConfigHome, "profile.json")) {
		return ""
	}
	return meta.ConfigHome
}

type RuntimeMeta struct {
	ConfigHome string `json:"config_home"`
}

func runtimeMetaPath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "runtime.json")
}

func SaveRuntimeMeta(runtimeDir string, meta RuntimeMeta) error {
	if runtimeDir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(runtimeMetaPath(runtimeDir), data, 0644)
}

func LoadRuntimeMeta(runtimeDir string) (*RuntimeMeta, error) {
	data, err := os.ReadFile(runtimeMetaPath(runtimeDir))
	if err != nil {
		return nil, err
	}
	var meta RuntimeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
