package module

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// UserNodeProvider reads raw nodes from user's profile.json
type UserNodeProvider struct{}

func (p *UserNodeProvider) Name() string {
	return "user"
}

func (p *UserNodeProvider) GetNodes() ([]Node, error) {
	paths := paths.Get()

	content, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error if file doesn't exist
		}
		return nil, err
	}

	if len(bytes.TrimSpace(content)) == 0 {
		return nil, nil
	}

	var rawConfig map[string]any
	if err := json.Unmarshal(content, &rawConfig); err != nil {
		return nil, err
	}

	var nodes []Node
	if rawOutboundsVal, ok := rawConfig["outbounds"]; ok {
		if list, ok := rawOutboundsVal.([]any); ok {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					tag := ""
					if t, ok := m["tag"].(string); ok {
						tag = t
					}
					outType := ""
					if t, ok := m["type"].(string); ok {
						outType = t
					}
					nodes = append(nodes, Node{
						Name:     tag,
						Type:     outType,
						Source:   "user", // Explicitly mark source
						Outbound: m,
					})
				}
			}
		}
	}

	return nodes, nil
}
