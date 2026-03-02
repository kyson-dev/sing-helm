package node

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// UserNodeProvider reads nodes directly from the user's config.
type UserNodeProvider struct{}

func (p *UserNodeProvider) Name() string {
	return "user"
}

func (p *UserNodeProvider) GetNodes() ([]model.Node, error) {
	paths := paths.Get()
	profileData, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty if profile does not exist yet
		}
		return nil, fmt.Errorf("read profile error: %w", err)
	}

	if len(profileData) == 0 {
		return nil, nil // Return empty if profile is a 0-byte file
	}

	var root map[string]any
	if err := json.Unmarshal(profileData, &root); err != nil {
		logger.Error("Failed to parse profile.json, skipping user nodes", "error", err)
		return nil, nil
	}

	outboundsRaw, ok := root["outbounds"]
	if !ok {
		return nil, nil
	}

	list, ok := outboundsRaw.([]any)
	if !ok {
		logger.Info("user outbounds is not a list")
		return nil, nil
	}

	var nodes []model.Node
	for i, raw := range list {
		outMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		outType, _ := outMap["type"].(string)
		if outType == "" || !IsActualOutboundType(outType) {
			continue // skip direct, block, dns, etc. They are handled globally.
		}

		name, _ := outMap["tag"].(string)
		if name == "" {
			name = fmt.Sprintf("user-%s-%d", outType, i+1)
		}
		delete(outMap, "tag")

		nodes = append(nodes, model.Node{
			Name:     name,
			Type:     outType,
			Source:   "user", // Indicates it came from user config
			Outbound: outMap,
		})
	}

	return nodes, nil
}
