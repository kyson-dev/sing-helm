package monitor

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

// --- 1. 数据结构定义 ---

// TrafficStats 对应 Clash API 的流量格式
type TrafficStats struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// Model 是 BubbleTea 的核心状态容器
type Model struct {
	Stats     TrafficStats // 当前流量数据
	Err       error        // 错误状态
	wsConn    *websocket.Conn
	apiBase   string       // API 地址，如 127.0.0.1:9090
	connected bool
}

// NewModel 初始化模型
func NewModel(apiHost string) Model {
	return Model{
		apiBase: apiHost,
	}
}

// --- 2. 消息定义 (Msg) ---
// BubbleTea 是基于消息驱动的，我们需要定义几种消息类型

type trafficMsg TrafficStats // 接收到新流量数据的消息
type connMsg struct {        // 连接成功的消息 
	conn *websocket.Conn
}             
type errMsg error            // 出错消息

// --- 3. 样式定义 (Styles) ---
var (
	// 定义一个蓝色的边框样式
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Align(lipgloss.Center)

	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true)

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0"))
)