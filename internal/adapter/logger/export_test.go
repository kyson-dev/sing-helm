package logger

import "sync"

// ResetForTest 重置 logger 实例和 once，仅供测试使用
func ResetForTest() {
	instance = nil
	once = sync.Once{}
}
