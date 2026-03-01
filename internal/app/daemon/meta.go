package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kyson-dev/sing-helm/internal/sys/lock"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// RuntimeMeta holds system status, such as the config path used by the running daemon.
type RuntimeMeta struct {
	ConfigHome string `json:"config_home"`
}

func runtimeMetaPath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "runtime.json")
}

// SaveRuntimeMeta saves daemon configurations metadata to the runtime directory.
func SaveRuntimeMeta(runtimeDir string, meta RuntimeMeta) error {
	if runtimeDir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runtimeMetaPath(runtimeDir), data, 0644)
}

// LoadRuntimeMeta reads the daemon metadata from the runtime directory.
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

// FindRuntimeConfigHome returns the config home from a running system daemon, if any.
func FindRuntimeConfigHome() string {
	runtimeDir := paths.ResolveRuntimeDir()
	if runtimeDir == "" {
		return ""
	}

	lockFile := filepath.Join(runtimeDir, "sing-helm.lock")
	if err := lock.CheckLock(lockFile); err != nil {
		// daemon not running or lock missing
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
