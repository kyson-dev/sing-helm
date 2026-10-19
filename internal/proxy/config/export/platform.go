package export

import "strings"

// applyPlatformCompat removes desktop-only services and desktop-only paths
// while preserving the portable cache settings required for stable fake-IP
// mappings.
func applyPlatformCompat(root map[string]any, platform string) {
	if strings.ToLower(strings.TrimSpace(platform)) != "ios" {
		return
	}

	stripTailscaleStateDirectory(root)

	experimental, ok := root["experimental"].(map[string]any)
	if !ok {
		return
	}

	delete(experimental, "clash_api")
	if cacheFile, ok := experimental["cache_file"].(map[string]any); ok {
		// Desktop paths are not valid inside the iOS application sandbox. An
		// empty path makes sing-box use its platform-local cache.db.
		delete(cacheFile, "path")
	}
	if len(experimental) == 0 {
		delete(root, "experimental")
	}
}

// stripTailscaleStateDirectory drops state_directory from Tailscale endpoints.
// The desktop value is an absolute path that is not valid inside the iOS
// sandbox; without it sing-box uses its platform-local default.
func stripTailscaleStateDirectory(root map[string]any) {
	endpoints, _ := root["endpoints"].([]any)
	for _, e := range endpoints {
		if endpoint, ok := e.(map[string]any); ok && endpoint["type"] == "tailscale" {
			delete(endpoint, "state_directory")
		}
	}
}
