package monitor

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyson/minibox/internal/adapter/logger"
)

// ============================================================================
// 消息处理器
// ============================================================================

// -----------------------------------------------------------------------------
// 1. 连接相关处理器
// -----------------------------------------------------------------------------

// handleConnected 处理连接成功
func (m *Model) handleConnected(msg connectedMsg) (Model, tea.Cmd) {
	m.wsConn = msg.conn
	m.connState.OnConnected()
	m.lastError = nil

	// 开始读取流量 + 拉取数据
	return *m, tea.Batch(
		cmdReadTraffic(m.wsConn),
		cmdFetchProxies(m.apiClient),
		cmdFetchStatus(m.apiClient),
	)
}

// handleDisconnected 处理连接断开
func (m *Model) handleDisconnected(msg disconnectedMsg) (Model, tea.Cmd) {
	m.lastError = msg.err
	m.connState.OnDisconnected() // 自动进入 Connecting 状态

	// 关闭旧连接
	if m.wsConn != nil {
		m.wsConn.Close()
		m.wsConn = nil
	}

	// 智能重连：检查是否正在更新 mode/route
	delay := time.Second
	if m.isUpdating() {
		delay = 2 * time.Second
	}

	return *m, cmdReconnectAfter(delay)
}

// handleReconnectTick 处理重连计时器
func (m *Model) handleReconnectTick() (Model, tea.Cmd) {
	// 如果正在更新，继续等待
	if m.isUpdating() {
		return *m, cmdReconnectAfter(500 * time.Millisecond)
	}

	// 直接发起连接（状态已经是 Connecting）
	return *m, cmdConnect(m.apiBase)
}

// -----------------------------------------------------------------------------
// 2. 数据更新处理器
// -----------------------------------------------------------------------------

// handleTraffic 处理流量数据
func (m *Model) handleTraffic(msg trafficMsg) (Model, tea.Cmd) {
	m.traffic.Up = msg.Up
	m.traffic.Down = msg.Down
	return *m, cmdReadTraffic(m.wsConn)
}

// handleStatus 处理状态信息
func (m *Model) handleStatus(msg statusMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		return *m, cmdStatusTick(m.apiClient)
	}

	m.proxyMode = msg.ProxyMode
	m.routeMode = msg.RouteMode
	m.connections = msg.Connections
	m.memory = msg.Memory
	m.traffic.TotalUp = msg.TotalUp
	m.traffic.TotalDown = msg.TotalDown

	return *m, cmdStatusTick(m.apiClient)
}

// handleProxies 处理代理列表
func (m *Model) handleProxies(msg proxiesMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		return *m, nil
	}

	m.proxies = msg.Proxies
	m.groups = extractGroups(msg.Proxies)

	// 首次加载时自动展开第一个组并测速
	if len(m.groups) > 0 && !m.expanded {
		m.expanded = true
		m.cursor.Group = 0
		m.cursor.Node = 0
		m.expandedList = m.proxies[m.groups[0]].All

		var cmds []tea.Cmd
		for _, nodeName := range m.expandedList {
			m.testing[nodeName] = true
			cmds = append(cmds, cmdTestLatency(m.apiClient, nodeName))
		}
		return *m, tea.Batch(cmds...)
	}

	return *m, nil
}

// handleLatency 处理延迟测试结果
func (m *Model) handleLatency(msg latencyMsg) (Model, tea.Cmd) {
	delete(m.testing, msg.Name)
	m.latencies[msg.Name] = msg.Delay
	return *m, nil
}

// -----------------------------------------------------------------------------
// 3. Mode/Route 切换结果处理器
// -----------------------------------------------------------------------------

// handleModeChanged 处理 mode 切换结果
func (m *Model) handleModeChanged(msg modeChangedMsg) (Model, tea.Cmd) {
	// 清除更新标志
	m.clearUpdating()

	if msg.Err == nil {
		m.proxyMode = msg.NewMode
	}

	// mode 切换会导致 sing-box 重启，连接会自动断开并重连
	// 不需要手动触发重连，handleDisconnected 会处理
	return *m, cmdFetchStatus(m.apiClient)
}

// handleRouteChanged 处理 route 切换结果
func (m *Model) handleRouteChanged(msg routeChangedMsg) (Model, tea.Cmd) {
	// 清除更新标志
	m.clearUpdating()

	if msg.Err == nil {
		m.routeMode = msg.NewRoute
	}

	return *m, cmdFetchStatus(m.apiClient)
}

// handleNodeChanged 处理 node 切换结果（不需要锁）
func (m *Model) handleNodeChanged(msg nodeChangedMsg) (Model, tea.Cmd) {
	// node 切换不会断链，只需刷新列表
	return *m, cmdFetchProxies(m.apiClient)
}

// -----------------------------------------------------------------------------
// 4. 键盘输入处理器
// -----------------------------------------------------------------------------

// handleKeyPress 处理键盘输入
func (m *Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return *m, tea.Quit

	case "up":
		return m.handleKeyUp()

	case "down":
		return m.handleKeyDown()

	case "left":
		return m.handleKeyLeft()

	case "right":
		return m.handleKeyRight()

	case "enter":
		return m.handleKeyEnter()

	case "t":
		return m.handleKeyTest()

	case "m":
		return m.handleKeyMode()

	case "r":
		return m.handleKeyRoute()
	}

	return *m, nil
}

// handleKeyUp 处理向上键
func (m *Model) handleKeyUp() (Model, tea.Cmd) {
	if m.expanded {
		if m.cursor.Node > 0 {
			m.cursor.Node--
		} else if m.cursor.Group > 0 {
			m.cursor.Group--
			m.expandedList = m.proxies[m.groups[m.cursor.Group]].All
			m.cursor.Node = len(m.expandedList) - 1
		} else {
			m.cursor.Group = len(m.groups) - 1
			m.expandedList = m.proxies[m.groups[m.cursor.Group]].All
			m.cursor.Node = len(m.expandedList) - 1
		}
	} else {
		if m.cursor.Group > 0 {
			m.cursor.Group--
		} else {
			m.cursor.Group = len(m.groups) - 1
		}
	}
	return *m, nil
}

// handleKeyDown 处理向下键
func (m *Model) handleKeyDown() (Model, tea.Cmd) {
	if m.expanded {
		if m.cursor.Node < len(m.expandedList)-1 {
			m.cursor.Node++
		} else if m.cursor.Group < len(m.groups)-1 {
			m.cursor.Group++
			m.expandedList = m.proxies[m.groups[m.cursor.Group]].All
			m.cursor.Node = 0
		} else {
			m.cursor.Group = 0
			m.expandedList = m.proxies[m.groups[m.cursor.Group]].All
			m.cursor.Node = 0
		}
	} else {
		if m.cursor.Group < len(m.groups)-1 {
			m.cursor.Group++
		} else {
			m.cursor.Group = 0
		}
	}
	return *m, nil
}

// handleKeyLeft 处理向左键（收起）
func (m *Model) handleKeyLeft() (Model, tea.Cmd) {
	if m.expanded {
		m.expanded = false
	}
	return *m, nil
}

// handleKeyRight 处理向右键（展开）
func (m *Model) handleKeyRight() (Model, tea.Cmd) {
	if !m.expanded && len(m.groups) > 0 {
		m.expanded = true
		m.expandedList = m.proxies[m.groups[m.cursor.Group]].All
		m.cursor.Node = 0
	}
	return *m, nil
}

// handleKeyEnter 处理回车键（切换节点）
func (m *Model) handleKeyEnter() (Model, tea.Cmd) {
	if !m.expanded || len(m.expandedList) == 0 {
		return m.handleKeyRight()
	}

	// node 切换不需要锁，直接发送请求
	group := m.groups[m.cursor.Group]
	node := m.expandedList[m.cursor.Node]

	return *m, cmdSwitchNode(m.apiClient, group, node)
}

// handleKeyTest 处理测速键
func (m *Model) handleKeyTest() (Model, tea.Cmd) {
	if !m.expanded {
		return *m, nil
	}

	var cmds []tea.Cmd
	for _, nodeName := range m.expandedList {
		m.testing[nodeName] = true
		cmds = append(cmds, cmdTestLatency(m.apiClient, nodeName))
	}

	return *m, tea.Batch(cmds...)
}

// handleKeyMode 处理 mode 切换键
// 使用互斥锁确保同一时间只有一个 mode/route 请求
func (m *Model) handleKeyMode() (Model, tea.Cmd) {
	// 检查是否正在更新
	if m.isUpdating() {
		logger.Debug("handleKeyMode", "rejected", "already updating")
		return *m, nil
	}

	// 设置更新标志（这个操作是同步的，在同一个事件循环内）
	m.setUpdating()
	logger.Debug("handleKeyMode", "started", m.proxyMode)

	return *m, cmdSwitchMode(m.proxyMode)
}

// handleKeyRoute 处理 route 切换键
func (m *Model) handleKeyRoute() (Model, tea.Cmd) {
	// 检查是否正在更新
	if m.isUpdating() {
		logger.Debug("handleKeyRoute", "rejected", "already updating")
		return *m, nil
	}

	// 设置更新标志
	m.setUpdating()
	logger.Debug("handleKeyRoute", "started", m.routeMode)

	return *m, cmdSwitchRoute(m.routeMode)
}

// =============================================================================
// 更新状态管理（使用互斥锁保护）
// =============================================================================

var updateMutex sync.Mutex
var updating bool

// isUpdating 检查是否正在更新 mode/route
func (m *Model) isUpdating() bool {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	return updating
}

// setUpdating 设置更新标志
func (m *Model) setUpdating() {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	updating = true
}

// clearUpdating 清除更新标志
func (m *Model) clearUpdating() {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	updating = false
}
