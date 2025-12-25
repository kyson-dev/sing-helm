package config

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kyson/minibox/internal/env"
)

type DaemonLock struct {
	file *os.File
}

func GetLockPath() string {
	return filepath.Join(filepath.Dir(env.Get().StateFile), "minibox.lock")
}

// AcquireLock 获取文件锁，非阻塞
// 如果已经被锁定，返回 error
func AcquireLock() (*DaemonLock, error) {
	path := GetLockPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// 尝试获取互斥锁，非阻塞
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("daemon already running")
	}

	return &DaemonLock{file: f}, nil
}

// CheckLock 检查锁是否被占用
// 如果锁被占用，返回 nil (daemon running)
// 如果锁未被占用，返回 error (daemon not running)
func CheckLock() error {
	path := GetLockPath()
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return errors.New("daemon not running (lock file missing)")
	}
	if err != nil {
		return err
	}
	defer f.Close()

	// 尝试获取锁
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// 获取锁失败，说明正在运行
		return nil
	}

	// 获取锁成功，说明没在运行，立即释放
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return errors.New("daemon not running")
}

// Release 释放锁
func (l *DaemonLock) Release() error {
	if l.file != nil {
		syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		l.file.Close()
		l.file = nil
		// 可选：删除锁文件
		os.Remove(GetLockPath())
	}
	return nil
}
