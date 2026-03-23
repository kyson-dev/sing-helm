package adapter

import (
	"fmt"
	"strings"
)

func ReadString(m map[string]any, keys ...string) string {
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

func ReadInt(m map[string]any, keys ...string) int {
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
				if parsed, err := ParseInt(v); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func ReadBool(m map[string]any, key string) bool {
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

func ReadStringList(m map[string]any, key string) []string {
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

func AsStringMap(val any) map[string]any {
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

func NormalizeStringMap(input map[string]any) map[string]string {
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

func ParseInt(value string) (int, error) {
	var out int
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &out)
	return out, err
}

func ApplyTLSOptions(outbound map[string]any, m map[string]any) {
	tlsEnabled := ReadBool(m, "tls")
	sni := ReadString(m, "sni", "servername", "server_name")
	skipVerify := ReadBool(m, "skip-cert-verify")
	alpn := ReadStringList(m, "alpn")
	clientFingerprint := ReadString(m, "client-fingerprint")

	realityOpts := AsStringMap(m["reality-opts"])

	if !tlsEnabled && sni == "" && len(alpn) == 0 && realityOpts == nil && clientFingerprint == "" {
		return
	}

	tls := map[string]any{"enabled": true}
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

	if realityOpts != nil {
		reality := map[string]any{"enabled": true}
		if publicKey := ReadString(realityOpts, "public-key", "public_key"); publicKey != "" {
			reality["public_key"] = publicKey
		}
		if shortID := ReadString(realityOpts, "short-id", "short_id"); shortID != "" {
			reality["short_id"] = shortID
		}
		tls["reality"] = reality
	}

	outbound["tls"] = tls
}

func ApplyTransportOptions(outbound map[string]any, m map[string]any) {
	network := strings.ToLower(ReadString(m, "network"))
	switch network {
	case "ws", "websocket":
		wsOpts := AsStringMap(m["ws-opts"])
		transport := map[string]any{"type": "ws"}
		if wsOpts != nil {
			if path := ReadString(wsOpts, "path"); path != "" {
				transport["path"] = path
			}
			headers := AsStringMap(wsOpts["headers"])
			if len(headers) > 0 {
				transport["headers"] = NormalizeStringMap(headers)
			}
		}
		outbound["transport"] = transport
	case "grpc":
		grpcOpts := AsStringMap(m["grpc-opts"])
		transport := map[string]any{"type": "grpc"}
		if grpcOpts != nil {
			if service := ReadString(grpcOpts, "grpc-service-name", "service-name"); service != "" {
				transport["service_name"] = service
			}
		}
		outbound["transport"] = transport
	}
}

func ApplyURITransport(outbound map[string]any, network string, query map[string][]string) {
	getQuery := func(k string) string {
		if v := query[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}

	transport := map[string]any{"type": network}
	switch network {
	case "ws":
		if path := getQuery("path"); path != "" {
			transport["path"] = path
		}
		if host := getQuery("host"); host != "" && host != outbound["server"] {
			transport["headers"] = map[string]string{"Host": host}
		}
	case "grpc":
		if serviceName := getQuery("serviceName"); serviceName != "" {
			transport["service_name"] = serviceName
		}
	}
	outbound["transport"] = transport
}
