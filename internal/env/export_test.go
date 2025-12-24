package env

import "sync"

// ResetForTest 重置单例状态 (仅供测试使用)
func ResetForTest() {
	once = sync.Once{}
	current = Paths{}
}

