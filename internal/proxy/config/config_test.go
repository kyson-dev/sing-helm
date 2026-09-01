package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestSaveToFileUsesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	path := filepath.Join(t.TempDir(), "raw.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("write loose raw config: %v", err)
	}
	if err := SaveToFile(path, &option.Options{}); err != nil {
		t.Fatalf("save raw config: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat raw config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("raw config mode = %04o, want 0600", got)
	}
}
