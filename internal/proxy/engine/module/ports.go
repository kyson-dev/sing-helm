package module

import (
	"os"
	"strconv"
	"net"
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

// GetFreePort 请求内核分配一个空闲端口
func getFreePort() (int, error) {
	// 监听端口 0，内核会自动分配一个空闲端口
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	// 返回分配到的端口
	return l.Addr().(*net.TCPAddr).Port, nil
}



