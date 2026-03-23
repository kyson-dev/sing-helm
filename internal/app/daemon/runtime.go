package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kyson-dev/sing-helm/internal/sys/lock"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// FindRuntimeConfigHome returns the config home from a running system daemon, if any.
func FindRuntimeConfigHome() string {
	runtimeDir := paths.ResolveRuntimeDir()
	if runtimeDir == "" {
		return ""
	}
	if err := lock.CheckLock(paths.GetRuntimeLockFileWithDir(runtimeDir)); err != nil {
		return ""
	}

	meta, err := loadRuntimeMeta(paths.GetRuntimeMetaFileWithDir(runtimeDir))
	if err != nil || meta == nil {
		return ""
	}
	if meta.ConfigHome == "" {
		return ""
	}
	if !fileExists(paths.GetProfileFileWithDir(meta.ConfigHome)) {
		return ""
	}
	return meta.ConfigHome
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// RuntimeMeta holds system status, such as the config path used by the running daemon.
type RuntimeMeta struct {
	ConfigHome string `json:"config_home"`
}

func saveRuntimeMeta(path string, meta RuntimeMeta) error {
	if path == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func loadRuntimeMeta(path string) (*RuntimeMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta RuntimeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}