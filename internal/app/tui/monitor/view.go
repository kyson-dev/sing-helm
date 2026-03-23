package monitor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// 视图渲染
// ============================================================================

// 颜色定义 - 使用柔和色调
var (
	colorGreen   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#008000", Dark: "#50FA7B"}) // 深绿/柔和绿
	colorYellow  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B08800", Dark: "#F1FA8C"}) // 深黄/柔和黄
	colorRed     = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C00000", Dark: "#FF6E6E"}) // 深红/柔和红
	colorGray    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#6272A4"}) // 深灰/灰紫
	colorCyan    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#005F87", Dark: "#8BE9FD"}) // 深青/柔和青
	colorMagenta = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8700AF", Dark: "#BD93F9"}) // 深紫/柔和紫
	colorWhite   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#F8F8F2"}) // 深黑/暖白
	colorDim     = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#44475A"}) // 浅灰/暗灰
	colorUpload  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#005F87", Dark: "#8BE9FD"}) // 上传使用青色
	colorDown    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#008000", Dark: "#50FA7B"}) // 下载使用绿色
)

// 样式定义
var (
	mainBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1).
			Width(28)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#626262"})

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)
)

// View BubbleTea 视图接口
func (m Model) View() string {
	// 根据连接状态显示不同界面
	switch m.ConnState() {
	case ConnStateConnecting:
		// 区分首次连接和重连
		if m.connState.IsReconnecting() {
			return renderReconnecting()
		}
		return renderConnecting()
	}

	// 已连接状态：显示完整界面
	header := renderHeader(m)

	// 左列：Status + Traffic
	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		renderStatusCard(m),
		"",
		renderTrafficCard(m),
	)

	// 右列：Connections + Traffic Total
	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		renderConnectionsCard(m),
		"",
		renderTrafficTotalCard(m),
	)

	// 左右拼接
	cards := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)

	// 代理节点面板
	proxies := renderProxyPanel(m)
	help := renderHelpBar()

	// 最终拼接
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		cards,
		"",
		proxies,
		"",
		help,
	)

	return mainBoxStyle.Render(content)
}

// renderConnecting 连接中界面
func renderConnecting() string {
	return mainBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render(" SingHelm Monitor "),
			"",
			colorCyan.Render("⟳ Connecting to Sing-box API..."),
			"",
			colorDim.Render("Press q to quit"),
		),
	)
}

// renderReconnecting 重连中界面
func renderReconnecting() string {
	return mainBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render(" SingHelm Monitor "),
			"",
			colorYellow.Render("🔄 Reconnecting..."),
			"",
			colorDim.Render("Mode switching may cause temporary disconnect"),
			"",
			colorDim.Render("Press q to quit"),
		),
	)
}

// renderHeader 标题栏（带状态指示器）
func renderHeader(m Model) string {
	title := " 📡 SingHelm Monitor "
	status := renderStatusIndicator(m)

	titlePart := titleStyle.Render(title)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		titlePart,
		" ",
		status,
	)

	return headerLine
}

// renderStatusIndicator 渲染状态指示器
func renderStatusIndicator(m Model) string {
	var dot, label string
	var dotStyle lipgloss.Style

	switch m.ConnState() {
	case ConnStateConnecting:
		if m.connState.IsReconnecting() {
			dotStyle = colorYellow
			label = "Reconnecting"
		} else {
			dotStyle = colorYellow
			label = "Connecting"
		}
	case ConnStateConnected:
		if m.IsUpdating() {
			dotStyle = colorCyan
			label = "Updating"
		} else {
			dotStyle = colorGreen
			label = "Connected"
		}
	default:
		dotStyle = colorGray
		label = "Unknown"
	}

	dot = "⏺"
	return dotStyle.Render(dot) + " " + colorDim.Render(label)
}

// renderStatusCard 状态卡片
func renderStatusCard(m Model) string {
	title := colorMagenta.Render("Status")

	modeLine := fmt.Sprintf("%s  %s",
		colorDim.Render("Mode:"),
		colorCyan.Render(fmt.Sprintf("[m] %s", m.ProxyMode())))

	routeLine := fmt.Sprintf("%s %s",
		colorDim.Render("Route:"),
		colorMagenta.Render(fmt.Sprintf("[r] %s", m.RouteMode())))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		colorDim.Render(strings.Repeat("─", 26)),
		modeLine,
		routeLine,
	)

	return cardStyle.Render(content)
}

// renderConnectionsCard 连接卡片
func renderConnectionsCard(m Model) string {
	title := colorMagenta.Render("Connections")

	totalLine := fmt.Sprintf("%-10s %s",
		colorDim.Render("Total:"),
		colorCyan.Render(fmt.Sprintf("%d", m.Connections())))

	memLine := fmt.Sprintf("%-10s %s",
		colorDim.Render("Memory:"),
		colorWhite.Render(formatMemory(m.Memory())))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		colorDim.Render(strings.Repeat("─", 26)),
		totalLine,
		memLine,
	)

	return cardStyle.Render(content)
}

// renderTrafficCard 流量卡片（当前速度）
func renderTrafficCard(m Model) string {
	title := colorMagenta.Render("Traffic")
	traffic := m.Traffic()

	upSpeed := formatBytes(traffic.Up)
	downSpeed := formatBytes(traffic.Down)

	upLine := fmt.Sprintf("%-10s %s",
		colorUpload.Render("Uplink:"),
		colorWhite.Render(upSpeed+"/s"))

	downLine := fmt.Sprintf("%-10s %s",
		colorDown.Render("Downlink:"),
		colorWhite.Render(downSpeed+"/s"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		colorDim.Render(strings.Repeat("─", 26)),
		upLine,
		downLine,
	)

	return cardStyle.Render(content)
}

// renderTrafficTotalCard 流量总计卡片
func renderTrafficTotalCard(m Model) string {
	title := colorMagenta.Render("Traffic Total")
	traffic := m.Traffic()

	upTotal := formatBytes(traffic.TotalUp)
	downTotal := formatBytes(traffic.TotalDown)

	upLine := fmt.Sprintf("%-10s %s",
		colorUpload.Render("Uplink:"),
		colorWhite.Render(upTotal))

	downLine := fmt.Sprintf("%-10s %s",
		colorDown.Render("Downlink:"),
		colorWhite.Render(downTotal))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		colorDim.Render(strings.Repeat("─", 26)),
		upLine,
		downLine,
	)

	return cardStyle.Render(content)
}

// formatMemory 格式化内存
func formatMemory(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatBytes 格式化字节
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// renderProxyPanel 代理节点面板
func renderProxyPanel(m Model) string {
	var lines []string
	lines = append(lines, colorMagenta.Render("  Proxies"))
	lines = append(lines, colorDim.Render("  "+strings.Repeat("─", 50)))

	groups := m.Groups()
	proxies := m.Proxies()

	if len(groups) == 0 {
		lines = append(lines, colorDim.Render("  No proxy groups available"))
		return strings.Join(lines, "\n")
	}

	cursor := m.Cursor()

	for i, groupName := range groups {
		groupData := proxies[groupName]

		// 组光标
		groupCursor := "  "
		if cursor.Group == i {
			groupCursor = colorMagenta.Render("▸ ")
		}

		// 组名和当前节点
		current := groupData.Now
		if current == "" {
			current = "-"
		}
		groupLine := fmt.Sprintf("%s%s %s",
			groupCursor,
			colorWhite.Render(groupName),
			colorDim.Render("["+current+"]"),
		)
		lines = append(lines, groupLine)

		// 展开的节点列表
		if m.IsExpanded() && cursor.Group == i {
			expandedList := m.ExpandedList()
			for j, nodeName := range expandedList {
				isLast := j == len(expandedList)-1
				isCurrent := nodeName == groupData.Now
				isSelected := cursor.Node == j

				// 树形连接符
				prefix := "    ├─ "
				if isLast {
					prefix = "    └─ "
				}

				// 节点图标
				icon := colorDim.Render("○")
				if isCurrent {
					icon = colorGreen.Render("●")
				} else if isSelected {
					icon = colorCyan.Render("◎")
				}

				// 延迟
				latencyStr := renderLatency(m, nodeName)

				// 当前标记
				currentMark := ""
				if isCurrent {
					currentMark = colorGreen.Render(" ✓")
				}

				// 节点名称
				paddedName := fmt.Sprintf("%-22s", nodeName)
				nodeNameStr := colorDim.Render(paddedName)
				if isSelected || isCurrent {
					nodeNameStr = colorWhite.Render(paddedName)
				}

				nodeLine := fmt.Sprintf("%s%s %s %s%s",
					colorDim.Render(prefix),
					icon,
					nodeNameStr,
					latencyStr,
					currentMark,
				)
				lines = append(lines, nodeLine)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// renderLatency 渲染延迟
func renderLatency(m Model, name string) string {
	if m.IsTesting(name) {
		return colorDim.Render("[... ] ")
	}

	// 对于 URLTest 类型的组，显示其选中节点的延迟
	actualName := name
	proxies := m.Proxies()
	if proxyData, exists := proxies[name]; exists {
		if proxyData.Type == "URLTest" && proxyData.Now != "" {
			actualName = proxyData.Now
		}
	}

	delay, exists := m.Latency(actualName)
	if !exists || delay == 0 {
		return colorDim.Render("[----] ")
	}

	if delay < 0 {
		return colorRed.Render("[FAIL] ")
	}

	// 根据延迟着色
	delayStr := fmt.Sprintf("[%4d]", delay)
	switch {
	case delay < 500:
		return colorGreen.Render(delayStr) + " "
	case delay < 1000:
		return colorYellow.Render(delayStr) + " "
	default:
		return colorRed.Render(delayStr) + " "
	}
}

// renderHelpBar 帮助栏
func renderHelpBar() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"↑↓", "navigate"},
		{"←→", "collapse/expand"},
		{"Enter", "select"},
		{"t", "test"},
		{"m", "mode"},
		{"r", "route"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+" "+helpStyle.Render(k.desc))
	}

	return helpStyle.Render("  ") + strings.Join(parts, helpStyle.Render("  •  "))
}
