package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/subscription"
	"github.com/kyson-dev/sing-helm/internal/sys/env"
	"github.com/spf13/cobra"
)

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

func deleteAllSubscriptions(cmd *cobra.Command, dir, cacheDir string) error {
	sources, err := subscription.LoadSources(dir)
	if err != nil {
		return fmt.Errorf("failed to load sources: %w", err)
	}

	if len(sources) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No subscriptions found.")
		return nil
	}

	var failed []string
	for _, source := range sources {
		if err := deleteOneSubscription(cmd, source.Name, dir, cacheDir); err != nil {
			failed = append(failed, source.Name)
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to delete %s: %v\n", source.Name, err)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("delete failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func deleteOneSubscription(cmd *cobra.Command, name, dir, cacheDir string) error {
	configPath := subscription.SourceFilePath(dir, name)
	cachePath := subscription.CacheFilePath(cacheDir, name)

	// Check if exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("subscription not found: %s", name)
	}

	// Remove config
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	// Remove cache (ignore error if not exists)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to remove cache file for %s: %v\n", name, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted subscription: %s\n", name)
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
