package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"gopkg.in/yaml.v3"
)

func Parse(content []byte, format string) ([]Node, error) {
	format = NormalizeFormat(strings.ToLower(strings.TrimSpace(format)))
	switch format {
	case FormatAuto:
		if nodes, err := parseSingBox(content); err == nil {
			return nodes, nil
		}
		if nodes, err := parseClash(content); err == nil {
			return nodes, nil
		}
		if nodes, err := parseBase64URI(content); err == nil {
			return nodes, nil
		}
		return nil, fmt.Errorf("unable to detect subscription format")
	case FormatSingBox:
		return parseSingBox(content)
	case FormatClash:
		return parseClash(content)
	case FormatBase64, "uri":
		return parseBase64URI(content)
	default:
		return nil, fmt.Errorf("unsupported subscription format: %s", format)
	}
}

func parseSingBox(content []byte) ([]Node, error) {
	var root map[string]any
	if err := json.Unmarshal(content, &root); err != nil {
		return nil, err
	}

	outboundsRaw, ok := root["outbounds"]
	if !ok {
		return nil, fmt.Errorf("missing outbounds")
	}

	list, ok := outboundsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid outbounds format")
	}

	var nodes []Node
	for i, raw := range list {
		outMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		outType := readString(outMap, "type")
		if outType == "" || !IsActualOutboundType(outType) {
			continue
		}
		name := readString(outMap, "tag")
		if name == "" {
			name = fmt.Sprintf("%s-%d", outType, i+1)
		}
		delete(outMap, "tag")

		nodes = append(nodes, Node{
			Name:     name,
			Type:     outType,
			Outbound: outMap,
		})
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported outbounds found")
	}
	return nodes, nil
}

func parseClash(content []byte) ([]Node, error) {
	var root map[string]any
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, err
	}

	proxiesRaw, ok := root["proxies"]
	if !ok {
		return nil, fmt.Errorf("missing proxies")
	}

	list, ok := proxiesRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid proxies format")
	}

	var nodes []Node
	for _, raw := range list {
		proxyMap := asStringMap(raw)
		if proxyMap == nil {
			continue
		}

		proxyType := strings.ToLower(readString(proxyMap, "type"))
		a, err := GetAdapter(proxyType)
		if err != nil {
			logger.Debug("Skipping proxy node", "type", proxyType, "error", err.Error())
			continue
		}

		node, err := a.FromClash(proxyMap)
		if err != nil {
			logger.Debug("Failed to parse clash node", "type", proxyType, "error", err.Error())
			continue
		}

		name := readString(proxyMap, "name")
		if name != "" {
			node.Name = name
		} else if node.Name == "" {
			node.Name = fmt.Sprintf("%s-%v:%v", node.Type, proxyMap["server"], proxyMap["port"])
		}

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported proxies found")
	}
	return nodes, nil
}

func parseBase64URI(content []byte) ([]Node, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		decoded = content
	}

	lines := strings.Split(string(decoded), "\n")
	var nodes []Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		idx := strings.Index(line, "://")
		if idx < 0 {
			continue
		}

		scheme := strings.ToLower(line[:idx])
		a, err := GetAdapter(scheme)
		if err != nil {
			logger.Debug("Skipping proxy node", "scheme", scheme, "error", err.Error())
			continue
		}

		node, err := a.FromURI(line[idx+3:])
		if err != nil {
			logger.Debug("Failed to parse URI node", "scheme", scheme, "error", err.Error())
			continue
		}

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxy URIs found")
	}
	return nodes, nil
}
