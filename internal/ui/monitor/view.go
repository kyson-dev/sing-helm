package monitor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// 颜色定义 - 使用柔和色调
var (
	colorGreen   = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")) // 柔和绿
	colorYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")) // 柔和黄
	colorRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6E6E")) // 柔和红
	colorGray    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")) // 灰紫
	colorCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")) // 柔和青
	colorMagenta = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")) // 柔和紫
	colorWhite   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")) // 暖白
	colorDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A")) // 暗灰
	colorUpload  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")) // 上传用柔和青
	colorDown    = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")) // 下载用柔和绿
)

// 样式定义
var (
	// 主边框
	mainBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	// 标题
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	// 卡片样式
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1).
			Width(28)

	// 帮助栏
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	// 高亮键
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)
)

func (m Model) View() string {
	// 根据连接状态显示不同界面
	switch m.ConnState {
	case ConnStateConnecting:
		return renderConnecting()
	case ConnStateReconnecting:
		return renderReconnecting(m)
	case ConnStateError:
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.Err)
	}

	// 构建各部分
	header := renderHeader()

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

	// 代理节点面板（全宽）
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

// renderConnecting 连接中动画
func renderConnecting() string {
	return mainBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render(" Minibox Monitor "),
			"",
			colorCyan.Render("⟳ Connecting to Sing-box API..."),
			"",
			colorDim.Render("Press q to quit"),
		),
	)
}

// renderReconnecting 重连中界面
func renderReconnecting(m Model) string {
	return mainBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render(" Minibox Monitor "),
			"",
			colorYellow.Render("🔄 Reconnecting..."),
			"",
			colorDim.Render("Mode switching may cause temporary disconnect"),
			"",
			colorDim.Render("Press q to quit"),
		),
	)
}

// renderHeader 标题栏
func renderHeader() string {
	return titleStyle.Render(" 📡 Minibox Monitor ")
}

// renderStatusCard 状态卡片
func renderStatusCard(m Model) string {
	title := colorMagenta.Render("Status")

	modeLine := fmt.Sprintf("%s  %s",
		colorDim.Render("Mode:"),
		colorCyan.Render(fmt.Sprintf("[m] %s", m.ProxyMode)))

	routeLine := fmt.Sprintf("%s %s",
		colorDim.Render("Route:"),
		colorMagenta.Render(fmt.Sprintf("[r] %s", m.RouteMode)))

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
		colorCyan.Render(fmt.Sprintf("%d", m.Connections)))

	memLine := fmt.Sprintf("%-10s %s",
		colorDim.Render("Memory:"),
		colorWhite.Render(formatMemory(m.Memory)))

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

	upSpeed := formatBytes(m.Stats.Up)
	downSpeed := formatBytes(m.Stats.Down)

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

	upTotal := formatBytes(m.TotalUp)
	downTotal := formatBytes(m.TotalDown)

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

// formatMemory 格式化内存 (uint64 bytes)
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

// renderProxyPanel 代理节点面板
func renderProxyPanel(m Model) string {
	var lines []string
	lines = append(lines, colorMagenta.Render("  Proxies"))
	lines = append(lines, colorDim.Render("  "+strings.Repeat("─", 50)))

	if len(m.Groups) == 0 {
		lines = append(lines, colorDim.Render("  No proxy groups available"))
		return strings.Join(lines, "\n")
	}

	for i, groupName := range m.Groups {
		groupData := m.Proxies[groupName]

		// 组光标
		cursor := "  "
		if m.CursorGroup == i {
			cursor = colorMagenta.Render("▸ ")
		}

		// 组名和当前节点
		current := groupData.Now
		if current == "" {
			current = "-"
		}
		groupLine := fmt.Sprintf("%s%s %s",
			cursor,
			colorWhite.Render(groupName),
			colorDim.Render("["+current+"]"),
		)
		lines = append(lines, groupLine)

		// 展开的节点列表
		if m.Expanded && m.CursorGroup == i {
			for j, nodeName := range m.ExpandedList {
				isLast := j == len(m.ExpandedList)-1
				isCurrent := nodeName == groupData.Now
				isSelected := m.CursorNode == j

				// 树形连接符
				prefix := "    ├─ "
				if isLast {
					prefix = "    └─ "
				}

				// 节点图标：
				// - 当前节点：绿色实心 ●
				// - 选中节点：青色空心 ○
				// - 普通节点：灰色空心 ○
				icon := colorDim.Render("○")
				if isCurrent {
					icon = colorGreen.Render("●")
				} else if isSelected {
					icon = colorCyan.Render("◎") // 圆环样式，中间有标记
				}

				// 延迟
				latencyStr := renderLatency(m, nodeName)

				// 当前标记
				currentMark := ""
				if isCurrent {
					currentMark = colorGreen.Render(" ✓")
				}

				// 先 pad 名字到固定宽度，再上色（否则 ANSI 颜色码会破坏对齐）
				paddedName := fmt.Sprintf("%-22s", nodeName)

				// 节点名称：选中时亮白色，其他暗灰
				nodeNameStr := colorDim.Render(paddedName)
				if isSelected || isCurrent {
					nodeNameStr = colorWhite.Render(paddedName) // 选中用白色
				}

				// 组合
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

// renderLatency 渲染延迟（固定 8 字符宽度）
func renderLatency(m Model, name string) string {
	if m.TestingNodes[name] {
		return colorDim.Render("[... ] ")
	}

	// 对于 URLTest 类型的组（如 auto），显示其选中节点的延迟
	actualName := name
	if proxyData, exists := m.Proxies[name]; exists {
		if proxyData.Type == "URLTest" && proxyData.Now != "" {
			// 使用选中节点的延迟
			actualName = proxyData.Now
		}
	}

	delay, exists := m.Latencies[actualName]
	if !exists || delay == 0 {
		return colorDim.Render("[----] ")
	}

	if delay < 0 {
		return colorRed.Render("[FAIL] ")
	}

	// 固定宽度 6 字符 + 括号 = 8 字符
	val := fmt.Sprintf("[%4dms]", delay)

	if delay < 500 {
		return colorGreen.Render(val)
	} else if delay < 1000 {
		return colorYellow.Render(val)
	} else {
		return colorRed.Render(val)
	}
}

// renderHelpBar 帮助栏
func renderHelpBar() string {
	keys := []struct{ key, desc string }{
		{"↑↓", "Move"},
		{"←→", "Expand"},
		{"Enter", "Select"},
		{"t", "Test"},
		{"m", "Mode"},
		{"r", "Route"},
		{"q", "Quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %s",
			keyStyle.Render(k.key),
			helpStyle.Render(k.desc),
		))
	}

	return "  " + strings.Join(parts, "  │  ")
}

// formatBytes 格式化字节
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
