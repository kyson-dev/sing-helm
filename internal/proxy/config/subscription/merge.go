package subscription

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
)

// LoadNodesFromCache reads from cache files honoring priority and enablement
func LoadNodesFromCache(sources []Source, cacheDir string) ([]node.Node, error) {
	var finalNodes []node.Node
	globalSeen := make(map[string]bool)

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

		// Source deduplication
		if s.DedupeValue() {
			nodes = dedupeWithinSource(nodes)
		}

		// Global deduplication (across sources)
		for _, n := range nodes {
			// use standard signature for global dedupe
			sig := globalSignature(n)
			if !globalSeen[sig] {
				globalSeen[sig] = true
				n.Source = s.Name
				finalNodes = append(finalNodes, n)
			}
		}
	}

	return finalNodes, nil
}

func appendTags(nodes []node.Node, tags []string) []node.Node {
	for i := range nodes {
		for _, tag := range tags {
			if !strings.Contains(nodes[i].Name, tag) {
				nodes[i].Name = nodes[i].Name + " " + tag
			}
		}
	}
	return nodes
}

func dedupeWithinSource(nodes []node.Node) []node.Node {
	seen := make(map[string]bool)
	var deduped []node.Node
	for _, n := range nodes {
		sig := localSignature(n)
		if !seen[sig] {
			seen[sig] = true
			deduped = append(deduped, n)
		}
	}
	return deduped
}

// localSignature is used for deduplication within the same source.
// We use name + type as signature since many users only have one server
// but use name to distinguish them.
func localSignature(n node.Node) string {
	return n.Name + "|" + n.Type
}

// globalSignature is used for cross-source deduplication.
// We try to use server+port combination if possible, fallback to localSignature.
func globalSignature(n node.Node) string {
	if n.Outbound != nil {
		server, hasServer := n.Outbound["server"].(string)
		port, hasPort := n.Outbound["server_port"]
		if hasServer && hasPort {
			return fmt.Sprintf("%s:%v|%s", server, port, n.Type)
		}
	}
	return localSignature(n)
}
