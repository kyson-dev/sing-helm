package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/subscription"
	"github.com/kyson-dev/sing-helm/internal/sys/env"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration files",
		Long: `Manage configuration files.

Available subcommands:
  list     - List base and subscription configs
  add      - Add a subscription config
  edit     - Edit base config or a subscription file
  refresh  - Refresh subscription cache`,
		// 不设置 RunE，让 cobra 在没有子命令时显示帮助
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// 启用命令建议（当输入错误时会提示相似的命令）
	cmd.SuggestionsMinimumDistance = 2

	cmd.AddCommand(
		newConfigListCommand(),
		newConfigAddCommand(),
		newConfigEditCommand(),
		newConfigRefreshCommand(),
		newConfigDeleteCommand(),
	)

	return cmd
}

func newConfigListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List base and subscription configs",
		RunE:  runConfigList,
	}
}

func newConfigAddCommand() *cobra.Command {
	var (
		format   string
		priority int
		enabled  bool
		dedupe   bool
	)
	cmd := &cobra.Command{
		Use:   "add [name] [url]",
		Short: "Add a subscription config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			url := strings.TrimSpace(args[1])
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			if strings.Contains(name, string(os.PathSeparator)) {
				return fmt.Errorf("name cannot contain path separators")
			}
			if url == "" {
				return fmt.Errorf("url cannot be empty")
			}

			paths := env.Get()
			if err := subscription.EnsureDirs(paths.SubConfigDir, paths.SubCacheDir); err != nil {
				return err
			}

			source := subscription.Source{
				Name:     name,
				URL:      url,
				Format:   format,
				Priority: priority,
				Enabled:  &enabled,
				Dedupe:   &dedupe,
			}

			path := subscription.SourceFilePath(paths.SubConfigDir, name)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("subscription already exists: %s", name)
			}
			if err := subscription.SaveSourceFile(path, source); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved: %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "auto", "Subscription format: auto, singbox, clash")
	cmd.Flags().IntVar(&priority, "priority", 0, "Priority for dedupe (higher wins)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable this subscription")
	cmd.Flags().BoolVar(&dedupe, "dedupe", true, "Enable dedupe for this subscription")
	return cmd
}

func newConfigEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit base config or a subscription file",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := env.Get()
			target := paths.ConfigFile
			if len(args) == 1 {
				if err := subscription.EnsureDirs(paths.SubConfigDir, paths.SubCacheDir); err != nil {
					return err
				}
				target = subscription.SourceFilePath(paths.SubConfigDir, strings.TrimSpace(args[0]))
			}
			return openInEditor(cmd, target)
		},
	}
}

func newConfigRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [name|all]",
		Short: "Refresh subscription cache",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := env.Get()
			if err := subscription.EnsureDirs(paths.SubConfigDir, paths.SubCacheDir); err != nil {
				return err
			}

			if len(args) == 0 || strings.EqualFold(args[0], "all") {
				return refreshAllSubscriptions(cmd, paths.SubConfigDir, paths.SubCacheDir)
			}

			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			return refreshOneSubscription(cmd, name, paths.SubConfigDir, paths.SubCacheDir)
		},
	}
}

func newConfigDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name|all]",
		Short: "Delete a subscription config and cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := env.Get()
			// 确保目录存在（虽然我们要删除东西，但如果目录都不存在也就没什么好删的，不过为了路径构建不出错）
			if err := subscription.EnsureDirs(paths.SubConfigDir, paths.SubCacheDir); err != nil {
				return err
			}

			if strings.EqualFold(args[0], "all") {
				return deleteAllSubscriptions(cmd, paths.SubConfigDir, paths.SubCacheDir)
			}

			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			return deleteOneSubscription(cmd, name, paths.SubConfigDir, paths.SubCacheDir)
		},
	}
}
