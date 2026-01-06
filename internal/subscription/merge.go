package subscription

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func LoadNodesFromCache(sources []Source, cacheDir string) ([]Node, error) {
	seen := make(map[string]bool)
	var nodes []Node

	for _, source := range sources {
		if !source.EnabledValue() {
			continue
		}
		cachePath := CacheFilePath(cacheDir, source.Name)
		cache, err := LoadCache(cachePath)
		if err != nil {
			continue
		}

		for _, node := range cache.Nodes {
			normalizeNode(&node, source)
			if node.Type == "" {
				continue
			}
			if source.DedupeValue() {
				hash, err := outboundHash(node.Outbound)
				if err != nil {
					continue
				}
				if seen[hash] {
					continue
				}
				seen[hash] = true
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func normalizeNode(node *Node, source Source) {
	if node.Source == "" {
		node.Source = source.Name
	}
	if node.Type == "" {
		node.Type = readString(node.Outbound, "type")
	}
	if node.Name == "" {
		node.Name = readString(node.Outbound, "tag")
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s-%s", node.Type, node.Source)
	}
	delete(node.Outbound, "tag")
}

func outboundHash(outbound map[string]any) (string, error) {
	if outbound == nil {
		return "", fmt.Errorf("empty outbound")
	}
	cloned := make(map[string]any, len(outbound))
	for key, value := range outbound {
		if key == "tag" {
			continue
		}
		cloned[key] = value
	}
	data, err := json.Marshal(cloned)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:]), nil
}
