package env

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

type DaemonLock struct {
	file *os.File
	path string
}

func GetLockPath(homeDir string) string {
	return filepath.Join(homeDir, "minibox.lock")
}

// AcquireLock 获取指定运行时目录的文件锁，非阻塞
// 如果已经被锁定，返回 error
func AcquireLock(runtimeDir string) (*DaemonLock, error) {
	path := GetLockPath(runtimeDir)
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// 尝试获取互斥锁，非阻塞
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("daemon already running")
	}

	return &DaemonLock{
		file: f,
		path: path,
	}, nil
}

// CheckLock 检查指定运行时目录的锁是否被占用
// 如果锁被占用，返回 nil (daemon running)
// 如果锁未被占用，返回 error (daemon not running)
func CheckLock(runtimeDir string) error {
	path := GetLockPath(runtimeDir)
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return errors.New("daemon not running (lock file missing)")
	}
	readOnly := false
	if err != nil {
		if os.IsPermission(err) {
			f, err = os.Open(path)
			if err != nil {
				return err
			}
			readOnly = true
		} else {
			return err
		}
	}
	defer f.Close()

	// 尝试获取锁
	lockType := syscall.LOCK_EX
	if readOnly {
		lockType = syscall.LOCK_SH
	}
	if err := syscall.Flock(int(f.Fd()), lockType|syscall.LOCK_NB); err != nil {
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
		// 可选：删除锁文件 (但为了 CheckLock 的 IsNotExist 判断，保留文件也可以，这里选择保留)
		// os.Remove(l.path)
	}
	return nil
}
