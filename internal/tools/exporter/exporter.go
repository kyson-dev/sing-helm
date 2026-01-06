package exporter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// Target controls compatibility transforms for exported configs.
type Target struct {
	Version  string
	Platform string
}

// Export serializes options and applies compatibility transforms when needed.
func Export(opts *option.Options, target Target) ([]byte, error) {
	data, err := singboxjson.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// No transforms needed if no target specified
	if strings.TrimSpace(target.Version) == "" && strings.TrimSpace(target.Platform) == "" {
		return data, nil
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply version-specific compatibility transforms
	if strings.TrimSpace(target.Version) != "" {
		if err := applyVersionCompat(root, target.Version); err != nil {
			return nil, err
		}
	}

	// Apply platform-specific compatibility transforms
	if strings.TrimSpace(target.Platform) != "" {
		applyPlatformCompat(root, target.Platform)
	}

	return json.Marshal(root)
}

// applyVersionCompat applies version-specific compatibility transforms
func applyVersionCompat(root map[string]any, version string) error {
	less, err := versionLess(version, "1.12.0")
	if err != nil {
		return err
	}

	if less {
		// v1.11.x compatibility transforms
		downgradeDNSServers(root)
		downgradeDNSDetour(root) // Add detour: direct for DNS servers
		downgradeRuleSets(root)
		downgradeTunInbounds(root)
		downgradeSelectorOutbounds(root)
	}

	return nil
}

// downgradeDNSServers converts v1.12+ DNS server format to v1.11.x format
func downgradeDNSServers(root map[string]any) {
	dns, ok := root["dns"].(map[string]any)
	if !ok {
		return
	}
	servers, ok := dns["servers"].([]any)
	if !ok {
		return
	}

	for _, entry := range servers {
		server, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		// Skip if already has legacy 'address' field
		if _, ok := server["address"]; ok {
			continue
		}

		typ, _ := server["type"].(string)
		host, _ := server["server"].(string)
		port := intFromAny(server["server_port"])
		path, _ := server["path"].(string)

		// Build legacy address
		address := buildLegacyDNSAddress(typ, host, port, path)
		if address != "" {
			server["address"] = address
		}

		// Rename domain_* to address_*
		if resolver, ok := server["domain_resolver"]; ok {
			server["address_resolver"] = resolver
		}
		if strategy, ok := server["domain_strategy"]; ok {
			server["address_strategy"] = strategy
		}

		// Remove v1.12+ fields
		delete(server, "type")
		delete(server, "server")
		delete(server, "server_port")
		delete(server, "path")
		delete(server, "headers")
		delete(server, "tls")
		delete(server, "domain_resolver")
		delete(server, "domain_strategy")
	}
}

// downgradeDNSDetour adds "detour": "direct" to DNS servers for v1.11.x
// This is critical for proper DNS resolution on iOS v1.11.4
func downgradeDNSDetour(root map[string]any) {
	dns, ok := root["dns"].(map[string]any)
	if !ok {
		return
	}
	servers, ok := dns["servers"].([]any)
	if !ok {
		return
	}

	for _, entry := range servers {
		server, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		tag, _ := server["tag"].(string)

		// Add detour: direct for local_dns and resolver_dns
		// These need direct connection to avoid DNS resolution loops
		if tag == "local_dns" || tag == "resolver_dns" {
			// Only add if not already present
			if _, hasDetour := server["detour"]; !hasDetour {
				server["detour"] = "direct"
			}
		}
	}
}

// downgradeRuleSets ensures rule_set format field is present for v1.11.x
func downgradeRuleSets(root map[string]any) {
	route, ok := root["route"].(map[string]any)
	if !ok {
		return
	}
	ruleSets, ok := route["rule_set"].([]any)
	if !ok {
		return
	}

	for _, entry := range ruleSets {
		ruleSet, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		// For remote rule sets in v1.11.4, format field is required
		typ, _ := ruleSet["type"].(string)
		if typ == "remote" {
			// Check if format already exists
			if _, hasFormat := ruleSet["format"]; !hasFormat {
				// .srs files are binary format
				url, _ := ruleSet["url"].(string)
				if strings.HasSuffix(url, ".srs") {
					ruleSet["format"] = "binary"
				} else {
					// Default to source format for .json files
					ruleSet["format"] = "source"
				}
			}
		}
	}
}

// downgradeTunInbounds converts tun inbound address field for v1.11.x
func downgradeTunInbounds(root map[string]any) {
	inbounds, ok := root["inbounds"].([]any)
	if !ok {
		return
	}

	for _, entry := range inbounds {
		inbound, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		typ, _ := inbound["type"].(string)
		if typ == "tun" {
			// Convert address to inet4_address for v1.11.4
			if address, ok := inbound["address"].(string); ok {
				inbound["inet4_address"] = address
				delete(inbound, "address")
			}
		}
	}
}

// downgradeSelectorOutbounds removes v1.12+ fields from selector/urltest outbounds
func downgradeSelectorOutbounds(root map[string]any) {
	outbounds, ok := root["outbounds"].([]any)
	if !ok {
		return
	}

	for _, entry := range outbounds {
		outbound, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		typ, _ := outbound["type"].(string)
		// Remove 'default' field from selector/urltest (introduced in v1.11.0)
		if typ == "selector" || typ == "urltest" {
			delete(outbound, "default")
		}
	}
}

// applyPlatformCompat applies platform-specific compatibility transforms
func applyPlatformCompat(root map[string]any, platform string) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "ios":
		// Avoid embedding desktop cache paths and local API listeners in mobile exports.
		delete(root, "experimental")
	}
}

// buildLegacyDNSAddress constructs legacy DNS address string from components
func buildLegacyDNSAddress(typ, host string, port int, path string) string {
	if host == "" {
		switch typ {
		case "local":
			return "local"
		case "fakeip":
			return "fakeip"
		case "dhcp":
			return "dhcp://auto"
		}
		return ""
	}

	switch typ {
	case "udp":
		return legacyAddressWithScheme("udp", host, port)
	case "tcp":
		return legacyAddressWithScheme("tcp", host, port)
	case "tls":
		return legacyAddressWithScheme("tls", host, port)
	case "quic":
		return legacyAddressWithScheme("quic", host, port)
	case "h3":
		return legacyAddressWithScheme("h3", host, port)
	case "https":
		return legacyHTTPSAddress(host, port, path)
	case "local":
		return "local"
	case "fakeip":
		return "fakeip"
	case "dhcp":
		return "dhcp://" + host
	default:
		return host
	}
}

func legacyAddressWithScheme(scheme, host string, port int) string {
	if port > 0 && !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, port)
	}
	return scheme + "://" + host
}

func legacyHTTPSAddress(host string, port int, path string) string {
	if port > 0 && port != 443 && !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, port)
	}
	if path == "" {
		path = "/dns-query"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "https://" + host + path
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

func versionLess(a, b string) (bool, error) {
	av, err := parseVersion(a)
	if err != nil {
		return false, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return false, err
	}

	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return true, nil
		}
		if av[i] > bv[i] {
			return false, nil
		}
	}
	return false, nil
}

func parseVersion(v string) ([3]int, error) {
	var out [3]int
	trimmed := strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if trimmed == "" {
		return out, fmt.Errorf("invalid version: %q", v)
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}

	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			out[i] = 0
			continue
		}
		part := strings.TrimSpace(parts[i])
		if part == "" {
			return out, fmt.Errorf("invalid version: %q", v)
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return out, fmt.Errorf("invalid version: %q", v)
		}
		out[i] = value
	}

	return out, nil
}
