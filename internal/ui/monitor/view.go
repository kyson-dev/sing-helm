package monitor

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (

	colorGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	colorYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	colorRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	colorGray   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

func (m Model) View() string {
    // 1. 顶部：流量面板
    trafficView := renderTraffic(m)
    
    // 2. 底部：节点列表
    proxyView := renderProxyList(m)
    
    // 3. 垂直拼接
    return lipgloss.JoinVertical(lipgloss.Left, trafficView, proxyView)
}

func renderProxyList(m Model) string {
    s := "\nProxies (Select with ↑/↓/Enter):\n"
    
    for i, groupName := range m.Groups {
        cursor := " "
        if m.CursorGroup == i { cursor = ">" } // 组光标
        
        groupData := m.Proxies[groupName]
        // 显示： > ProxyGroup [当前节点]
        s += fmt.Sprintf("%s %s [%s]\n", cursor, groupName, groupData.Now)
        
        // 如果展开了当前组
        if m.Expanded && m.CursorGroup == i {
            for j, nodeName := range m.ExpandedList {
                nodeCursor := " "
				if m.CursorNode == j { nodeCursor = "*" }
				
				active := ""
				if nodeName == groupData.Now { active = "(current)" }
				
				// [新增] 渲染延迟
				latencyStr := renderLatency(m, nodeName)

				// 格式: * NodeName [120ms] (current)
				s += fmt.Sprintf("   %s %-30s %s %s\n", nodeCursor, nodeName, latencyStr, active)
            }
        }
    }
    return s
}

// View 渲染 UI 字符串
func renderTraffic(m Model) string {
	if m.Err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.Err)
	}

	if !m.connected {
		return "Connecting to Sing-box API...\n"
	}

	// 格式化流量
	upStr := formatBytes(m.Stats.Up)
	downStr := formatBytes(m.Stats.Down)

	// 构建界面内容
	content := fmt.Sprintf(
		"%s\n\n↑ Upload:   %s/s\n↓ Download: %s/s\n\n%s",
		titleStyle.Render(" Minibox Monitor "),
		upStr,
		downStr,
		infoStyle.Render("Press 'q' to quit"),
	)

	// 用边框包起来
	return boxStyle.Render(content)
}

// renderLatency 辅助函数：根据延迟返回带颜色的字符串
func renderLatency(m Model, name string) string {
	// 1. 检查是否正在测试
	if m.TestingNodes[name] {
		return colorGray.Render("[...]")
	}

	// 2. 检查是否有结果
	delay, exists := m.Latencies[name]
	if !exists || delay == 0 {
		return "" // 没测过就不显示
	}

	// 3. 渲染结果
	if delay < 0 {
		return colorRed.Render("[TIMEOUT]")
	}

	val := fmt.Sprintf("[%dms]", delay)
	
	if delay < 300 {
		return colorGreen.Render(val)
	} else if delay < 800 {
		return colorYellow.Render(val)
	} else {
		return colorRed.Render(val)
	}
}

// formatBytes 辅助函数
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