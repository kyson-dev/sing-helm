package paths

import (
	"os"
	"path/filepath"
	"sync"
)

// Paths 定义了应用所有的关键路径
type Paths struct {
	HomeDir         string // 主目录
	RuntimeDir      string // 运行时目录 (socket/lock/log/state)
	RuntimeMetaFile string // runtime.json
	ConfigFile      string // profile.json (用户配置)
	RawConfigFile   string // raw.json (生成的完整配置)
	SubConfigDir    string // subscriptions 目录
	SubCacheDir     string // subscriptions cache 目录
	LogDir          string // log 目录
	LogFile         string // sing-helm.log
	StateFile       string // state.json
	LockFile        string // sing-helm.lock
	SocketFile      string // 仅 Linux 用，或存放 API 地址的文件
	AssetDir        string // 存放 geoip.db/geosite.db
	CacheFile       string // cache.db (sing-box 缓存)
}

// RuntimeMeta holds system status, such as the config path used by the running daemon.
type RuntimeMeta struct {
	ConfigHome string `json:"config_home"`
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
// 为了保持兼容性，我们可以让 Init("") 依旧使用默认 ~/.sing-helm，
// 但真正的智能选择逻辑交给 setup.go
func path_init(home string) error {
	var err error
	once.Do(func() {
		current, err = resolve(home)
	})
	return err
}

// Resolve computes Paths from the given home directory without touching global state.
// This is the preferred way to obtain Paths in DI-based code.
func resolve(home string) (Paths, error) {
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".sing-helm")
	}

	absHome, err := filepath.Abs(home)
	if err != nil {
		return Paths{}, err
	}

	if err := os.MkdirAll(absHome, 0755); err != nil {
		return Paths{}, err
	}

	runtimeDir := resolveRuntimeDir()
	runtimeDir, err = filepath.Abs(runtimeDir)
	if err != nil {
		return Paths{}, err
	}

	logDir := resolveLogDir(runtimeDir)
	return getPath(absHome, runtimeDir, logDir), nil
}

// GetPath 根据主目录生成路径配置 (纯函数)
func getPath(home string, runtimeDir string, logDir string) Paths {
	logFile := ""
	if logDir != "" {
		logFile = filepath.Join(logDir, "sing-helm.log")
	}
	return Paths{
		HomeDir:         home,
		RuntimeDir:      runtimeDir,
		RuntimeMetaFile: filepath.Join(runtimeDir, "runtime.json"),
		ConfigFile:      filepath.Join(home, "profile.json"),
		RawConfigFile:   filepath.Join(runtimeDir, "raw.json"),
		SubConfigDir:    filepath.Join(home, "subscriptions"),
		SubCacheDir:     filepath.Join(home, "subscriptions", "cache"),
		LogDir:          logDir,
		LogFile:         logFile,
		StateFile:       filepath.Join(runtimeDir, "state.json"),
		LockFile:        filepath.Join(runtimeDir, "sing-helm.lock"),
		SocketFile:      filepath.Join(runtimeDir, "ipc.sock"),
		AssetDir:        filepath.Join(runtimeDir, "assets"),
		CacheFile:       filepath.Join(runtimeDir, "cache.db"),
	}
}

// ResetForTest 重置环境单例状态
// ⚠️ 仅供测试使用，生产代码禁止调用
func ResetForTest() {
	current = Paths{}
	once = sync.Once{}
	ForTestResetRuntimeDir()
}

func ForTestInit(home string) error {
	return path_init(home)
}
