package export

import "strings"

// applyPlatformCompat removes desktop-only services while preserving the
// portable cache settings required for stable fake-IP mappings.
func applyPlatformCompat(root map[string]any, platform string) {
	if strings.ToLower(strings.TrimSpace(platform)) != "ios" {
		return
	}

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
