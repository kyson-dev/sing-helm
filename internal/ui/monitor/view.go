package monitor

import (
	"fmt"
)

// View 渲染 UI 字符串
func (m Model) View() string {
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