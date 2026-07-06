package export

import (
	"encoding/json"
	"fmt"
	"strings"
)

// applyCompatForV1114 applies explicit compatibility transforms for sing-box 1.11.4.
func applyCompatForV1114(root map[string]any) {
	downgradeFakeIPServer(root)
	downgradeDNSServers(root)
	downgradeDNSDetour(root)
	downgradeRuleSets(root)
	downgradeSelectorOutbounds(root)
	downgradeDefaultDomainResolver(root)
	downgradeOutboundTLS(root)
}

// downgradeFakeIPServer extracts the v1.12+ type:"fakeip" DNS server's inline
// inet4_range/inet6_range into v1.11.x's top-level dns.fakeip block. Must run
// before downgradeDNSServers, which converts the server itself into the legacy
// "address":"fakeip" form and would otherwise leave inet4_range/inet6_range
// dangling on a server object where v1.11.x doesn't expect them.
func downgradeFakeIPServer(root map[string]any) {
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
		if server["type"] != "fakeip" {
			continue
		}
		fakeip := map[string]any{"enabled": true}
		if inet4Range, hasInet4 := server["inet4_range"]; hasInet4 {
			fakeip["inet4_range"] = inet4Range
			delete(server, "inet4_range")
		}
		if inet6Range, hasInet6 := server["inet6_range"]; hasInet6 {
			fakeip["inet6_range"] = inet6Range
			delete(server, "inet6_range")
		}
		dns["fakeip"] = fakeip
	}
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

// downgradeDNSDetour ensures local_dns has explicit "detour": "direct" in v1.11.x.
// In sing-box 1.11.4 a DNS server without detour falls back to the default route;
// local_dns (AliDNS) should always go direct, never through the proxy.
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
		if server["tag"] == "local_dns" {
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

// downgradeDefaultDomainResolver strips route.default_domain_resolver, a
// v1.12+ field that v1.11.x rejects under DisallowUnknownFields.
func downgradeDefaultDomainResolver(root map[string]any) {
	route, ok := root["route"].(map[string]any)
	if !ok {
		return
	}
	delete(route, "default_domain_resolver")
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

// downgradeOutboundTLS strips certificate_public_key_sha256 (added in sing-box
// v1.13.0) from outbound tls blocks for v1.11.4, which rejects it under
// DisallowUnknownFields. Since pinning can't be enforced, fall back to
// insecure so the connection still succeeds, matching the same fallback used
// when the adapter can't derive a pin in the first place.
func downgradeOutboundTLS(root map[string]any) {
	outbounds, ok := root["outbounds"].([]any)
	if !ok {
		return
	}

	for _, entry := range outbounds {
		outbound, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		tls, ok := outbound["tls"].(map[string]any)
		if !ok {
			continue
		}
		if _, hasPin := tls["certificate_public_key_sha256"]; !hasPin {
			continue
		}
		delete(tls, "certificate_public_key_sha256")
		tls["insecure"] = true
	}
}

// applyPlatformCompat applies platform-specific compatibility transforms
func applyPlatformCompat(root map[string]any, platform string) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "ios":
		// Avoid embedding desktop cache paths and local API listeners in mobile exports.
		delete(root, "experimental")
		// 中国移动网络（5G/LTE）广泛使用 IPv6（DS-Lite、NAT64），
		// ipv4_only 会导致 IPv6-only 服务解析失败，在部分蜂窝网络下严重影响连通性。
		// iOS 使用 prefer_ipv4：有 IPv4 时优先返回，否则回退 IPv6。
		if dns, ok := root["dns"].(map[string]any); ok {
			dns["strategy"] = "prefer_ipv4"
		}
		// route.go 里无条件的 ip_version:6 reject 是为桌面双栈网络设计的
		// （快速 RST 逼迫客户端回退 IPv4）。IPv6-only 蜂窝网络（NAT64/DS-Lite）
		// 没有 IPv4 可回退，保留这条规则会和上面的 prefer_ipv4 直接冲突，
		// 导致纯 IPv6 网络下完全无法连接，因此 iOS 导出去掉它。
		removeIPv6RejectRoute(root)
	}
}

// removeIPv6RejectRoute strips the bare {"ip_version":6,"action":"reject"} rule
// injected by route.go. Only matches the exact bare form (no other matchers) so
// a user-authored rule that happens to combine ip_version:6 with other criteria
// is left untouched.
func removeIPv6RejectRoute(root map[string]any) {
	route, ok := root["route"].(map[string]any)
	if !ok {
		return
	}
	rules, ok := route["rules"].([]any)
	if !ok {
		return
	}
	filtered := make([]any, 0, len(rules))
	for _, entry := range rules {
		rule, ok := entry.(map[string]any)
		if ok && len(rule) == 2 && intFromAny(rule["ip_version"]) == 6 && rule["action"] == "reject" {
			continue
		}
		filtered = append(filtered, entry)
	}
	route["rules"] = filtered
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
