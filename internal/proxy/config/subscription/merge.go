package subscription

import (
	"path/filepath"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
)

// LoadNodesFromCache reads from cache files honoring priority and enablement
func LoadNodesFromCache(sources []Source, cacheDir string) ([]model.Node, error) {
	var finalNodes []model.Node
	for _, s := range sources {
		if !s.EnabledValue() {
			logger.Debug("Skipping disabled source", "name", s.Name)
			continue
		}

		cachePath := filepath.Join(cacheDir, s.Name+".json")
		cache, err := LoadCache(cachePath)
		if err != nil {
			logger.Error("Failed to load cache for source", "name", s.Name, "error", err)
			continue
		}

		nodes := cache.Nodes
		if len(nodes) == 0 {
			continue
		}

		// Apply tags to nodes
		if len(s.Tags) > 0 {
			nodes = appendTags(nodes, s.Tags)
		}

		// Pass dedupe intention to the node level
		for _, n := range nodes {
			n.Source = s.Name
			n.SkipDedupe = !s.DedupeValue()
			finalNodes = append(finalNodes, n)
		}
	}

	return finalNodes, nil
}

func appendTags(nodes []model.Node, tags []string) []model.Node {
	for i := range nodes {
		for _, tag := range tags {
			if !strings.Contains(nodes[i].Name, tag) {
				nodes[i].Name = nodes[i].Name + " " + tag
			}
		}
	}
	return nodes
}
