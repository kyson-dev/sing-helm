package logger

import (
	"log/slog"
	"sync"
)

// ResetForTest 重置 logger 实例和 once，仅供测试使用
func ResetForTest() {
	instance = nil
	once = sync.Once{}
}

// Get 暴露内部 get() 函数，仅供测试使用
func Get() *slog.Logger {
	return get()
}
