package monitor

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/adapter/logger"
)


func(m Model) Init() tea.Cmd {
	return connectWS(m.apiBase)
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