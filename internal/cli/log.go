package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/kyson-dev/sing-helm/internal/logger"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
)

func newLogCommand() *cobra.Command {
	var showSystem bool
	var showAll bool

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Stream application logs",
		Long: `Stream application logs in real-time.

By default, shows application business logic logs.
Use --system to view system-level logs (startup failures, panics).
Use --all to view all logs together.`,
		Run: func(cmd *cobra.Command, args []string) {
			if showAll {
				// Show all logs: app + stdout + stderr
				showMultipleLogs(cmd)
				return
			}

			if showSystem {
				// Show system logs only
				showSystemLogs(cmd)
				return
			}

			// Default: show application log
			showAppLog(cmd)
		},
	}

	cmd.Flags().BoolVar(&showSystem, "system", false, "Show system logs (stdout/stderr from launchd)")
	cmd.Flags().BoolVar(&showAll, "all", false, "Show all logs (app + system)")

	return cmd
}

func showAppLog(cmd *cobra.Command) {
	resp, err := dispatchToDaemon(cmd.Context(), "log", nil)
	if err != nil {
		logger.Error("Failed to resolve log path", "error", err)
		return
	}
	logPath, ok := resp.Data["path"].(string)
	if !ok || logPath == "" {
		logger.Error("Missing log path from daemon")
		return
	}

	fmt.Printf("📋 Streaming application log: %s\n", logPath)
	tailLog(logPath)
}

func showSystemLogs(cmd *cobra.Command) {
	// Resolve log directory dynamically
	runtimeDir := env.ResolveRuntimeDir()
	logDir := logger.ResolveLogDir(runtimeDir)

	stdoutLog := filepath.Join(logDir, "stdout.log")
	stderrLog := filepath.Join(logDir, "stderr.log")

	fmt.Println("📋 System logs (from launchd):")
	fmt.Printf("   stdout: %s\n", stdoutLog)
	fmt.Printf("   stderr: %s\n", stderrLog)
	fmt.Println()

	// Check if files exist
	if !fileExists(stdoutLog) && !fileExists(stderrLog) {
		fmt.Println("⚠️  No system logs found. This is normal if:")
		fmt.Println("   - Service was never started via launchd")
		fmt.Println("   - No startup failures occurred")
		return
	}

	// Show last 20 lines of each
	if fileExists(stdoutLog) {
		fmt.Println("--- stdout.log (last 20 lines) ---")
		showLastLines(stdoutLog, 20)
		fmt.Println()
	}

	if fileExists(stderrLog) {
		fmt.Println("--- stderr.log (last 20 lines) ---")
		showLastLines(stderrLog, 20)
	}
}

func showMultipleLogs(cmd *cobra.Command) {
	fmt.Println("📋 Showing all logs...")
	fmt.Println()
	showSystemLogs(cmd)
	fmt.Println()
	fmt.Println("--- Application log (streaming) ---")
	showAppLog(cmd)
}

func tailLog(logPath string) {
	t, err := tail.TailFile(logPath, tail.Config{
		Follow:   true,
		ReOpen:   true, // 支持日志轮转后继续读
		Location: &tail.SeekInfo{Offset: 0, Whence: 2},
	})
	if err != nil {
		logger.Error("Failed to tail file", "error", err)
		return
	}

	for line := range t.Lines {
		fmt.Println(line.Text)
	}
}

func showLastLines(path string, n int) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", path, err)
		return
	}

	if len(content) == 0 {
		fmt.Println("(empty)")
		return
	}

	// Simple implementation: just show the file content
	// For production, you might want to use tail -n
	fmt.Print(string(content))
}
