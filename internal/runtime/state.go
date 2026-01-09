package runtime

import (
	"encoding/json"
	"os"

	"github.com/kysonzou/sing-helm/internal/env"
)

type RuntimeState struct {
	RunOptions RunOptions `json:"run_options"`
	PID        int        `json:"pid"`
}

func GetStatePath() string {
	return env.Get().StateFile
}

func SaveState(s *RuntimeState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
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
