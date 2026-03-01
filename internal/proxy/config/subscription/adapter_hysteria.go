package subscription

import (
	"fmt"
	"net/url"
)

// HysteriaAdapter handles Hysteria protocol
type HysteriaAdapter struct{}

func init() {
	RegisterAdapter("hysteria", &HysteriaAdapter{})
}

func (a *HysteriaAdapter) FromClash(m map[string]any) (Node, error) {
	server := readString(m, "server")
	port := readInt(m, "port")
	if server == "" || port == 0 {
		return Node{}, fmt.Errorf("missing server or port")
	}

	auth := readString(m, "auth_str", "auth-str", "auth")
	protocol := readString(m, "protocol")
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
	if upMbps := readInt(m, "up", "up-mbps"); upMbps > 0 {
		outbound["up_mbps"] = upMbps
	}
	if downMbps := readInt(m, "down", "down-mbps"); downMbps > 0 {
		outbound["down_mbps"] = downMbps
	}

	ApplyTLSOptions(outbound, m)

	return Node{
		Type:     "hysteria",
		Outbound: outbound,
	}, nil
}

func (a *HysteriaAdapter) FromURI(uriStr string) (Node, error) {
	u, err := url.Parse("hysteria://" + uriStr)
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

	tls := map[string]any{"enabled": true}
	if sni := query.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if query.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	outbound["tls"] = tls

	return Node{
		Name:     name,
		Type:     "hysteria",
		Outbound: outbound,
	}, nil
}

// Hysteria2Adapter handles Hysteria2 protocol
type Hysteria2Adapter struct{}

func init() {
	RegisterAdapter("hysteria2", &Hysteria2Adapter{})
	RegisterAdapter("hy2", &Hysteria2Adapter{})
}

func (a *Hysteria2Adapter) FromClash(m map[string]any) (Node, error) {
	server := readString(m, "server")
	port := readInt(m, "port")
	if server == "" || port == 0 {
		return Node{}, fmt.Errorf("missing server or port")
	}

	password := readString(m, "password")
	outbound := map[string]any{
		"type":        "hysteria2",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	if upMbps := readInt(m, "up", "up-mbps"); upMbps > 0 {
		outbound["up_mbps"] = upMbps
	}
	if downMbps := readInt(m, "down", "down-mbps"); downMbps > 0 {
		outbound["down_mbps"] = downMbps
	}

	ApplyTLSOptions(outbound, m)

	return Node{
		Type:     "hysteria2",
		Outbound: outbound,
	}, nil
}

func (a *Hysteria2Adapter) FromURI(uriStr string) (Node, error) {
	u, err := url.Parse("hysteria2://" + uriStr)
	if err != nil {
		return Node{}, err
	}

	password := u.User.Username()
	server := u.Hostname()
	port := u.Port()
	name := u.Fragment
	query := u.Query()

	if server == "" || port == "" {
		return Node{}, fmt.Errorf("missing required fields")
	}

	portNum, _ := parseInt(port)
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

	return Node{
		Name:     name,
		Type:     "hysteria2",
		Outbound: outbound,
	}, nil
}
