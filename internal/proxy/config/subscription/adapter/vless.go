package adapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
)

// VMessAdapter handles VMess protocol in Clash and URI formats.
type VMessAdapter struct{}

func init() {
	Register("vmess", &VMessAdapter{})
}

func (a *VMessAdapter) FromClash(m map[string]any) (model.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return model.Node{}, fmt.Errorf("missing server or port")
	}

	uuid := ReadString(m, "uuid")
	cipher := ReadString(m, "cipher", "security")
	if cipher == "" {
		cipher = "auto"
	}

	outbound := map[string]any{
		"type":        "vmess",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"security":    cipher,
	}

	if alterID := ReadInt(m, "alterId", "alter-id"); alterID > 0 {
		outbound["alter_id"] = alterID
	}

	ApplyTLSOptions(outbound, m)
	ApplyTransportOptions(outbound, m)

	return model.Node{
		Type:     "vmess",
		Outbound: outbound,
	}, nil
}

func (a *VMessAdapter) FromURI(uri string) (model.Node, error) {
	decoded, err := base64.StdEncoding.DecodeString(uri)
	if err != nil {
		return model.Node{}, fmt.Errorf("invalid vmess URI: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(decoded, &m); err != nil {
		return model.Node{}, fmt.Errorf("invalid vmess config: %w", err)
	}

	server := ReadString(m, "add", "address")
	port := ReadInt(m, "port")
	uuid := ReadString(m, "id", "uuid")
	name := ReadString(m, "ps", "name")

	if server == "" || port == 0 || uuid == "" {
		return model.Node{}, fmt.Errorf("missing required fields")
	}

	outbound := map[string]any{
		"type":        "vmess",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"security":    ReadString(m, "scy", "security"),
	}

	if outbound["security"] == "" {
		outbound["security"] = "auto"
	}

	if alterID := ReadInt(m, "aid", "alterId"); alterID > 0 {
		outbound["alter_id"] = alterID
	}

	if tls := ReadString(m, "tls"); tls == "tls" {
		tlsConfig := map[string]any{"enabled": true}
		if sni := ReadString(m, "sni"); sni != "" {
			tlsConfig["server_name"] = sni
		}
		outbound["tls"] = tlsConfig
	}

	if network := ReadString(m, "net", "network"); network != "" {
		transport := map[string]any{"type": network}
		switch network {
		case "ws":
			if path := ReadString(m, "path"); path != "" {
				transport["path"] = path
			}
			if host := ReadString(m, "host"); host != "" {
				transport["headers"] = map[string]string{"Host": host}
			}
		case "grpc":
			if serviceName := ReadString(m, "path", "serviceName"); serviceName != "" {
				transport["service_name"] = serviceName
			}
		}
		outbound["transport"] = transport
	}

	return model.Node{
		Name:     name,
		Type:     "vmess",
		Outbound: outbound,
	}, nil
}

// VLessAdapter handles VLess protocol in Clash and URI formats.
type VLessAdapter struct{}

func init() {
	Register("vless", &VLessAdapter{})
}

func (a *VLessAdapter) FromClash(m map[string]any) (model.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return model.Node{}, fmt.Errorf("missing server or port")
	}

	uuid := ReadString(m, "uuid")
	outbound := map[string]any{
		"type":        "vless",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow := ReadString(m, "flow"); flow != "" {
		outbound["flow"] = flow
	}

	ApplyTLSOptions(outbound, m)
	ApplyTransportOptions(outbound, m)

	return model.Node{
		Type:     "vless",
		Outbound: outbound,
	}, nil
}

func (a *VLessAdapter) FromURI(uriStr string) (model.Node, error) {
	u, err := url.Parse("vless://" + uriStr)
	if err != nil {
		return model.Node{}, err
	}

	uuid := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if uuid == "" || server == "" || port == "" {
		return model.Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := ParseInt(port)
	outbound := map[string]any{
		"type":        "vless",
		"server":      server,
		"server_port": portNum,
		"uuid":        uuid,
	}

	if flow := query.Get("flow"); flow != "" {
		outbound["flow"] = flow
	}

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

	if network := query.Get("type"); network != "" {
		ApplyURITransport(outbound, network, query)
	}

	return model.Node{
		Name:     name,
		Type:     "vless",
		Outbound: outbound,
	}, nil
}
