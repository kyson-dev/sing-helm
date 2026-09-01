package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestServeCommandExportsLatestOnly(t *testing.T) {
	cmd := newServeCommand()
	if flag := cmd.Flags().Lookup("target-version"); flag != nil {
		t.Fatalf("deprecated target-version flag is still registered: %v", flag)
	}
	if got, want := defaultExportPath("ios"), "bin/singbox-config-latest-ios.json"; got != want {
		t.Fatalf("default export path = %q, want %q", got, want)
	}
}

func TestSaveExportUsesPrivateFileWithoutChangingParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	parent := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatalf("create export parent: %v", err)
	}
	output := filepath.Join(parent, "config.json")
	if err := saveExport(output, []byte(`{"secret":true}`)); err != nil {
		t.Fatalf("save export: %v", err)
	}

	assertPathMode(t, output, 0600)
	assertPathMode(t, parent, 0755)
}

func assertPathMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %04o, want %04o", path, got, want)
	}
}
