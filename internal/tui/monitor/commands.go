package monitor

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/kysonzou/sing-helm/internal/client"
	"github.com/kysonzou/sing-helm/internal/controller"
)

// ============================================================================
// 异步命令定义
// 所有与外部系统交互的操作都封装为 tea.Cmd
// ============================================================================

// -----------------------------------------------------------------------------
// 1. 连接命令
// -----------------------------------------------------------------------------

// cmdConnect 建立 WebSocket 连接
func cmdConnect(host string) tea.Cmd {
	return func() tea.Msg {
		url := "ws://" + host + "/traffic?token="
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			return disconnectedMsg{err: err}
		}
		return connectedMsg{conn: conn}
	}
}

// cmdReconnectAfter 延迟重连
func cmdReconnectAfter(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return reconnectTickMsg{}
	})
}

// cmdReadTraffic 读取流量数据（阻塞）
func cmdReadTraffic(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		if conn == nil {
			return disconnectedMsg{err: nil}
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var stats struct {
			Up   int64 `json:"up"`
			Down int64 `json:"down"`
		}
		if err := conn.ReadJSON(&stats); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return disconnectedMsg{err: err}
			}
			return disconnectedMsg{err: err}
		}
		return trafficMsg{Up: stats.Up, Down: stats.Down}
	}
}

// -----------------------------------------------------------------------------
// 2. 数据获取命令
// -----------------------------------------------------------------------------

// cmdFetchStatus 获取状态信息
func cmdFetchStatus(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		// 从 sing-box API 获取连接信息
		conns, err := c.GetConnections()
		// 从 daemon 获取配置模式 + API 地址
		proxyMode := "unknown"
		routeMode := "unknown"
		apiBase := ""
		if status, err := controller.FetchStatus(context.Background()); err == nil {
			if status.ProxyMode != "" {
				proxyMode = status.ProxyMode
			}
			if status.RouteMode != "" {
				routeMode = status.RouteMode
			}
			apiBase = apiBaseFromStatus(status)
		}

		if err != nil {
			return statusMsg{
				ProxyMode: proxyMode,
				RouteMode: routeMode,
				APIBase:   apiBase,
				Err:       err,
			}
		}

		return statusMsg{
			ProxyMode:   proxyMode,
			RouteMode:   routeMode,
			Connections: len(conns.Connections),
			Memory:      conns.Memory,
			TotalUp:     conns.UploadTotal,
			TotalDown:   conns.DownloadTotal,
			APIBase:     apiBase,
		}
	}
}

// cmdFetchProxies 获取代理列表
func cmdFetchProxies(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		proxies, err := c.GetProxies()
		if err != nil {
			return proxiesMsg{Err: err}
		}
		return proxiesMsg{Proxies: proxies}
	}
}

// cmdTestLatency 测试节点延迟
func cmdTestLatency(c *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		delay, err := c.GetNodeDelay(name, "http://www.gstatic.com/generate_204", 2000)
		if err != nil {
			return latencyMsg{Name: name, Delay: -1}
		}
		return latencyMsg{Name: name, Delay: delay}
	}
}

// cmdStatusTick 状态定时刷新
func cmdStatusTick(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return statusTickMsg{}
	})
}

// -----------------------------------------------------------------------------
// 3. 模式切换命令
// -----------------------------------------------------------------------------

// cmdSwitchMode 切换代理模式
// 智能跳过 TUN（非 root 时）
func cmdSwitchMode(current string) tea.Cmd {
	return func() tea.Msg {
		// 计算下一个模式
		var next string
		switch strings.ToLower(current) {
		case "system":
			next = "tun" 
		case "tun":
			next = "default"
		case "default":
			next = "system"
		default:
			next = "system"
		}

		// 调用 daemon 切换
		_, err := controller.SwitchProxyMode(next)
		if err != nil {
			return modeChangedMsg{NewMode: current, Err: err}
			// TUN 权限错误时自动跳过
			// if strings.Contains(err.Error(), "permission") && next == "tun" {
			// 	next = "default"
			// 	_, err = controller.SwitchProxyMode(next)
			// }
			// if err != nil {
			// 	return modeChangedMsg{NewMode: current, Err: err}
			// }
		}

		return modeChangedMsg{NewMode: next, Err: nil}
	}
}

// cmdSwitchRoute 切换路由模式
func cmdSwitchRoute(current string) tea.Cmd {
	return func() tea.Msg {
		var next string
		switch strings.ToLower(current) {
		case "rule":
			next = "global"
		case "global":
			next = "direct"
		default:
			next = "rule"
		}

		_, err := controller.SwitchRouteMode(next)
		if err != nil {
			return routeChangedMsg{NewRoute: current, Err: err}
		}
		return routeChangedMsg{NewRoute: next, Err: nil}
	}
}

// cmdSwitchNode 切换节点
func cmdSwitchNode(c *client.Client, group, node string) tea.Cmd {
	return func() tea.Msg {
		err := c.SelectProxy(group, node)
		if err != nil {
			return nodeChangedMsg{Group: group, Node: node, Err: err}
		}
		return nodeChangedMsg{Group: group, Node: node, Err: nil}
	}
}

// -----------------------------------------------------------------------------
// 4. 工具函数
// -----------------------------------------------------------------------------

// extractGroups 从代理列表中提取可切换的组
func extractGroups(proxies map[string]client.ProxyData) []string {
	var groups []string
	for name, data := range proxies {
		if data.Type == "Selector" || data.Type == "URLTest" {
			groups = append(groups, name)
		}
	}

	// 排序：auto 放最后
	sort.Slice(groups, func(i, j int) bool {
		if groups[i] == "auto" {
			return false
		}
		if groups[j] == "auto" {
			return true
		}
		return groups[i] < groups[j]
	})

	return groups
}

func apiBaseFromStatus(status *controller.Status) string {
	if status == nil || status.APIPort == 0 {
		return ""
	}
	addr := status.ListenAddr
	if addr == "" {
		addr = "127.0.0.1"
	}
	return addr + ":" + strconv.Itoa(status.APIPort)
}
