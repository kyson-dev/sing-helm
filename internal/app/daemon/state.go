package daemon

import (
	"encoding/json"
	"os"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

type RuntimeState struct {
	RunOptions model.RunOptions `json:"run_options"`
	PID        int              `json:"pid"`
}

// SaveState saves runtime state to the given path.
func SaveState(s *RuntimeState) error {
	return SaveStateTo(defaultStatePath(), s)
}

// LoadState loads runtime state from the default path.
func LoadState() (*RuntimeState, error) {
	return LoadStateFrom(defaultStatePath())
}

// SaveStateTo saves runtime state to a specific path (DI-friendly).
func SaveStateTo(path string, s *RuntimeState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadStateFrom loads runtime state from a specific path (DI-friendly).
func LoadStateFrom(path string) (*RuntimeState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s RuntimeState
	err = json.Unmarshal(data, &s)
	return &s, err
}

// defaultStatePath returns the state file path from global platform config.
// This is kept for backward compatibility during migration.
func defaultStatePath() string {
	// Lazy import to avoid circular dependency at package level.
	// Uses the global singleton — callers that want DI should use SaveStateTo/LoadStateFrom.
	return paths.Get().StateFile
}
