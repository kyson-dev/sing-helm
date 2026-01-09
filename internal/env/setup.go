package env

import (
	"os"
	"path/filepath"
)

// Setup 初始化环境，是应用启动的唯一环境入口
// homeFlag: 命令行传入的 --home 参数
// 逻辑：
// 1. 指定了 homeFlag -> 用之
// 2. 未指定 -> 优先级：系统 daemon 关联的配置 > 活跃实例 > 第一个注册目录 > 默认 ~/.sing-helm
// 3. 无论如何 -> 注册该环境
func Setup(homeFlag string) error {
	resolvedHome := ""

	// 1. 如果指定了 homeFlag，直接使用 (强制模式)
	if homeFlag != "" {
		resolvedHome = homeFlag
	} else {
		// 2. 自动探测：优先系统 daemon 关联的配置
		if runtimeHome := FindRuntimeConfigHome(); runtimeHome != "" {
			resolvedHome = runtimeHome
		} else {
			// 使用默认值
			userHome, _ := os.UserHomeDir()
			resolvedHome = filepath.Join(userHome, ".sing-helm")
		}
	}

	// 3. 初始化全局路径配置 (创建目录等)
	// Init 会确保目录存在并设置全局 current 变量
	if err := Init(resolvedHome); err != nil {
		return err
	}

	return nil
}
