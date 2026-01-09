package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyson/sing-helm/internal/logger"
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
		// 尝试 base64 URI 格式
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
		node, err := clashProxyToNode(proxyMap)
		if err != nil {
			// 记录跳过的节点，帮助调试
			name := readString(proxyMap, "name")
			proxyType := readString(proxyMap, "type")
			logger.Debug("Skipping proxy node", "name", name, "type", proxyType, "error", err.Error())
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
	case "hysteria":
		auth := readString(m, "auth_str", "auth-str", "auth")
		protocol := readString(m, "protocol")
		if protocol == "" {
			protocol = "udp"
		}
		outbound["type"] = "hysteria"
		if auth != "" {
			outbound["auth_str"] = auth
		}
		if upMbps := readInt(m, "up", "up-mbps"); upMbps > 0 {
			outbound["up_mbps"] = upMbps
		}
		if downMbps := readInt(m, "down", "down-mbps"); downMbps > 0 {
			outbound["down_mbps"] = downMbps
		}
		outbound["protocol"] = protocol
	case "hysteria2", "hy2":
		password := readString(m, "password")
		outbound["type"] = "hysteria2"
		outbound["password"] = password
		if upMbps := readInt(m, "up", "up-mbps"); upMbps > 0 {
			outbound["up_mbps"] = upMbps
		}
		if downMbps := readInt(m, "down", "down-mbps"); downMbps > 0 {
			outbound["down_mbps"] = downMbps
		}
	default:
		return Node{}, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	switch proxyType {
	case "vmess", "vless", "trojan", "hysteria", "hysteria2", "hy2":
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
	clientFingerprint := readString(m, "client-fingerprint")

	// 检查是否有 reality 配置
	realityOpts := asStringMap(m["reality-opts"])

	if !tlsEnabled && sni == "" && len(alpn) == 0 && realityOpts == nil && clientFingerprint == "" {
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
	if clientFingerprint != "" {
		tls["utls"] = map[string]any{
			"enabled":     true,
			"fingerprint": clientFingerprint,
		}
	}

	// 处理 reality 配置
	if realityOpts != nil {
		reality := map[string]any{
			"enabled": true,
		}
		if publicKey := readString(realityOpts, "public-key", "public_key"); publicKey != "" {
			reality["public_key"] = publicKey
		}
		if shortID := readString(realityOpts, "short-id", "short_id"); shortID != "" {
			reality["short_id"] = shortID
		}
		tls["reality"] = reality
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

// parseBase64URI 解析 base64 编码的 URI 订阅格式
// 格式: base64(vmess://...\nvless://...\n...)
func parseBase64URI(content []byte) ([]Node, error) {
	// 尝试 base64 解码
	decoded, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		// 可能已经是解码后的内容
		decoded = content
	}

	// 按行分割
	lines := strings.Split(string(decoded), "\n")
	var nodes []Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		node, err := parseProxyURI(line)
		if err != nil {
			logger.Debug("Skipping invalid URI", "uri", line[:min(len(line), 50)], "error", err.Error())
			continue
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxy URIs found")
	}
	return nodes, nil
}

// parseProxyURI 解析单个代理 URI
func parseProxyURI(uri string) (Node, error) {
	// 解析 scheme
	idx := strings.Index(uri, "://")
	if idx < 0 {
		return Node{}, fmt.Errorf("invalid URI format")
	}

	scheme := strings.ToLower(uri[:idx])
	rest := uri[idx+3:]

	switch scheme {
	case "vmess":
		return parseVMessURI(rest)
	case "vless":
		return parseVLessURI(rest)
	case "trojan":
		return parseTrojanURI(rest)
	case "ss", "shadowsocks":
		return parseShadowsocksURI(rest)
	case "hysteria":
		return parseHysteriaURI(rest)
	case "hysteria2", "hy2":
		return parseHysteria2URI(rest)
	default:
		return Node{}, fmt.Errorf("unsupported URI scheme: %s", scheme)
	}
}

// parseVLessURI 解析 vless:// URI
func parseVLessURI(uri string) (Node, error) {
	// 格式: vless://uuid@server:port?params#name
	u, err := url.Parse("vless://" + uri)
	if err != nil {
		return Node{}, err
	}

	uuid := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if uuid == "" || server == "" || port == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := parseInt(port)
	outbound := map[string]any{
		"type":        "vless",
		"server":      server,
		"server_port": portNum,
		"uuid":        uuid,
	}

	// 解析参数
	if flow := query.Get("flow"); flow != "" {
		outbound["flow"] = flow
	}

	// TLS 配置
	if security := query.Get("security"); security == "tls" || security == "reality" {
		tls := map[string]any{"enabled": true}
		if sni := query.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		if fp := query.Get("fp"); fp != "" {
			tls["utls"] = map[string]any{
				"enabled":     true,
				"fingerprint": fp,
			}
		}

		// Reality 配置
		if security == "reality" {
			reality := map[string]any{"enabled": true}
			if pbk := query.Get("pbk"); pbk != "" {
				reality["public_key"] = pbk
			}
			if sid := query.Get("sid"); sid != "" {
				reality["short_id"] = sid
			}
			tls["reality"] = reality
		}

		outbound["tls"] = tls
	}

	// 传输配置
	if network := query.Get("type"); network != "" {
		applyURITransport(outbound, network, query)
	}

	if name == "" {
		name = fmt.Sprintf("vless-%s:%s", server, port)
	}

	return Node{
		Name:     name,
		Type:     "vless",
		Outbound: outbound,
	}, nil
}

// parseVMessURI 解析 vmess:// URI (通常是 base64 编码的 JSON)
func parseVMessURI(uri string) (Node, error) {
	// VMess URI 通常是 base64 编码的 JSON
	decoded, err := base64.StdEncoding.DecodeString(uri)
	if err != nil {
		return Node{}, fmt.Errorf("invalid vmess URI: %w", err)
	}

	var vmessConfig map[string]any
	if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
		return Node{}, fmt.Errorf("invalid vmess config: %w", err)
	}

	server := readString(vmessConfig, "add", "address")
	port := readInt(vmessConfig, "port")
	uuid := readString(vmessConfig, "id", "uuid")
	name := readString(vmessConfig, "ps", "name")

	if server == "" || port == 0 || uuid == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	outbound := map[string]any{
		"type":        "vmess",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"security":    readString(vmessConfig, "scy", "security"),
	}

	if outbound["security"] == "" {
		outbound["security"] = "auto"
	}

	if alterID := readInt(vmessConfig, "aid", "alterId"); alterID > 0 {
		outbound["alter_id"] = alterID
	}

	// TLS
	if tls := readString(vmessConfig, "tls"); tls == "tls" {
		tlsConfig := map[string]any{"enabled": true}
		if sni := readString(vmessConfig, "sni"); sni != "" {
			tlsConfig["server_name"] = sni
		}
		outbound["tls"] = tlsConfig
	}

	// 传输
	if network := readString(vmessConfig, "net", "network"); network != "" {
		transport := map[string]any{"type": network}

		switch network {
		case "ws":
			if path := readString(vmessConfig, "path"); path != "" {
				transport["path"] = path
			}
			if host := readString(vmessConfig, "host"); host != "" {
				transport["headers"] = map[string]string{"Host": host}
			}
		case "grpc":
			if serviceName := readString(vmessConfig, "path", "serviceName"); serviceName != "" {
				transport["service_name"] = serviceName
			}
		}

		outbound["transport"] = transport
	}

	if name == "" {
		name = fmt.Sprintf("vmess-%s:%d", server, port)
	}

	return Node{
		Name:     name,
		Type:     "vmess",
		Outbound: outbound,
	}, nil
}

// parseTrojanURI 解析 trojan:// URI
func parseTrojanURI(uri string) (Node, error) {
	// 格式: trojan://password@server:port?params#name
	u, err := url.Parse("trojan://" + uri)
	if err != nil {
		return Node{}, err
	}

	password := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if password == "" || server == "" || port == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := parseInt(port)
	outbound := map[string]any{
		"type":        "trojan",
		"server":      server,
		"server_port": portNum,
		"password":    password,
	}

	// TLS 配置
	if security := query.Get("security"); security == "tls" || query.Get("tls") == "1" {
		tls := map[string]any{"enabled": true}
		if sni := query.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		outbound["tls"] = tls
	}

	// 传输配置
	if network := query.Get("type"); network != "" {
		applyURITransport(outbound, network, query)
	}

	if name == "" {
		name = fmt.Sprintf("trojan-%s:%s", server, port)
	}

	return Node{
		Name:     name,
		Type:     "trojan",
		Outbound: outbound,
	}, nil
}

// parseShadowsocksURI 解析 ss:// URI
func parseShadowsocksURI(uri string) (Node, error) {
	// 格式: ss://base64(method:password)@server:port#name
	// 或: ss://method:password@server:port#name

	parts := strings.SplitN(uri, "@", 2)
	if len(parts) != 2 {
		return Node{}, fmt.Errorf("invalid ss URI format")
	}

	// 尝试解码第一部分
	methodPassword := parts[0]
	decoded, err := base64.URLEncoding.DecodeString(methodPassword)
	if err == nil {
		methodPassword = string(decoded)
	}

	mpParts := strings.SplitN(methodPassword, ":", 2)
	if len(mpParts) != 2 {
		return Node{}, fmt.Errorf("invalid method:password format")
	}

	method := mpParts[0]
	password := mpParts[1]

	// 解析服务器和端口
	serverPart := parts[1]
	hashIdx := strings.Index(serverPart, "#")
	name := ""
	if hashIdx >= 0 {
		name, _ = url.QueryUnescape(serverPart[hashIdx+1:])
		serverPart = serverPart[:hashIdx]
	}

	spParts := strings.SplitN(serverPart, ":", 2)
	if len(spParts) != 2 {
		return Node{}, fmt.Errorf("invalid server:port format")
	}

	server := spParts[0]
	port, _ := parseInt(spParts[1])

	if name == "" {
		name = fmt.Sprintf("ss-%s:%d", server, port)
	}

	return Node{
		Name: name,
		Type: "shadowsocks",
		Outbound: map[string]any{
			"type":        "shadowsocks",
			"server":      server,
			"server_port": port,
			"method":      method,
			"password":    password,
		},
	}, nil
}

// parseHysteriaURI 解析 hysteria:// URI
func parseHysteriaURI(uri string) (Node, error) {
	u, err := url.Parse("hysteria://" + uri)
	if err != nil {
		return Node{}, err
	}

	auth := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if server == "" || port == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := parseInt(port)
	outbound := map[string]any{
		"type":        "hysteria",
		"server":      server,
		"server_port": portNum,
	}

	if auth != "" {
		outbound["auth_str"] = auth
	}

	if upMbps := query.Get("up"); upMbps != "" {
		if up, _ := parseInt(upMbps); up > 0 {
			outbound["up_mbps"] = up
		}
	}
	if downMbps := query.Get("down"); downMbps != "" {
		if down, _ := parseInt(downMbps); down > 0 {
			outbound["down_mbps"] = down
		}
	}

	// TLS
	tls := map[string]any{"enabled": true}
	if sni := query.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if query.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	outbound["tls"] = tls

	if name == "" {
		name = fmt.Sprintf("hysteria-%s:%s", server, port)
	}

	return Node{
		Name:     name,
		Type:     "hysteria",
		Outbound: outbound,
	}, nil
}

// parseHysteria2URI 解析 hysteria2:// URI
func parseHysteria2URI(uri string) (Node, error) {
	u, err := url.Parse("hysteria2://" + uri)
	if err != nil {
		return Node{}, err
	}

	password := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if password == "" || server == "" || port == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := parseInt(port)
	outbound := map[string]any{
		"type":        "hysteria2",
		"server":      server,
		"server_port": portNum,
		"password":    password,
	}

	if upMbps := query.Get("up"); upMbps != "" {
		if up, _ := parseInt(upMbps); up > 0 {
			outbound["up_mbps"] = up
		}
	}
	if downMbps := query.Get("down"); downMbps != "" {
		if down, _ := parseInt(downMbps); down > 0 {
			outbound["down_mbps"] = down
		}
	}

	// TLS
	tls := map[string]any{"enabled": true}
	if sni := query.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if query.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	outbound["tls"] = tls

	if name == "" {
		name = fmt.Sprintf("hysteria2-%s:%s", server, port)
	}

	return Node{
		Name:     name,
		Type:     "hysteria2",
		Outbound: outbound,
	}, nil
}

// applyURITransport 应用 URI 查询参数中的传输配置
func applyURITransport(outbound map[string]any, network string, query url.Values) {
	transport := map[string]any{"type": network}

	switch network {
	case "ws":
		if path := query.Get("path"); path != "" {
			transport["path"] = path
		}
		if host := query.Get("host"); host != "" {
			transport["headers"] = map[string]string{"Host": host}
		}
	case "grpc":
		if serviceName := query.Get("serviceName"); serviceName != "" {
			transport["service_name"] = serviceName
		}
	}

	outbound["transport"] = transport
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
