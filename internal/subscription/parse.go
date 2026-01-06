package subscription

import (
	"encoding/json"
	"fmt"
	"strings"

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
		return nil, fmt.Errorf("unable to detect subscription format")
	case FormatSingBox:
		return parseSingBox(content)
	case FormatClash:
		return parseClash(content)
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
		node, err := clashProxyToNode(proxyMap)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported proxies found")
	}
	return nodes, nil
}

func clashProxyToNode(m map[string]any) (Node, error) {
	name := strings.TrimSpace(readString(m, "name"))
	proxyType := strings.ToLower(readString(m, "type"))
	server := readString(m, "server")
	port := readInt(m, "port")
	if server == "" || port == 0 {
		return Node{}, fmt.Errorf("missing server or port")
	}

	outbound := map[string]any{
		"server":      server,
		"server_port": port,
	}

	switch proxyType {
	case "vmess":
		uuid := readString(m, "uuid")
		cipher := readString(m, "cipher", "security")
		if cipher == "" {
			cipher = "auto"
		}
		outbound["type"] = "vmess"
		outbound["uuid"] = uuid
		outbound["security"] = cipher
		if alterID := readInt(m, "alterId", "alter-id"); alterID > 0 {
			outbound["alter_id"] = alterID
		}
	case "vless":
		uuid := readString(m, "uuid")
		outbound["type"] = "vless"
		outbound["uuid"] = uuid
		if flow := readString(m, "flow"); flow != "" {
			outbound["flow"] = flow
		}
	case "trojan":
		password := readString(m, "password")
		outbound["type"] = "trojan"
		outbound["password"] = password
	case "ss", "shadowsocks":
		password := readString(m, "password")
		cipher := readString(m, "cipher")
		outbound["type"] = "shadowsocks"
		outbound["password"] = password
		outbound["method"] = cipher
		if plugin := readString(m, "plugin"); plugin != "" {
			outbound["plugin"] = plugin
		}
		if pluginOpts := readString(m, "plugin-opts", "plugin_opts"); pluginOpts != "" {
			outbound["plugin_opts"] = pluginOpts
		}
	default:
		return Node{}, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	switch proxyType {
	case "vmess", "vless", "trojan":
		applyTLSOptions(outbound, m)
		applyTransportOptions(outbound, m)
	}

	outType := readString(outbound, "type")
	if name == "" {
		name = fmt.Sprintf("%s-%s:%d", outType, server, port)
	}

	return Node{
		Name:     name,
		Type:     outType,
		Outbound: outbound,
	}, nil
}

func applyTLSOptions(outbound map[string]any, m map[string]any) {
	tlsEnabled := readBool(m, "tls")
	sni := readString(m, "sni", "servername", "server_name")
	skipVerify := readBool(m, "skip-cert-verify")
	alpn := readStringList(m, "alpn")

	if !tlsEnabled && sni == "" && len(alpn) == 0 {
		return
	}

	tls := map[string]any{
		"enabled": true,
	}
	if sni != "" {
		tls["server_name"] = sni
	}
	if skipVerify {
		tls["insecure"] = true
	}
	if len(alpn) > 0 {
		tls["alpn"] = alpn
	}
	outbound["tls"] = tls
}

func applyTransportOptions(outbound map[string]any, m map[string]any) {
	network := strings.ToLower(readString(m, "network"))
	switch network {
	case "ws", "websocket":
		wsOpts := asStringMap(m["ws-opts"])
		transport := map[string]any{
			"type": "ws",
		}
		if wsOpts != nil {
			if path := readString(wsOpts, "path"); path != "" {
				transport["path"] = path
			}
			headers := asStringMap(wsOpts["headers"])
			if len(headers) > 0 {
				transport["headers"] = normalizeStringMap(headers)
			}
		}
		outbound["transport"] = transport
	case "grpc":
		grpcOpts := asStringMap(m["grpc-opts"])
		transport := map[string]any{
			"type": "grpc",
		}
		if grpcOpts != nil {
			if service := readString(grpcOpts, "grpc-service-name", "service-name"); service != "" {
				transport["service_name"] = service
			}
		}
		outbound["transport"] = transport
	}
}

func readString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case string:
				return v
			case fmt.Stringer:
				return v.String()
			}
		}
	}
	return ""
}

func readInt(m map[string]any, keys ...string) int {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case uint64:
				return int(v)
			case uint32:
				return int(v)
			case float32:
				return int(v)
			case string:
				if parsed, err := parseInt(v); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func readBool(m map[string]any, key string) bool {
	val, ok := m[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || v == "1"
	default:
		return false
	}
}

func readStringList(m map[string]any, key string) []string {
	val, ok := m[key]
	if !ok {
		return nil
	}
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

func asStringMap(val any) map[string]any {
	switch v := val.(type) {
	case map[string]any:
		return v
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			out[fmt.Sprint(key)] = value
		}
		return out
	default:
		return nil
	}
}

func normalizeStringMap(input map[string]any) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, val := range input {
		switch v := val.(type) {
		case string:
			out[key] = v
		default:
			out[key] = fmt.Sprint(v)
		}
	}
	return out
}

func parseInt(value string) (int, error) {
	var out int
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &out)
	return out, err
}
