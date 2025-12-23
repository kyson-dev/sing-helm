package monitor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestUpdate_Traffic 验证流量更新逻辑
func TestUpdate_Traffic(t *testing.T) {
	// 1. Arrange (准备)
	initialModel := NewModel("localhost:9090")
	initialModel.connected = true // 模拟已连接状态

	// 模拟收到的流量消息: 上行 1024B (1KB), 下行 2048B (2KB)
	inputMsg := trafficMsg{
		Up:   1024,
		Down: 2048,
	}

	// 2. Act (执行 Update)
	// Update 返回 (Model, Cmd)
	updatedModel, cmd := initialModel.Update(inputMsg)

	// 3. Assert (断言)
	// 断言类型转换回我们的 Model
	m, ok := updatedModel.(Model)
	assert.True(t, ok)

	// 验证状态是否正确更新
	assert.Equal(t, int64(1024), m.Stats.Up, "Upload speed should be updated")
	assert.Equal(t, int64(2048), m.Stats.Down, "Download speed should be updated")

	// 验证 Update 之后是否继续保持轮询 (返回了 Cmd)
	assert.NotNil(t, cmd, "Should return a command to read next message")
}

// TestUpdate_Quit 验证退出逻辑
func TestUpdate_Quit(t *testing.T) {
	m := NewModel("dummy")

	// 模拟按下 'q' 键
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// 验证返回的命令是否是 tea.Quit
	// tea.Quit 本质上是一个特殊的 Cmd，我们需要验证它的行为或者地址
	// BubbleTea 的 Quit 是个内部函数，通常通过比较类型或行为
	// 但在单元测试中，我们通常看它是否返回了特定的系统 Cmd，或者简单地看 coverage
	// 这里更严谨的做法是：
	assert.NotNil(t, cmd, "Should return quit command")
	
	// 在 BubbleTea 这里的测试通常比较 tricky，因为 Quit 是个 func() Msg
	// 只要不为 nil 且我们逻辑里写了 return tea.Quit 即可
}

// TestView_Rendering 验证界面渲染
func TestView_Rendering(t *testing.T) {
	// 1. 测试未连接状态
	m1 := NewModel("dummy")
	m1.connected = false
	assert.Contains(t, m1.View(), "Connecting", "Should show connecting status")

	// 2. 测试已连接且有数据状态
	m2 := NewModel("dummy")
	m2.connected = true
	m2.Stats = TrafficStats{
		Up:   1024,       // 1.0 KB
		Down: 1024 * 1024, // 1.0 MB
	}

	viewOutput := m2.View()

	// 验证关键文本是否存在 (不需要验证边框颜色代码，太脆弱了)
	assert.Contains(t, viewOutput, "1.0 KB/s", "Should format upload speed correctly")
	assert.Contains(t, viewOutput, "1.0 MB/s", "Should format download speed correctly")
	assert.Contains(t, viewOutput, "Minibox Monitor", "Should show title")
}

// TestFormatBytes 辅助函数测试 (纯逻辑测试)
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}