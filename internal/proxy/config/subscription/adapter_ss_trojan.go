package subscription

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// ShadowsocksAdapter handles Shadowsocks protocol
type ShadowsocksAdapter struct{}

func init() {
	RegisterAdapter("ss", &ShadowsocksAdapter{})
	RegisterAdapter("shadowsocks", &ShadowsocksAdapter{})
}

func (a *ShadowsocksAdapter) FromClash(m map[string]any) (Node, error) {
	server := readString(m, "server")
	port := readInt(m, "port")
	if server == "" || port == 0 {
		return Node{}, fmt.Errorf("missing server or port")
	}

	password := readString(m, "password")
	cipher := readString(m, "cipher")

	outbound := map[string]any{
		"type":        "shadowsocks",
		"server":      server,
		"server_port": port,
		"password":    password,
		"method":      cipher,
	}

	if plugin := readString(m, "plugin"); plugin != "" {
		outbound["plugin"] = plugin
	}
	if pluginOpts := readString(m, "plugin-opts", "plugin_opts"); pluginOpts != "" {
		outbound["plugin_opts"] = pluginOpts
	}

	return Node{
		Type:     "shadowsocks",
		Outbound: outbound,
	}, nil
}

func (a *ShadowsocksAdapter) FromURI(uriStr string) (Node, error) {
	parts := strings.SplitN(uriStr, "@", 2)
	if len(parts) != 2 {
		return Node{}, fmt.Errorf("invalid ss URI format")
	}

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

// TrojanAdapter handles Trojan protocol
type TrojanAdapter struct{}

func init() {
	RegisterAdapter("trojan", &TrojanAdapter{})
}

func (a *TrojanAdapter) FromClash(m map[string]any) (Node, error) {
	server := readString(m, "server")
	port := readInt(m, "port")
	if server == "" || port == 0 {
		return Node{}, fmt.Errorf("missing server or port")
	}

	password := readString(m, "password")

	outbound := map[string]any{
		"type":        "trojan",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	ApplyTLSOptions(outbound, m)
	ApplyTransportOptions(outbound, m)

	return Node{
		Type:     "trojan",
		Outbound: outbound,
	}, nil
}

func (a *TrojanAdapter) FromURI(uriStr string) (Node, error) {
	u, err := url.Parse("trojan://" + uriStr)
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

	if security := query.Get("security"); security == "tls" || query.Get("tls") == "1" {
		tls := map[string]any{"enabled": true}
		if sni := query.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		outbound["tls"] = tls
	}

	if network := query.Get("type"); network != "" {
		ApplyURITransport(outbound, network, query)
	}

	return Node{
		Name:     name,
		Type:     "trojan",
		Outbound: outbound,
	}, nil
}
