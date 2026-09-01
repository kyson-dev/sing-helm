package subscription

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveSourceAndCacheUsePrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	root := t.TempDir()
	configDir := filepath.Join(root, "subscriptions")
	source := Source{Name: "US", URL: "https://example.com/sub"}
	if err := SaveSource(configDir, source); err != nil {
		t.Fatalf("save source: %v", err)
	}
	assertStorageMode(t, configDir, 0700)
	assertStorageMode(t, filepath.Join(configDir, "US.json"), 0600)

	cacheDir := filepath.Join(configDir, "cache")
	if err := os.Mkdir(cacheDir, 0700); err != nil {
		t.Fatalf("create cache dir: %v", err)
	}
	cachePath := filepath.Join(cacheDir, "US.json")
	if err := os.WriteFile(cachePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write loose cache: %v", err)
	}
	if err := SaveCache(cachePath, Cache{Source: source}); err != nil {
		t.Fatalf("save cache: %v", err)
	}
	assertStorageMode(t, cachePath, 0600)
}

func assertStorageMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %04o, want %04o", path, got, want)
	}
}
