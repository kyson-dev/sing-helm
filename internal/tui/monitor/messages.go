package monitor

import (
	"github.com/gorilla/websocket"
	"github.com/kyson/minibox/internal/client"
)

// ============================================================================
// 消息定义
// BubbleTea 基于消息驱动，所有异步操作都通过消息通知状态变更
// ============================================================================

// -----------------------------------------------------------------------------
// 1. 连接相关消息
// -----------------------------------------------------------------------------

// connectedMsg WebSocket 连接成功
type connectedMsg struct {
	conn *websocket.Conn
}

// disconnectedMsg 连接断开（触发重连）
type disconnectedMsg struct {
	err error
}

// reconnectTickMsg 重连计时器触发
type reconnectTickMsg struct{}

// -----------------------------------------------------------------------------
// 2. 数据更新消息
// -----------------------------------------------------------------------------

// trafficMsg 实时流量数据
type trafficMsg struct {
	Up   int64
	Down int64
}

// statusMsg 状态信息（从 daemon 获取）
type statusMsg struct {
	ProxyMode   string
	RouteMode   string
	Connections int
	Memory      uint64
	TotalUp     int64
	TotalDown   int64
	APIBase     string
	Err         error
}

// statusTickMsg 触发状态轮询
type statusTickMsg struct{}

// proxiesMsg 代理节点列表
type proxiesMsg struct {
	Proxies map[string]client.ProxyData
	Err     error
}

// latencyMsg 节点延迟测试结果
type latencyMsg struct {
	Name  string
	Delay int // -1 表示失败/超时
}

// -----------------------------------------------------------------------------
// 3. 请求结果消息
// -----------------------------------------------------------------------------

// modeChangedMsg mode 切换完成
type modeChangedMsg struct {
	NewMode string
	Err     error
}

// routeChangedMsg route 切换完成
type routeChangedMsg struct {
	NewRoute string
	Err      error
}

// nodeChangedMsg node 切换完成
type nodeChangedMsg struct {
	Group string
	Node  string
	Err   error
}
