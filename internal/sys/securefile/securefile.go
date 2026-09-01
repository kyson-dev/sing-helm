package securefile

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	PrivateDirMode  os.FileMode = 0700
	PrivateFileMode os.FileMode = 0600
)

// EnsurePrivateDir creates path when needed and enforces owner-only access.
func EnsurePrivateDir(path string) error {
	if err := os.MkdirAll(path, PrivateDirMode); err != nil {
		return err
	}
	return hardenDir(path)
}

// HardenPrivateDir enforces owner-only access when path exists.
func HardenPrivateDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return hardenDirInfo(path, info)
}

// HardenPrivateFile enforces owner-only access when path exists.
func HardenPrivateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlinked private file: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private file is not a regular file: %s", path)
	}
	return os.Chmod(path, PrivateFileMode)
}

// EnsurePrivateFile creates an empty owner-only file when needed and hardens an
// existing regular file. It never follows a symlink at the target path.
func EnsurePrivateFile(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlinked private file: %s", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("private file is not a regular file: %s", path)
		}
		return os.Chmod(path, PrivateFileMode)
	}
	if !os.IsNotExist(err) {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, PrivateFileMode)
	if err != nil {
		return err
	}
	return f.Close()
}

// WriteFile atomically replaces path with an owner-only regular file. The
// parent directory must already exist; its permissions are intentionally left
// unchanged so callers can safely export into user-selected directories.
func WriteFile(path string, data []byte) error {
	if err := validateTarget(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sing-helm-private-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(PrivateFileMode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		closed = true
		return err
	}
	closed = true

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, PrivateFileMode)
}

func validateTarget(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlinked private file: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private file is not a regular file: %s", path)
	}
	return nil
}

func hardenDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return hardenDirInfo(path, info)
}

func hardenDirInfo(path string, info os.FileInfo) error {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || filepath.Dir(cleanPath) == cleanPath {
		return fmt.Errorf("refusing broad private directory target: %s", path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlinked private directory: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("private directory is not a directory: %s", path)
	}
	return os.Chmod(path, PrivateDirMode)
}
