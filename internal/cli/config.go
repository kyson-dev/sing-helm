package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/subscription"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration files",
		RunE:  runConfigList,
	}

	cmd.AddCommand(
		newConfigListCommand(),
		newConfigAddCommand(),
		newConfigEditCommand(),
		newConfigRefreshCommand(),
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

func runConfigList(cmd *cobra.Command, _ []string) error {
	paths := env.Get()
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "Base config: %s\n", paths.ConfigFile)
	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		fmt.Fprintln(out, "  (missing)")
	}

	sources, err := subscription.LoadSources(paths.SubConfigDir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v\n", err)
	}

	fmt.Fprintln(out, "Subscriptions:")
	if len(sources) == 0 {
		fmt.Fprintln(out, "  (none)")
		return nil
	}

	for _, source := range sources {
		status := "enabled"
		if !source.EnabledValue() {
			status = "disabled"
		}
		cachePath := subscription.CacheFilePath(paths.SubCacheDir, source.Name)
		cache, err := subscription.LoadCache(cachePath)
		updated := "-"
		nodes := 0
		if err == nil {
			updated = cache.UpdatedAt
			nodes = len(cache.Nodes)
		}
		format := subscription.NormalizeFormat(source.Format)
		fmt.Fprintf(out, "- %s  %s  format=%s  nodes=%d  updated=%s  url=%s\n",
			source.Name, status, format, nodes, updated, source.URL)
	}

	return nil
}

func refreshAllSubscriptions(cmd *cobra.Command, dir, cacheDir string) error {
	sources, err := subscription.LoadSources(dir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v\n", err)
	}

	if len(sources) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No subscription configs found.")
		return nil
	}

	var failed []string
	for _, source := range sources {
		if !source.EnabledValue() {
			continue
		}
		if err := refreshSource(cmd, source, cacheDir); err != nil {
			failed = append(failed, source.Name)
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to refresh %s: %v\n", source.Name, err)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("refresh failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func refreshOneSubscription(cmd *cobra.Command, name, dir, cacheDir string) error {
	path := subscription.SourceFilePath(dir, name)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("subscription not found: %s", name)
		}
		return err
	}

	source, err := subscription.LoadSourceFile(path)
	if err != nil {
		return err
	}
	return refreshSource(cmd, source, cacheDir)
}

func refreshSource(cmd *cobra.Command, source subscription.Source, cacheDir string) error {
	cache, err := subscription.RefreshSource(cmd.Context(), source, cacheDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %s: %d nodes\n", source.Name, len(cache.Nodes))
	fmt.Fprintln(cmd.OutOrStdout(), "Restart sing-box to apply changes.")
	return nil
}

func openInEditor(cmd *cobra.Command, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(cmd.OutOrStdout(), "Configuration file not found: %s\n", path)
		fmt.Fprintln(cmd.OutOrStdout(), "It will be created when you save in the editor.")
	}

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Opening: %s\n", path)
	fmt.Fprintf(cmd.OutOrStdout(), "Editor:  %s\n\n", editor)

	editorArgs := strings.Fields(editor)
	editorCmd := exec.Command(editorArgs[0], append(editorArgs[1:], filepath.Clean(path))...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}
