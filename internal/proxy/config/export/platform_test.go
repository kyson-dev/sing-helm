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

// 没有 experimental 字段时，函数不能提前返回而漏掉 endpoints。
func TestApplyPlatformCompatIOS_StripsTailscaleStateDirectory(t *testing.T) {
	root := map[string]any{
		"endpoints": []any{
			map[string]any{
				"type":            "tailscale",
				"tag":             "ts",
				"state_directory": "/usr/local/var/run/sing-helm/tailscale",
				"hostname":        "keep-me",
			},
			map[string]any{
				"type":            "wireguard",
				"tag":             "wg",
				"state_directory": "/untouched",
			},
		},
	}

	applyPlatformCompat(root, "ios")

	endpoints := root["endpoints"].([]any)
	ts := endpoints[0].(map[string]any)
	if _, ok := ts["state_directory"]; ok {
		t.Fatalf("iOS export must not contain a desktop tailscale state_directory: %#v", ts)
	}
	if ts["type"] != "tailscale" || ts["tag"] != "ts" || ts["hostname"] != "keep-me" {
		t.Fatalf("other tailscale fields must be preserved: %#v", ts)
	}
	other := endpoints[1].(map[string]any)
	if other["state_directory"] != "/untouched" {
		t.Fatalf("non-tailscale endpoints must not be modified: %#v", other)
	}
}

func TestApplyPlatformCompatNonIOS_KeepsTailscaleStateDirectory(t *testing.T) {
	root := map[string]any{
		"endpoints": []any{
			map[string]any{"type": "tailscale", "tag": "ts", "state_directory": "/desktop/ts"},
		},
	}

	applyPlatformCompat(root, "macos")

	ts := root["endpoints"].([]any)[0].(map[string]any)
	if ts["state_directory"] != "/desktop/ts" {
		t.Fatalf("non-iOS export must keep state_directory: %#v", ts)
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
