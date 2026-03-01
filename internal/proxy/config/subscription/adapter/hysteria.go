package adapter

import (
	"fmt"
	"net/url"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
)

// HysteriaAdapter handles Hysteria protocol
type HysteriaAdapter struct{}

func init() {
	Register("hysteria", &HysteriaAdapter{})
}

func (a *HysteriaAdapter) FromClash(m map[string]any) (node.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return node.Node{}, fmt.Errorf("missing server or port")
	}

	auth := ReadString(m, "auth_str", "auth-str", "auth")
	protocol := ReadString(m, "protocol")
	if protocol == "" {
		protocol = "udp"
	}

	outbound := map[string]any{
		"type":        "hysteria",
		"server":      server,
		"server_port": port,
		"protocol":    protocol,
	}

	if auth != "" {
		outbound["auth_str"] = auth
	}
	if upMbps := ReadInt(m, "up", "up-mbps"); upMbps > 0 {
		outbound["up_mbps"] = upMbps
	}
	if downMbps := ReadInt(m, "down", "down-mbps"); downMbps > 0 {
		outbound["down_mbps"] = downMbps
	}

	ApplyTLSOptions(outbound, m)

	return node.Node{
		Type:     "hysteria",
		Outbound: outbound,
	}, nil
}

func (a *HysteriaAdapter) FromURI(uriStr string) (node.Node, error) {
	u, err := url.Parse("hysteria://" + uriStr)
	if err != nil {
		return node.Node{}, err
	}

	auth := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if server == "" || port == "" {
		return node.Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := ParseInt(port)
	outbound := map[string]any{
		"type":        "hysteria",
		"server":      server,
		"server_port": portNum,
	}

	if auth != "" {
		outbound["auth_str"] = auth
	}

	if upMbps := query.Get("up"); upMbps != "" {
		if up, _ := ParseInt(upMbps); up > 0 {
			outbound["up_mbps"] = up
		}
	}
	if downMbps := query.Get("down"); downMbps != "" {
		if down, _ := ParseInt(downMbps); down > 0 {
			outbound["down_mbps"] = down
		}
	}

	tls := map[string]any{"enabled": true}
	if sni := query.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if query.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	outbound["tls"] = tls

	return node.Node{
		Name:     name,
		Type:     "hysteria",
		Outbound: outbound,
	}, nil
}

// Hysteria2Adapter handles Hysteria2 protocol
type Hysteria2Adapter struct{}

func init() {
	Register("hysteria2", &Hysteria2Adapter{})
	Register("hy2", &Hysteria2Adapter{})
}

func (a *Hysteria2Adapter) FromClash(m map[string]any) (node.Node, error) {
	server := ReadString(m, "server")
	port := ReadInt(m, "port")
	if server == "" || port == 0 {
		return node.Node{}, fmt.Errorf("missing server or port")
	}

	password := ReadString(m, "password")
	outbound := map[string]any{
		"type":        "hysteria2",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	if upMbps := ReadInt(m, "up", "up-mbps"); upMbps > 0 {
		outbound["up_mbps"] = upMbps
	}
	if downMbps := ReadInt(m, "down", "down-mbps"); downMbps > 0 {
		outbound["down_mbps"] = downMbps
	}

	ApplyTLSOptions(outbound, m)

	return node.Node{
		Type:     "hysteria2",
		Outbound: outbound,
	}, nil
}

func (a *Hysteria2Adapter) FromURI(uriStr string) (node.Node, error) {
	u, err := url.Parse("hysteria2://" + uriStr)
	if err != nil {
		return node.Node{}, err
	}

	password := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if server == "" || port == "" {
		return node.Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := ParseInt(port)
	outbound := map[string]any{
		"type":        "hysteria2",
		"server":      server,
		"server_port": portNum,
		"password":    password,
	}

	tls := map[string]any{"enabled": true}
	if sni := query.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if query.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	outbound["tls"] = tls

	return node.Node{
		Name:     name,
		Type:     "hysteria2",
		Outbound: outbound,
	}, nil
}
