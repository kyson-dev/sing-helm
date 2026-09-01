package securefile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteFileCreatesAndTightensPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := WriteFile(path, []byte("first")); err != nil {
		t.Fatalf("create private file: %v", err)
	}
	assertMode(t, path, PrivateFileMode)

	if err := os.Chmod(path, 0644); err != nil {
		t.Fatalf("loosen permissions: %v", err)
	}
	if err := WriteFile(path, []byte("second")); err != nil {
		t.Fatalf("rewrite private file: %v", err)
	}
	assertMode(t, path, PrivateFileMode)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("content = %q, want second", data)
	}
}

func TestEnsurePrivateDirTightensExistingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	path := filepath.Join(t.TempDir(), "subscriptions")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := EnsurePrivateDir(path); err != nil {
		t.Fatalf("ensure private directory: %v", err)
	}
	assertMode(t, path, PrivateDirMode)
}

func TestPrivateFileOperationsRejectSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "profile.json")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	for name, fn := range map[string]func() error{
		"harden": func() error { return HardenPrivateFile(link) },
		"ensure": func() error { return EnsurePrivateFile(link) },
		"write":  func() error { return WriteFile(link, []byte("replacement")) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := fn(); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("error = %v, want symlink rejection", err)
			}
		})
	}
}

func TestEnsurePrivateDirRejectsFilesystemRoot(t *testing.T) {
	root := string(os.PathSeparator)
	if volume := filepath.VolumeName(t.TempDir()); volume != "" {
		root = volume + string(os.PathSeparator)
	}
	if err := EnsurePrivateDir(root); err == nil || !strings.Contains(err.Error(), "broad") {
		t.Fatalf("error = %v, want broad-directory rejection", err)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %04o, want %04o", path, got, want)
	}
}
