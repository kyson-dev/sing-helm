package netutil

import (
	"net"
)

// GetFreePort 请求内核分配一个空闲端口
func GetFreePort() (int, error) {
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