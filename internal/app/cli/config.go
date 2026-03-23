package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/subscription"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
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

			p := paths.Get()
			if err := os.MkdirAll(p.SubConfigDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			if err := os.MkdirAll(p.SubCacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache directory: %w", err)
			}

			sources, err := subscription.LoadSources(p.SubConfigDir)
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			for _, s := range sources {
				if s.Name == name {
					return fmt.Errorf("subscription already exists: %s", name)
				}
			}

			source := subscription.Source{
				Name:     name,
				URL:      url,
				Format:   subscription.NormalizeFormat(format),
				Priority: priority,
				Enabled:  &enabled,
				Dedupe:   &dedupe,
			}
			if err := subscription.SaveSource(p.SubConfigDir, source); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved: %s\n", filepath.Join(p.SubConfigDir, source.Name+".json"))
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
			p := paths.Get()
			target := p.ConfigFile
			if len(args) == 1 {
				if err := os.MkdirAll(p.SubConfigDir, 0755); err != nil {
					return fmt.Errorf("failed to create config dir: %w", err)
				}
				if err := os.MkdirAll(p.SubCacheDir, 0755); err != nil {
					return fmt.Errorf("failed to create cache dir: %w", err)
				}
				target = filepath.Join(p.SubConfigDir, args[0]+".json")
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
			p := paths.Get()
			if err := os.MkdirAll(p.SubConfigDir, 0755); err != nil {
				return fmt.Errorf("failed to create config dir: %w", err)
			}
			if err := os.MkdirAll(p.SubCacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache dir: %w", err)
			}

			if len(args) == 0 || strings.EqualFold(args[0], "all") {
				return refreshAllSubscriptions(cmd, p.SubConfigDir, p.SubCacheDir)
			}

			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			return refreshOneSubscription(cmd, name, p.SubConfigDir, p.SubCacheDir)
		},
	}
}

func newConfigDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name|all]",
		Short: "Delete a subscription config and cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := paths.Get()
			if err := os.MkdirAll(p.SubConfigDir, 0755); err != nil {
				return fmt.Errorf("failed to create config dir: %w", err)
			}
			if err := os.MkdirAll(p.SubCacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache dir: %w", err)
			}

			if strings.EqualFold(args[0], "all") {
				return deleteAllSubscriptions(cmd, p.SubConfigDir, p.SubCacheDir)
			}

			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			return deleteOneSubscription(cmd, name, p.SubConfigDir, p.SubCacheDir)
		},
	}
}
