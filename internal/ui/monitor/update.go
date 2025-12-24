package monitor

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/client"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		connectWS(m.apiBase),      // 连 WS
		fetchProxies(m.apiClient), // 拉节点列表
	)
}

// Update 处理所有事件 (键盘输入、IO 消息)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// 1. 处理按键
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			logger.Info("quit monitor")
			return m, tea.Quit // 退出程序
		case "up":
			if m.Expanded {
				if m.CursorNode > 0 {
					m.CursorNode--
				}
			} else {
				if m.CursorGroup > 0 {
					m.CursorGroup--
				}
			}
		case "down":
			if m.Expanded {
				if m.CursorNode < len(m.ExpandedList)-1 {
					m.CursorNode++
				}
			} else {
				if m.CursorGroup < len(m.Groups)-1 {
					m.CursorGroup++
				}
			}
			// --- 展开/收起/选择逻辑 ---
		case "enter":
			if !m.Expanded {
				// 1. 展开组
				m.Expanded = true
				currentGroup := m.Groups[m.CursorGroup]
				m.ExpandedList = m.Proxies[currentGroup].All
				m.CursorNode = 0 // 重置节点光标
			} else {
				// 2. 选中节点并切换
				group := m.Groups[m.CursorGroup]
				node := m.ExpandedList[m.CursorNode]

				// 执行切换 (异步命令)
				return m, switchNode(m.apiClient, group, node)
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
		}

	// 2. 处理连接成功
	case connMsg:
		m.wsConn = msg.conn
		m.connected = true
		return m, readTraffic(m.wsConn)

	// 3. 处理流量更新
	case trafficMsg:
		m.Stats = TrafficStats(msg)
		// 处理完一条数据，继续读下一条 (循环)
		return m, readTraffic(m.wsConn)

	// 4. 处理错误
	case errMsg:
		m.Err = msg
		return m, tea.Quit

	// 获取所有节点
	case proxiesMsg:
		m.Proxies = msg
		// 提取所有的 Selector 组名并排序
		m.Groups = extractSelectorGroups(msg)
		return m, nil

		// 处理 switchNode 成功的 Msg，重新拉取列表以刷新 "Now" 状态
	case nodeSwitchedMsg:
		m.Expanded = false                  // 切换成功后收起
		return m, fetchProxies(m.apiClient) // 刷新列表

	// 处理测速结果
	case latencyMsg:
		// 清除测试中状态
		delete(m.TestingNodes, msg.Name)
		// 保存延迟结果
		m.Latencies[msg.Name] = msg.Delay
		return m, nil
	}

	return m, nil
}

func connectWS(host string) tea.Cmd {
	return func() tea.Msg {
		u := "ws://" + host + "/traffic?token=" // 如果有 token 需要加在这里
		conn, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			logger.Error("connect to ws failed", "error", err)
			return errMsg(err)
		}
		return connMsg{conn: conn}
	}
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
			logger.Error("read traffic failed", "error", err)
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

// extractSelectorGroups 从所有代理中提取出类型为 Selector 的组名，并排序
func extractSelectorGroups(proxies map[string]client.ProxyData) []string {
	var groups []string

	for name, data := range proxies {
		// 关键点：我们只显示 "Selector" 类型
		// 因为只有这种类型可以包含其他节点，允许用户切换
		// 忽略 "Direct", "Reject", "Vmess", "Shadowsocks" 等具体节点
		if data.Type == "Selector" {
			groups = append(groups, name)
		}
	}

	// 必须排序！如果不排序，Go 的 map 遍历顺序是随机的
	// 会导致你的 TUI 界面每次刷新时，列表顺序都在变，光标会乱跳
	sort.Strings(groups)

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
