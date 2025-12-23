package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type RuntimeState struct {
	RunOptions `json:"run_options"`
	PID     int `json:"pid"`
}

func GetStatePath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".minibox")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "state.json")
}

func SaveState(s *RuntimeState) error {
	data, _ := json.Marshal(s)
	return os.WriteFile(GetStatePath(), data, 0644)
}

func LoadState() (*RuntimeState, error) {
	data, err := os.ReadFile(GetStatePath())
	if err != nil {
		return nil, err
	}
	var s RuntimeState
	err = json.Unmarshal(data, &s)
	return &s, err
}
