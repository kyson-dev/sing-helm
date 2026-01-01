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
	initialModel.connState.OnConnected() // 模拟已连接状态

	// 模拟收到的流量消息: 上行 1024B (1KB), 下行 2048B (2KB)
	inputMsg := trafficMsg{
		Up:   1024,
		Down: 2048,
	}

	// 2. Act (执行 Update)
	updatedModel, cmd := initialModel.Update(inputMsg)

	// 3. Assert (断言)
	m, ok := updatedModel.(Model)
	assert.True(t, ok)

	// 验证状态是否正确更新
	traffic := m.Traffic()
	assert.Equal(t, int64(1024), traffic.Up, "Upload speed should be updated")
	assert.Equal(t, int64(2048), traffic.Down, "Download speed should be updated")

	// 验证 Update 之后是否继续保持轮询 (返回了 Cmd)
	assert.NotNil(t, cmd, "Should return a command to read next message")
}

// TestUpdate_Quit 验证退出逻辑
func TestUpdate_Quit(t *testing.T) {
	m := NewModel("dummy")

	// 模拟按下 'q' 键
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	assert.NotNil(t, cmd, "Should return quit command")
}

// TestView_Rendering 验证界面渲染
func TestView_Rendering(t *testing.T) {
	// 1. 测试未连接状态
	m1 := NewModel("dummy")
	// 默认已经是 Connecting 状态
	assert.Contains(t, m1.View(), "Connecting", "Should show connecting status")

	// 2. 测试已连接且有数据状态
	m2 := NewModel("dummy")
	m2.connState.OnConnected()

	// 通过 Update 消息来更新状态，而不是直接修改私有字段
	m2, _ = m2.handleTraffic(trafficMsg{
		Up:   1024,        // 1.0 KB
		Down: 1024 * 1024, // 1.0 MB
	})

	viewOutput := m2.View()

	// 验证关键文本是否存在
	assert.Contains(t, viewOutput, "1.0 KB/s", "Should format upload speed correctly")
	assert.Contains(t, viewOutput, "1.0 MB/s", "Should format download speed correctly")
	assert.Contains(t, viewOutput, "Minibox Monitor", "Should show title")

	// 验证 connected 状态显示
	assert.Contains(t, viewOutput, "Connected", "Should show connected status")
}

// TestFormatBytes 辅助函数测试
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

// TestUpdate_ModeSwitch 验证模式切换逻辑
func TestUpdate_ModeSwitch(t *testing.T) {
	m := NewModel("dummy")
	m.connState.OnConnected()

	// 1. 模拟按下 'm' 键 (切换 mode)
	// 这个操作会触发 updating 标志设置为 true
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updatedModel.(Model)

	assert.NotNil(t, cmd)
	assert.True(t, m.IsUpdating(), "Should be updating after mode switch request")

	// 2. 模拟收到 modeChangedMsg (成功)
	updatedModel, cmd = m.Update(modeChangedMsg{NewMode: "tun", Err: nil})
	m = updatedModel.(Model)

	// 验证状态
	assert.False(t, m.IsUpdating(), "Should stop updating after success")
	assert.Equal(t, "tun", m.ProxyMode(), "Mode should be updated")
	assert.NotNil(t, cmd, "Should trigger status fetch")
}

// TestUpdate_ConcurrentRequests 验证并发请求保护
func TestUpdate_ConcurrentRequests(t *testing.T) {
	m := NewModel("dummy")
	m.connState.OnConnected()

	// 1. 发起第一个请求 (Mode)
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updatedModel.(Model)
	assert.True(t, m.IsUpdating())

	// 2. 尝试发起第二个请求 (Route) - 应该被忽略
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updatedModel.(Model)

	assert.Nil(t, cmd, "Concurrent request should be ignored (return nil cmd)")
	assert.True(t, m.IsUpdating(), "Should still be updating")

	// 3. 完成第一个请求
	updatedModel, _ = m.Update(modeChangedMsg{NewMode: "tun", Err: nil})
	m = updatedModel.(Model)
	assert.False(t, m.IsUpdating())

	// 4. 现在可以发起新请求
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.NotNil(t, cmd, "New request should be accepted after previous one finishes")
}

// TestSmartReconnect 验证智能重连
func TestSmartReconnect(t *testing.T) {
	m := NewModel("dummy")
	m.connState.OnConnected()

	// 1. 正常断链 -> 立即重连 (1s)
	updatedModel, _ := m.Update(disconnectedMsg{err: nil})
	m = updatedModel.(Model)

	// 这里很难直接验证 cmd 内容，但我们可以验证状态
	assert.Equal(t, "Connecting", m.ConnState().String())

	// 2. 更新期间断链 -> 延迟重连 (2s)
	m.connState.OnConnected()
	m.setUpdating()

	updatedModel, _ = m.Update(disconnectedMsg{err: nil})
	m = updatedModel.(Model)

	// 状态应该是 Connecting (disconnectedMsg 会触发 OnDisconnected)
	assert.Equal(t, "Connecting", m.ConnState().String())
	assert.True(t, m.IsUpdating())
}
