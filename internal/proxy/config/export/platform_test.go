package export

import "testing"

func TestApplyPlatformCompatIOS_PreservesFakeIPCache(t *testing.T) {
	root := map[string]any{
		"experimental": map[string]any{
			"clash_api": map[string]any{"external_controller": "127.0.0.1:9090"},
			"cache_file": map[string]any{
				"enabled":      true,
				"path":         "/desktop/cache.db",
				"store_fakeip": true,
			},
		},
	}

	applyPlatformCompat(root, "ios")

	experimental := root["experimental"].(map[string]any)
	if _, ok := experimental["clash_api"]; ok {
		t.Fatal("iOS export must not contain clash_api")
	}
	cacheFile := experimental["cache_file"].(map[string]any)
	if cacheFile["enabled"] != true || cacheFile["store_fakeip"] != true {
		t.Fatalf("fake-IP cache settings were not preserved: %#v", cacheFile)
	}
	if _, ok := cacheFile["path"]; ok {
		t.Fatalf("iOS export must not contain a desktop cache path: %#v", cacheFile)
	}
}

func TestApplyPlatformCompatNonIOS_Unchanged(t *testing.T) {
	root := map[string]any{
		"experimental": map[string]any{
			"clash_api": map[string]any{"external_controller": "127.0.0.1:9090"},
		},
	}

	applyPlatformCompat(root, "macos")

	experimental := root["experimental"].(map[string]any)
	if _, ok := experimental["clash_api"]; !ok {
		t.Fatal("non-iOS export must remain unchanged")
	}
}
