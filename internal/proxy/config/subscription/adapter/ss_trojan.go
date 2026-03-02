package adapter

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
)

// ShadowsocksAdapter handles Shadowsocks protocol
type ShadowsocksAdapter struct{}

func init() {
	Register("ss", &ShadowsocksAdapter{})
	Register("shadowsocks", &ShadowsocksAdapter{})
}

func (a *ShadowsocksAdapter) FromClash(m map[string]any) (model.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return model.Node{}, fmt.Errorf("missing server or port")
	}

	password := ReadString(m, "password")
	cipher := ReadString(m, "cipher")

	outbound := map[string]any{
		"type":        "shadowsocks",
		"server":      server,
		"server_port": port,
		"password":    password,
		"method":      cipher,
	}

	if plugin := ReadString(m, "plugin"); plugin != "" {
		outbound["plugin"] = plugin
	}
	if pluginOpts := ReadString(m, "plugin-opts", "plugin_opts"); pluginOpts != "" {
		outbound["plugin_opts"] = pluginOpts
	}

	return model.Node{
		Type:     "shadowsocks",
		Outbound: outbound,
	}, nil
}

func (a *ShadowsocksAdapter) FromURI(uriStr string) (model.Node, error) {
	parts := strings.SplitN(uriStr, "@", 2)
	if len(parts) != 2 {
		return model.Node{}, fmt.Errorf("invalid ss URI format")
	}

	methodPassword := parts[0]
	decoded, err := base64.URLEncoding.DecodeString(methodPassword)
	if err == nil {
		methodPassword = string(decoded)
	}

	mpParts := strings.SplitN(methodPassword, ":", 2)
	if len(mpParts) != 2 {
		return model.Node{}, fmt.Errorf("invalid method:password format")
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
		return model.Node{}, fmt.Errorf("invalid server:port format")
	}

	server := spParts[0]
	port, _ := ParseInt(spParts[1])

	return model.Node{
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
	Register("trojan", &TrojanAdapter{})
}

func (a *TrojanAdapter) FromClash(m map[string]any) (model.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return model.Node{}, fmt.Errorf("missing server or port")
	}

	password := ReadString(m, "password")

	outbound := map[string]any{
		"type":        "trojan",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	ApplyTLSOptions(outbound, m)
	ApplyTransportOptions(outbound, m)

	return model.Node{
		Type:     "trojan",
		Outbound: outbound,
	}, nil
}

func (a *TrojanAdapter) FromURI(uriStr string) (model.Node, error) {
	u, err := url.Parse("trojan://" + uriStr)
	if err != nil {
		return model.Node{}, err
	}

	password := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if password == "" || server == "" || port == "" {
		return model.Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := ParseInt(port)
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

	return model.Node{
		Name:     name,
		Type:     "trojan",
		Outbound: outbound,
	}, nil
}
