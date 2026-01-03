package config

import (
	"os"
	"strconv"
)

// getPortOverride 返回测试期间指定的端口。
// 如果对应环境变量有效（正整数），返回该端口并标记为存在。
func getPortOverride(envKey string) (int, bool) {
	value := os.Getenv(envKey)
	if value == "" {
		return 0, false
	}

	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 {
		return 0, false
	}
	return port, true
}
