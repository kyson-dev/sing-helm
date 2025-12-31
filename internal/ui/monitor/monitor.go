package monitor

import (
	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/core/client"
)

// --- 1. 数据结构定义 ---

// ConnState 连接状态
type ConnState int

const (
	ConnStateConnecting   ConnState = iota // 正在连接
	ConnStateConnected                     // 已连接
	ConnStateReconnecting                  // 重连中
	ConnStateError                         // 错误
)

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
	apiBase   string // API 地址，如 127.0.0.1:9090
	connected bool
	apiClient *client.Client

	// --- 连接状态 ---
	ConnState ConnState // 连接状态

	// --- 累计流量 ---
	TotalUp   int64 // 累计上传
	TotalDown int64 // 累计下载

	// --- 状态信息 ---
	ProxyMode   string // 代理模式: system, tun, default
	RouteMode   string // 路由模式: rule, global, direct
	Connections int    // 当前活跃连接数
	Memory      uint64 // 内存使用 (bytes)

	// --- 节点列表状态 ---
	Groups  []string                    // 所有的组名 (用于排序显示)
	Proxies map[string]client.ProxyData // 所有代理详情

	// --- 测速状态 ---
	// -1 表示超时或错误, 0 表示未测试
	Latencies    map[string]int
	TestingNodes map[string]bool // 正在测试中的节点

	// --- UI 交互状态 ---
	CursorGroup  int      // 当前光标在哪个组
	CursorNode   int      // 当前光标在哪个节点 (如果展开了组)
	Expanded     bool     // 当前组是否展开
	ExpandedList []string // 展开组里的节点列表缓存

	// --- 窗口尺寸 ---
	Width  int
	Height int
}

// NewModel 初始化模型
func NewModel(apiHost string) Model {
	return Model{
		apiBase:   apiHost,
		apiClient: client.New(apiHost), // 初始化 HTTP 客户端
		Proxies:   make(map[string]client.ProxyData),

		// --- 测速状态 ---
		Latencies:    make(map[string]int),
		TestingNodes: make(map[string]bool),

		// --- 默认状态 ---
		ProxyMode: "system",
		RouteMode: "rule",
		ConnState: ConnStateConnecting,
	}
}

// --- 2. 消息定义 (Msg) ---
// BubbleTea 是基于消息驱动的，我们需要定义几种消息类型

type trafficMsg TrafficStats // 接收到新流量数据的消息
type connMsg struct {        // 连接成功的消息
	conn *websocket.Conn
}
type errMsg error                           // 出错消息
type proxiesMsg map[string]client.ProxyData // 接收到新节点列表的消息
type nodeSwitchedMsg struct{}               // 节点切换完成的消息
type latencyMsg struct {                    // 测速结果消息
	Name  string
	Delay int // -1 代表失败
}

// 状态信息消息
type statusMsg struct {
	ProxyMode   string // 代理模式
	RouteMode   string // 路由模式
	Connections int    // 连接数
	Memory      uint64 // 内存
	TotalUp     int64  // 总上传流量
	TotalDown   int64  // 总下载流量
}

// 模式切换消息
type modeChangedMsg struct {
	Mode string
	Err  error
}
type routeChangedMsg struct {
	Mode string
	Err  error
}

// 重连相关消息
type reconnectMsg struct{}     // 触发重连
type reconnectFailMsg struct{} // 重连失败
