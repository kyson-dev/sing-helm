package monitor

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
                if m.CursorNode == j { nodeCursor = "*" } // 节点光标
                
                // 高亮当前选中的节点
                active := ""
                if nodeName == groupData.Now { active = "(current)" }
                
                s += fmt.Sprintf("   %s %s %s\n", nodeCursor, nodeName, active)
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