package monitor

// ============================================================================
// 连接状态定义（简化版）
// ============================================================================

// ConnState 连接状态
type ConnState int

const (
	ConnStateConnecting ConnState = iota // 正在连接（首次或重连）
	ConnStateConnected                   // 已连接
)

func (s ConnState) String() string {
	switch s {
	case ConnStateConnecting:
		return "Connecting"
	case ConnStateConnected:
		return "Connected"
	default:
		return "Unknown"
	}
}

// IsConnected 是否已连接
func (s ConnState) IsConnected() bool {
	return s == ConnStateConnected
}

// ============================================================================
// 连接状态机
// ============================================================================

// ConnectionStateMachine 连接状态机
type ConnectionStateMachine struct {
	State        ConnState
	ReconnectCnt int // 重连次数，0 表示首次连接
}

// OnConnected 连接成功
func (m *ConnectionStateMachine) OnConnected() {
	m.State = ConnStateConnected
}

// OnDisconnected 连接断开，自动进入重连状态
func (m *ConnectionStateMachine) OnDisconnected() {
	m.State = ConnStateConnecting
	m.ReconnectCnt++
}

// IsReconnecting 是否是重连（而非首次连接）
func (m *ConnectionStateMachine) IsReconnecting() bool {
	return m.ReconnectCnt > 0 && m.State == ConnStateConnecting
}
