package env

import (
	"os"
	"path/filepath"
)

// Setup 初始化环境，是应用启动的唯一环境入口
// homeFlag: 命令行传入的 --home 参数
// 逻辑：
// 1. 指定了 homeFlag -> 用之
// 2. 未指定 -> 优先级：活跃实例 > 第一个注册目录 > 默认 ~/.minibox
// 3. 无论如何 -> 注册该环境
func Setup(homeFlag string) error {
	resolvedHome := ""

	// 1. 如果指定了 homeFlag，直接使用 (强制模式)
	if homeFlag != "" {
		resolvedHome = homeFlag
	} else {
		// 2. 自动探测：优先活跃实例，其次第一个注册目录
		if active := FindActive(); active != "" {
			// 找到活跃实例，使用它
			resolvedHome = active
		} else {
			// 没有活跃实例，检查注册表
			list := GetList()
			if len(list) > 0 {
				// 使用第一个注册的目录（最近使用的）
				resolvedHome = list[0]
			} else {
				// 注册表为空，使用默认值
				userHome, _ := os.UserHomeDir()
				resolvedHome = filepath.Join(userHome, ".minibox")
			}
		}
	}

	// 3. 初始化全局路径配置 (创建目录等)
	// Init 会确保目录存在并设置全局 current 变量
	if err := Init(resolvedHome); err != nil {
		return err
	}

	// 4. 注册到注册表 (确保下次能发现)
	// Register 会将 resolvedHome 添加到 ~/.config/minibox/registry.json
	return Register(resolvedHome)
}
