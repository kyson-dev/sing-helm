package monitor

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// BubbleTea 接口实现
// ============================================================================

// Init 初始化，发起连接和数据拉取
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		cmdConnect(m.apiBase),
	)
}

// Update 消息分发器
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// =========================================================================
	// 连接相关消息
	// =========================================================================
	case connectedMsg:
		newM, cmd := m.handleConnected(msg)
		return newM, cmd

	case disconnectedMsg:
		newM, cmd := m.handleDisconnected(msg)
		return newM, cmd

	case reconnectTickMsg:
		newM, cmd := m.handleReconnectTick()
		return newM, cmd

	// =========================================================================
	// 数据更新消息
	// =========================================================================
	case trafficMsg:
		newM, cmd := m.handleTraffic(msg)
		return newM, cmd

	case statusMsg:
		newM, cmd := m.handleStatus(msg)
		return newM, cmd

	case proxiesMsg:
		newM, cmd := m.handleProxies(msg)
		return newM, cmd

	case latencyMsg:
		newM, cmd := m.handleLatency(msg)
		return newM, cmd

	// =========================================================================
	// 请求结果消息
	// =========================================================================
	case modeChangedMsg:
		newM, cmd := m.handleModeChanged(msg)
		return newM, cmd

	case routeChangedMsg:
		newM, cmd := m.handleRouteChanged(msg)
		return newM, cmd

	case nodeChangedMsg:
		newM, cmd := m.handleNodeChanged(msg)
		return newM, cmd

	// =========================================================================
	// 键盘输入
	// =========================================================================
	case tea.KeyMsg:
		newM, cmd := m.handleKeyPress(msg)
		return newM, cmd
	}

	return m, nil
}
