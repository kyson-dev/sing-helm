package monitor

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/core/client"
)

// ============================================================================
// Model 定义
// TUI 的核心状态容器，分为几个层次
// ============================================================================

// Model TUI 核心模型
type Model struct {
	// =========================================================================
	// 第一层：连接管理
	// =========================================================================
	connState ConnectionStateMachine // 连接状态机
	wsConn    *websocket.Conn        // WebSocket 连接
	apiBase   string                 // API 地址
	apiClient *client.Client         // HTTP 客户端
	lastError error                  // 最近的错误
	updating  bool                   // mode/route 更新中（防止重复请求）

	// 状态轮询控制
	statusInFlight bool
	statusInterval time.Duration
	reconnectWait  bool

	// =========================================================================
	// 第二层：业务数据
	// =========================================================================

	// --- 实时数据 ---
	traffic     TrafficData // 流量数据
	connections int         // 当前连接数
	memory      uint64      // 内存使用

	// --- 配置状态 ---
	proxyMode string // 代理模式: system, tun, default
	routeMode string // 路由模式: rule, global, direct

	// --- 节点列表 ---
	groups    []string                    // 代理组列表
	proxies   map[string]client.ProxyData // 代理详情
	latencies map[string]int              // 节点延迟 (-1=失败, 0=未测试)
	testing   map[string]bool             // 正在测速的节点

	// =========================================================================
	// 第三层：UI 交互状态
	// =========================================================================
	cursor       CursorState // 光标状态
	expanded     bool        // 是否展开节点列表
	expandedList []string    // 展开的节点列表缓存
}

// TrafficData 流量数据
type TrafficData struct {
	Up        int64 // 当前上传速度
	Down      int64 // 当前下载速度
	TotalUp   int64 // 累计上传
	TotalDown int64 // 累计下载
}

// CursorState 光标状态
type CursorState struct {
	Group int // 当前组索引
	Node  int // 当前节点索引（展开时有效）
}

// NewModel 创建新的 Model
func NewModel(apiHost string) Model {
	return Model{
		// 连接管理
		apiBase:   apiHost,
		apiClient: client.New(apiHost),
		connState: ConnectionStateMachine{State: ConnStateConnecting},
		statusInterval: time.Second,

		// 业务数据初始化
		proxyMode: "unknown",
		routeMode: "unknown",
		proxies:   make(map[string]client.ProxyData),
		latencies: make(map[string]int),
		testing:   make(map[string]bool),
	}
}

// ============================================================================
// Model 访问器（只读）
// ============================================================================

// IsConnected 是否已连接
func (m *Model) IsConnected() bool {
	return m.connState.State.IsConnected()
}

// ConnState 获取连接状态
func (m *Model) ConnState() ConnState {
	return m.connState.State
}

// IsUpdating 是否正在更新 mode/route（使用全局锁）
func (m *Model) IsUpdating() bool {
	return m.updating
}

// ProxyMode 获取代理模式
func (m *Model) ProxyMode() string {
	return m.proxyMode
}

// RouteMode 获取路由模式
func (m *Model) RouteMode() string {
	return m.routeMode
}

// Traffic 获取流量数据
func (m *Model) Traffic() TrafficData {
	return m.traffic
}

// Groups 获取代理组列表
func (m *Model) Groups() []string {
	return m.groups
}

// Proxies 获取代理详情
func (m *Model) Proxies() map[string]client.ProxyData {
	return m.proxies
}

// Latency 获取节点延迟
func (m *Model) Latency(name string) (int, bool) {
	delay, ok := m.latencies[name]
	return delay, ok
}

// IsTesting 节点是否正在测速
func (m *Model) IsTesting(name string) bool {
	return m.testing[name]
}

// Cursor 获取光标状态
func (m *Model) Cursor() CursorState {
	return m.cursor
}

// IsExpanded 是否展开
func (m *Model) IsExpanded() bool {
	return m.expanded
}

// ExpandedList 获取展开的节点列表
func (m *Model) ExpandedList() []string {
	return m.expandedList
}

// Connections 获取连接数
func (m *Model) Connections() int {
	return m.connections
}

// Memory 获取内存使用
func (m *Model) Memory() uint64 {
	return m.memory
}

// LastError 获取最近的错误
func (m *Model) LastError() error {
	return m.lastError
}
