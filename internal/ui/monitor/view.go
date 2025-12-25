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

	// 节点列表区域
	proxyBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)

	// 帮助栏
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	// 高亮键
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)
)

func (m Model) View() string {
	if m.Err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.Err)
	}

	if !m.connected {
		return renderConnecting()
	}

	// 构建各部分
	header := renderHeader()
	traffic := renderTrafficPanel(m)
	proxies := renderProxyPanel(m)
	help := renderHelpBar()

	// 拼接
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		traffic,
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

// renderHeader 标题栏
func renderHeader() string {
	return titleStyle.Render(" 📡 Minibox Monitor ")
}

// renderTrafficPanel 流量面板
func renderTrafficPanel(m Model) string {
	upSpeed := formatBytes(m.Stats.Up)
	downSpeed := formatBytes(m.Stats.Down)
	upTotal := formatBytes(m.TotalUp)
	downTotal := formatBytes(m.TotalDown)

	// 使用表格式布局
	upLine := fmt.Sprintf("  %s %-12s %s",
		colorUpload.Render("▲ Upload:"),
		colorWhite.Render(upSpeed+"/s"),
		colorDim.Render("Total: "+upTotal),
	)

	downLine := fmt.Sprintf("  %s %-12s %s",
		colorDown.Render("▼ Download:"),
		colorWhite.Render(downSpeed+"/s"),
		colorDim.Render("Total: "+downTotal),
	)

	return lipgloss.JoinVertical(lipgloss.Left, upLine, downLine)
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

	delay, exists := m.Latencies[name]
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
		{"←", "Collapse"},
		{"→/Enter", "Expand"},
		{"Space", "Select"},
		{"t", "Test"},
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
