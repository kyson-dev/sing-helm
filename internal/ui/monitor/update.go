package monitor

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/core/client"
	"github.com/kyson/minibox/internal/core/controller"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		connectWS(m.apiBase),      // 连 WS
		fetchProxies(m.apiClient), // 拉节点列表
		fetchStatus(m.apiClient),  // 获取状态信息
	)
}

// Update 处理所有事件 (键盘输入、IO 消息)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// 1. 处理按键
	// case tea.WindowSizeMsg:
	// 	m.Width = msg.Width
	// 	m.Height = msg.Height
	// 	return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			// logger.Info("quit monitor")
			return m, tea.Quit // 退出程序
		case "up":
			if m.Expanded {
				if m.CursorNode > 0 {
					// 正常向上移动
					m.CursorNode--
				} else {
					// 在第一个节点，切换到上一组
					if m.CursorGroup > 0 {
						m.CursorGroup--
					} else {
						// 第一组循环到最后一组
						m.CursorGroup = len(m.Groups) - 1
					}
					// 展开新组，光标在最后一个节点
					newGroup := m.Groups[m.CursorGroup]
					m.ExpandedList = m.Proxies[newGroup].All
					m.CursorNode = len(m.ExpandedList) - 1
				}
			} else {
				if m.CursorGroup > 0 {
					m.CursorGroup--
				} else {
					// 循环到最后一组
					m.CursorGroup = len(m.Groups) - 1
				}
			}
		case "down":
			if m.Expanded {
				if m.CursorNode < len(m.ExpandedList)-1 {
					// 正常向下移动
					m.CursorNode++
				} else {
					// 在最后一个节点，切换到下一组
					if m.CursorGroup < len(m.Groups)-1 {
						m.CursorGroup++
					} else {
						// 最后一组循环到第一组
						m.CursorGroup = 0
					}
					// 展开新组，光标在第一个节点
					newGroup := m.Groups[m.CursorGroup]
					m.ExpandedList = m.Proxies[newGroup].All
					m.CursorNode = 0
				}
			} else {
				if m.CursorGroup < len(m.Groups)-1 {
					m.CursorGroup++
				} else {
					// 循环到第一组
					m.CursorGroup = 0
				}
			}
			// --- 展开/收起逻辑 ---
		case "right":
			if !m.Expanded && len(m.Groups) > 0 {
				// 展开组
				m.Expanded = true
				currentGroup := m.Groups[m.CursorGroup]
				m.ExpandedList = m.Proxies[currentGroup].All
				m.CursorNode = 0 // 重置节点光标
			}

		// --- 选择节点 (不收起) ---
		case "enter": // Enter 键
			if m.Expanded && len(m.ExpandedList) > 0 {
				// 检查是否空闲
				if m.RequestState != RequestStateIdle {
					return m, nil // 有请求进行中，忽略
				}
				group := m.Groups[m.CursorGroup]
				node := m.ExpandedList[m.CursorNode]
				// 设置请求状态
				m.RequestState = RequestingNode
				// 执行切换 (异步命令)，不收起列表
				return m, switchNode(m.apiClient, group, node)
			} else {
				// 选择展开
				return m, func() tea.Msg {
					return tea.KeyMsg{
						Type:  tea.KeyRunes,
						Runes: []rune("right"),
					}
				}
			}

		case "left":
			if m.Expanded {
				m.Expanded = false // 收起
			}
		case "t":
			// 只有在展开节点列表时才允许测速
			if m.Expanded {
				var cmds []tea.Cmd

				// 遍历当前展开的所有节点
				for _, nodeName := range m.ExpandedList {
					// 标记为正在测试
					m.TestingNodes[nodeName] = true
					// 启动协程去测速
					cmds = append(cmds, checkNodeLatency(m.apiClient, nodeName))
				}

				// tea.Batch 可以并行执行多个 Cmd
				return m, tea.Batch(cmds...)
			}
		case "m": // 切换代理模式
			// 检查是否空闲
			if m.RequestState != RequestStateIdle {
				return m, nil // 有请求进行中，忽略
			}
			m.RequestState = RequestingMode
			// 启动超时保护（5秒后自动重置）
			return m, tea.Batch(
				switchProxyMode(m.ProxyMode),
				tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
					return requestTimeoutMsg{RequestType: "mode"}
				}),
			)
		case "r": // 切换路由模式
			// 检查是否空闲
			if m.RequestState != RequestStateIdle {
				return m, nil // 有请求进行中，忽略
			}
			m.RequestState = RequestingRoute
			// 启动超时保护（5秒后自动重置）
			return m, tea.Batch(
				switchRouteMode(m.RouteMode),
				tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
					return requestTimeoutMsg{RequestType: "route"}
				}),
			)
		}

	// 2. 处理连接成功
	case connMsg:
		m.wsConn = msg.conn
		m.connected = true
		m.ConnState = ConnStateConnected
		m.Err = nil

		// 重置请求状态（新连接，清除旧状态）
		m.RequestState = RequestStateIdle

		return m, tea.Batch(readTraffic(m.wsConn), fetchProxies(m.apiClient), fetchStatus(m.apiClient))

	// 3. 处理流量更新
	case trafficMsg:
		m.Stats = TrafficStats(msg)
		// 处理完一条数据，继续读下一条 (循环)
		// 性能优化：只循环流量，不触发高开销的 IPC 状态查询
		return m, readTraffic(m.wsConn)

	// 4. 处理错误 - 触发重连而不是退出
	case errMsg:
		// logger.Info("Connection error, will reconnect", "error", msg)
		m.Err = msg
		m.connected = false
		m.ConnState = ConnStateReconnecting

		// 重置请求状态（因为连接断开了）
		m.RequestState = RequestStateIdle

		if m.wsConn != nil {
			m.wsConn.Close()
			m.wsConn = nil
		}
		return m, reconnectAfterDelay(1 * time.Second)

	// 5. 处理重连消息
	case reconnectMsg:
		// logger.Info("Attempting to reconnect...")
		// 确保请求状态已重置
		m.RequestState = RequestStateIdle
		return m, connectWS(m.apiBase)
	case proxiesMsg:
		m.Proxies = msg
		// 提取所有的 Selector 组名并排序
		m.Groups = extractSelectorGroups(msg)

		// 默认展开第一个组并自动测速
		if len(m.Groups) > 0 && !m.Expanded {
			m.Expanded = true
			m.CursorGroup = 0
			m.ExpandedList = m.Proxies[m.Groups[0]].All
			m.CursorNode = 0

			// 自动触发测速：发送 't' 按键消息
			return m, func() tea.Msg {
				return tea.KeyMsg{
					Type:  tea.KeyRunes,
					Runes: []rune{'t'},
				}
			}
		}

		return m, nil

		// 处理 switchNode 成功的 Msg，重新拉取列表以刷新 "Now" 状态
	case nodeSwitchedMsg:
		//m.Expanded = false                  // 切换成功后收起
		m.RequestState = RequestStateIdle   // 重置为空闲
		return m, fetchProxies(m.apiClient) // 刷新列表

	// 处理测速结果
	case latencyMsg:
		// 清除测试中状态
		delete(m.TestingNodes, msg.Name)
		// 保存延迟结果
		m.Latencies[msg.Name] = msg.Delay
		return m, nil

	// 处理状态信息更新
	case statusMsg:
		m.ProxyMode = msg.ProxyMode // 更新 ProxyMode
		m.RouteMode = msg.RouteMode
		m.Connections = msg.Connections
		m.Memory = msg.Memory
		m.TotalUp = msg.TotalUp
		m.TotalDown = msg.TotalDown

		// 建立 1秒 的独立心跳，低频查询状态，避免 IPC 风暴
		return m, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
			return fetchStatus(m.apiClient)()
		})

	// 处理模式切换结果
	case modeChangedMsg:
		m.RequestState = RequestStateIdle // 重置为空闲
		if msg.Err == nil {
			m.ProxyMode = msg.Mode
		}
		// mode 切换会导致 sing-box 重启，需要重新连接 WebSocket
		if m.wsConn != nil {
			m.wsConn.Close()
			m.wsConn = nil
		}
		m.ConnState = ConnStateReconnecting
		return m, tea.Batch(
			fetchStatus(m.apiClient),
			reconnectAfterDelay(500*time.Millisecond), // 等待 sing-box 重启
		)

	case routeChangedMsg:
		m.RequestState = RequestStateIdle // 重置为空闲
		if msg.Err == nil {
			m.RouteMode = msg.Mode
		}
		// route 切换会导致 sing-box 重启，需要重新连接 WebSocket
		if m.wsConn != nil {
			m.wsConn.Close()
			m.wsConn = nil
		}
		m.ConnState = ConnStateReconnecting
		return m, tea.Batch(
			fetchStatus(m.apiClient),
			reconnectAfterDelay(500*time.Millisecond), // 等待 sing-box 重启
		)

	// 处理请求超时（防止状态卡住）
	case requestTimeoutMsg:
		// 只在对应的请求状态下才重置
		switch msg.RequestType {
		case "mode":
			if m.RequestState == RequestingMode {
				m.RequestState = RequestStateIdle
			}
		case "route":
			if m.RequestState == RequestingRoute {
				m.RequestState = RequestStateIdle
			}
		case "node":
			if m.RequestState == RequestingNode {
				m.RequestState = RequestStateIdle
			}
		}
		return m, nil
	}

	return m, nil
}

func connectWS(host string) tea.Cmd {
	return func() tea.Msg {
		u := "ws://" + host + "/traffic?token=" // 如果有 token 需要加在这里
		conn, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			// logger.Error("connect to ws failed", "error", err)
			return errMsg(err)
		}
		return connMsg{conn: conn}
	}
}

// reconnectAfterDelay 延迟重连
func reconnectAfterDelay(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return reconnectMsg{}
	})
}

// fetchProxies 异步拉取节点
func fetchProxies(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		p, err := c.GetProxies()
		if err != nil {
			return errMsg(err)
		}
		return proxiesMsg(p)
	}
}

// readTraffic 读取下一条 WebSocket 消息
func readTraffic(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		if conn == nil {
			return nil
		}
		var stats TrafficStats
		// ReadJSON 会阻塞，直到有新数据
		if err := conn.ReadJSON(&stats); err != nil {
			// logger.Error("read traffic failed", "error", err)
			return errMsg(err)
		}
		return trafficMsg(stats)
	}
}

func switchNode(c *client.Client, group, node string) tea.Cmd {
	return func() tea.Msg {
		err := c.SelectProxy(group, node)
		if err != nil {
			return errMsg(err)
		}
		return nodeSwitchedMsg{}
	}
}

// extractSelectorGroups 从所有代理中提取出可切换的组（Selector 和 URLTest）
func extractSelectorGroups(proxies map[string]client.ProxyData) []string {
	var groups []string

	for name, data := range proxies {
		// Selector: 手动选择组（如 proxy）
		// URLTest: 自动测速组（如 auto）
		if data.Type == "Selector" || data.Type == "URLTest" {
			groups = append(groups, name)
		}
	}

	// 自定义排序：auto 放在最后，其他按字母顺序
	sort.Slice(groups, func(i, j int) bool {
		// 如果 i 是 auto，放在后面
		if groups[i] == "auto" {
			return false
		}
		// 如果 j 是 auto，i 放在前面
		if groups[j] == "auto" {
			return true
		}
		// 其他情况按字母顺序
		return groups[i] < groups[j]
	})

	return groups
}

// checkNodeLatency 创建测速命令
func checkNodeLatency(c *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		// 使用 Google 生成页进行测试，超时 2000ms
		delay, err := c.GetNodeDelay(name, "http://www.gstatic.com/generate_204", 2000)
		if err != nil {
			// -1 表示失败/超时
			return latencyMsg{Name: name, Delay: -1}
		}
		return latencyMsg{Name: name, Delay: delay}
	}
}

// fetchStatus 获取状态信息（路由模式、连接数、内存、总流量）
func fetchStatus(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		// 1. 从 API 获取动态数据 (连接、内存、流量)
		conns, err := c.GetConnections()
		if err != nil {
			return statusMsg{} // 出错时返回空状态
		}

		// 2. 从 daemon 状态获取配置模式 (ProxyMode, RouteMode)
		// API 不会返回正确的业务模式
		proxyMode := "unknown"
		routeMode := "unknown"

		if status, err := controller.FetchStatus(context.Background()); err == nil {
			if status.ProxyMode != "" {
				proxyMode = status.ProxyMode
			}
			if status.RouteMode != "" {
				routeMode = status.RouteMode
			}
		}

		return statusMsg{
			ProxyMode:   proxyMode,
			RouteMode:   routeMode,
			Connections: len(conns.Connections),
			Memory:      conns.Memory,
			TotalUp:     conns.UploadTotal,
			TotalDown:   conns.DownloadTotal,
		}
	}
}

// switchProxyMode 切换代理模式 (通过 IPC 触发 daemon 重载)
// 智能跳过 TUN：如果 daemon 不是 root 权限，自动跳过 TUN 模式
func switchProxyMode(current string) tea.Cmd {
	return func() tea.Msg {
		// 检查 daemon 是否以 root 权限运行
		status, err := controller.FetchStatus(context.Background())
		isRoot := false
		if err == nil {
			// 通过检查 PID 对应的进程权限来判断
			// 简化方案：尝试切换到 TUN，如果失败则跳过
			isRoot = (status.PID > 0) // 这里需要更准确的检查，暂时简化
		}

		// 循环切换逻辑
		var next string
		switch strings.ToLower(current) {
		case "system":
			if isRoot {
				next = "tun"
			} else {
				// 非 root，跳过 TUN，直接到 default
				next = "default"
			}
		case "tun":
			next = "default"
		case "default":
			next = "system"
		default:
			next = "system"
		}

		// 通过 IPC 调用 daemon 切换模式（会触发 sing-box 重载）
		_, err = controller.SwitchProxyMode(next)
		if err != nil {
			// 如果是 TUN 权限错误，自动跳到下一个模式
			if strings.Contains(err.Error(), "root permission") || strings.Contains(err.Error(), "permission") {
				// 跳过 TUN，尝试下一个模式
				if next == "tun" {
					next = "default"
					_, err = controller.SwitchProxyMode(next)
					if err != nil {
						return modeChangedMsg{Mode: current, Err: err}
					}
					return modeChangedMsg{Mode: next, Err: nil}
				}
			}
			return modeChangedMsg{Mode: current, Err: err}
		}

		return modeChangedMsg{Mode: next, Err: nil}
	}
}

// switchRouteMode 切换路由模式 (通过 API 实时更新)
// switchRouteMode 切换路由模式 (通过 IPC 触发 daemon 重载)
func switchRouteMode(current string) tea.Cmd {
	return func() tea.Msg {
		// 循环切换: rule -> global -> direct -> rule
		var next string
		switch strings.ToLower(current) {
		case "rule":
			next = "global"
		case "global":
			next = "direct"
		default:
			next = "rule"
		}

		// 通过 IPC 调用 daemon 切换模式（确保生效并持久化）
		_, err := controller.SwitchRouteMode(next)
		if err != nil {
			// logger.Error("Failed to switch route mode", "error", err, "from", current, "to", next)
			return routeChangedMsg{Mode: current, Err: err}
		}
		// logger.Info("Route mode switched via IPC", "from", current, "to", next)
		return routeChangedMsg{Mode: next, Err: nil}
	}
}
