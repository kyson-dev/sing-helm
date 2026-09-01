package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenInEditorRehardensFileAfterEditorFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX editor script and permission bits")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	editor := filepath.Join(dir, "editor")
	script := []byte("#!/bin/sh\nchmod 0644 \"$1\"\nexit 1\n")
	if err := os.WriteFile(editor, script, 0700); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("EDITOR", editor)

	if err := openInEditor(nil, path); err == nil {
		t.Fatal("expected editor failure")
	}
	assertPathMode(t, path, 0600)
}
