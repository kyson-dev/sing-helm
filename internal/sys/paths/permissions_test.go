package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHardenConfigPermissionsMigratesKnownConfigTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	home := filepath.Join(t.TempDir(), "home")
	subscriptions := filepath.Join(home, "subscriptions")
	cache := filepath.Join(subscriptions, "cache")
	if err := os.MkdirAll(cache, 0755); err != nil {
		t.Fatalf("create config tree: %v", err)
	}
	for _, dir := range []string{home, subscriptions, cache} {
		if err := os.Chmod(dir, 0755); err != nil {
			t.Fatalf("set loose directory mode: %v", err)
		}
	}
	profile := filepath.Join(home, "profile.json")
	source := filepath.Join(subscriptions, "US.json")
	cached := filepath.Join(cache, "US.json")
	unrelated := filepath.Join(subscriptions, "notes.txt")
	for _, path := range []string{profile, source, cached, unrelated} {
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	p := getPath(home, filepath.Join(t.TempDir(), "runtime"), "")
	if err := hardenConfigPermissions(p); err != nil {
		t.Fatalf("harden config permissions: %v", err)
	}

	assertPermission(t, home, 0700)
	assertPermission(t, subscriptions, 0700)
	assertPermission(t, cache, 0700)
	assertPermission(t, profile, 0600)
	assertPermission(t, source, 0600)
	assertPermission(t, cached, 0600)
	assertPermission(t, unrelated, 0644)
}

func assertPermission(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %04o, want %04o", path, got, want)
	}
}
