package module

import (
	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/subscription"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// SubscriptionNodeProvider reads subscription nodes from cache
type SubscriptionNodeProvider struct{}

func (p *SubscriptionNodeProvider) Name() string {
	return "subscription"
}

func (p *SubscriptionNodeProvider) GetNodes() ([]node.Node, error) {
	paths := paths.Get()
	sources, err := subscription.LoadSources(paths.SubConfigDir)
	if err != nil {
		logger.Error("Failed to load subscription sources", "error", err)
	}

	subNodes, err := subscription.LoadNodesFromCache(sources, paths.SubCacheDir)
	if err != nil {
		logger.Error("Failed to load subscription cache", "error", err)
		return nil, nil // Return empty list instead of failing the whole build
	}

	var nodes []node.Node
	for _, n := range subNodes {
		if n.Outbound == nil || n.Source == "" {
			continue
		}

		outboundCopy := make(map[string]any, len(n.Outbound))
		for k, v := range n.Outbound {
			outboundCopy[k] = v
		}

		nodes = append(nodes, node.Node{
			Name:     n.Name,
			Type:     n.Type,
			Source:   n.Source, // Provide the sub source name
			Outbound: outboundCopy,
		})
	}

	return nodes, nil
}
