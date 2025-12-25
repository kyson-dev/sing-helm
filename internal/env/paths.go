package env

import (
	"os"
	"path/filepath"
	"sync"
)

// Paths 定义了应用所有的关键路径
type Paths struct {
	HomeDir       string // 主目录
	ConfigFile    string // profile.json (用户配置)
	RawConfigFile string // raw.json (生成的完整配置)
	LogFile       string // minibox.log
	StateFile     string // state.json
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

var (
	// 这个变量是给 ldflags 注入用的
	// 默认为空，如果有注入，它就会变成 "/var/lib/minibox" 之类的值
	DefaultHome string
)

// Init 初始化环境
// flagHome: 命令行传入的 --home 参数，为空则自动探测
func Init(flagHome string) error {
	var err error
	once.Do(func() {
		// 1. 确定主目录 (HomeDir)
		home := ""

		if flagHome != "" {
			// 1. 最高优先级：命令行 Flag (--home)
			home = flagHome
		} else if envHome := os.Getenv("MINIBOX_HOME"); envHome != "" {
			// 2. 次高优先级：环境变量 (用户在 .zshrc 里配的)
			home = envHome
		} else if DefaultHome != "" {
			// 3. 第三优先级：构建时注入的默认值 (ldflags)
			// 适用于发行版打包，比如 rpm/deb 包希望默认在 /var/lib/minibox
			home = DefaultHome
		} else {
			// 4. 最低优先级：代码里的硬编码兜底
			userHome, _ := os.UserHomeDir()
			home = filepath.Join(userHome, ".minibox")
		}

		// 转换成绝对路径，避免后续逻辑混乱
		home, err = filepath.Abs(home)
		if err != nil {
			return
		}

		// 2. 确保主目录存在
		if err = os.MkdirAll(home, 0755); err != nil {
			return
		}

		// 3. 定义子路径
		current = Paths{
			HomeDir:       home,
			ConfigFile:    filepath.Join(home, "profile.json"),
			RawConfigFile: filepath.Join(home, "raw.json"),
			LogFile:       filepath.Join(home, "minibox.log"),
			StateFile:     filepath.Join(home, "state.json"),
			SocketFile:    filepath.Join(home, "ipc.sock"),
			AssetDir:      filepath.Join(home, "assets"),
			CacheFile:     filepath.Join(home, "cache.db"),
		}
	})
	return err
}

// ResetForTest 重置环境单例状态
// ⚠️ 仅供测试使用，生产代码禁止调用
func ResetForTest() {
	current = Paths{}
	once = sync.Once{}
}
