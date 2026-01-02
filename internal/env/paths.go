package env

import (
	"os"
	"path/filepath"
	"sync"
)

// Paths 定义了应用所有的关键路径
type Paths struct {
	HomeDir       string // 主目录
	RuntimeDir    string // 运行时目录 (socket/lock/log/state)
	ConfigFile    string // profile.json (用户配置)
	RawConfigFile string // raw.json (生成的完整配置)
	LogFile       string // minibox.log
	StateFile     string // state.json
	LookFile      string // minibox.lock
	SocketFile    string // 仅 Linux 用，或存放 API 地址的文件
	AssetDir      string // 存放 geoip.db/geosite.db
	CacheFile     string // cache.db (sing-box 缓存)
}

var (
	current Paths
	once    sync.Once
)

// Get 获取全局路径配置
func Get() Paths {
	return current
}

// Init 初始化环境
// home: 必须是已解析的绝对路径或相对路径，如果为空则报错（或者使用默认？）
// 为了保持兼容性，我们可以让 Init("") 依旧使用默认 ~/.minibox，
// 但真正的智能选择逻辑交给 setup.go
func Init(home string) error {
	var err error
	once.Do(func() {
		if home == "" {
			// 兜底默认值
			userHome, _ := os.UserHomeDir()
			home = filepath.Join(userHome, ".minibox")
		}

		// 转换成绝对路径
		home, err = filepath.Abs(home)
		if err != nil {
			return
		}

		// 确保主目录存在
		if err = os.MkdirAll(home, 0755); err != nil {
			return
		}

		runtimeDir := ResolveRuntimeDir()
		runtimeDir, err = filepath.Abs(runtimeDir)
		if err != nil {
			return
		}

		logDir := ResolveLogDir(runtimeDir)
		current = GetPath(home, runtimeDir, logDir)
	})
	return err
}

// GetPath 根据主目录生成路径配置 (纯函数)
func GetPath(home string, runtimeDir string, logDir string) Paths {
	logFile := ""
	if logDir != "" {
		logFile = filepath.Join(logDir, "minibox.log")
	}
	return Paths{
		HomeDir:       home,
		RuntimeDir:    runtimeDir,
		ConfigFile:    filepath.Join(home, "profile.json"),
		RawConfigFile: filepath.Join(runtimeDir, "raw.json"),
		LogFile:       logFile,
		StateFile:     filepath.Join(runtimeDir, "state.json"),
		LookFile:      GetLockPath(runtimeDir), // 使用 lock.go 中的单一事实来源
		SocketFile:    filepath.Join(runtimeDir, "ipc.sock"),
		AssetDir:      filepath.Join(runtimeDir, "assets"),
		CacheFile:     filepath.Join(runtimeDir, "cache.db"),
	}
}

// ResetForTest 重置环境单例状态
// ⚠️ 仅供测试使用，生产代码禁止调用
func ResetForTest() {
	current = Paths{}
	once = sync.Once{}
	ResetRuntimeDir()
}
