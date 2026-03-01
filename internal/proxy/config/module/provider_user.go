package module

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// UserNodeProvider reads nodes directly from the user's config.
type UserNodeProvider struct{}

func (p *UserNodeProvider) Name() string {
	return "user"
}

func (p *UserNodeProvider) GetNodes() ([]node.Node, error) {
	paths := paths.Get()
	profileData, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty if profile does not exist yet
		}
		return nil, fmt.Errorf("read profile error: %w", err)
	}

	var root map[string]any
	if err := json.Unmarshal(profileData, &root); err != nil {
		return nil, fmt.Errorf("unmarshal profile error: %w", err)
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

	var nodes []node.Node
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

		nodes = append(nodes, node.Node{
			Name:     name,
			Type:     outType,
			Source:   "user", // Indicates it came from user config
			Outbound: outMap,
		})
	}

	return nodes, nil
}
