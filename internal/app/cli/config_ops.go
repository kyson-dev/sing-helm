package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/subscription"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
	"github.com/spf13/cobra"
)

func runConfigList(cmd *cobra.Command, args []string) error { //nolint:unparam
	paths := paths.Get()
	fmt.Fprintf(cmd.OutOrStdout(), "Base Config: %s\n", paths.ConfigFile)

	sources, _ := subscription.LoadSources(paths.SubConfigDir)
	if len(sources) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nNo subscriptions found.\n")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nSubscriptions (%d):\n", len(sources))
	for _, source := range sources {
		status := "enabled"
		if !source.EnabledValue() {
			status = "disabled"
		}
		var cacheInfo string
		cachePath := filepath.Join(paths.SubCacheDir, source.Name+".json")
		if cache, err := subscription.LoadCache(cachePath); err == nil {
			cacheInfo = fmt.Sprintf("%d nodes, updated: %s", len(cache.Nodes), cache.UpdatedAt)
		} else {
			cacheInfo = "not cached"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s, P%d): %s [%s]\n",
			source.Name, status, source.Priority, source.URL, cacheInfo)
	}

	return nil
}

func openInEditor(cmd *cobra.Command, path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // Default to vim if EDITOR not set
	}

	// Simple editor opener
	if strings.HasSuffix(path, ".json") {
		// allow edit json files
	}

	execCmd := exec.Command(editor, path)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor %s: %w", editor, err)
	}

	return nil
}

func refreshAllSubscriptions(cmd *cobra.Command, configDir, cacheDir string) error {
	sources, err := subscription.LoadSources(configDir)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No subscriptions to refresh.\n")
		return nil
	}

	for _, source := range sources {
		if !source.EnabledValue() {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipping disabled subscription: %s\n", source.Name)
			continue
		}
		if err := subscription.Refresh(context.Background(), source, cacheDir); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to refresh %s: %v\n", source.Name, err)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Refreshed: %s\n", source.Name)
	}
	return nil
}

func refreshOneSubscription(cmd *cobra.Command, name, configDir, cacheDir string) error {
	sources, err := subscription.LoadSources(configDir)
	if err != nil {
		return err
	}
	var targetSource *subscription.Source
	for _, s := range sources {
		if s.Name == name {
			targetSource = &s
			break
		}
	}
	if targetSource == nil {
		return fmt.Errorf("subscription not found: %s", name)
	}

	if err := subscription.Refresh(context.Background(), *targetSource, cacheDir); err != nil {
		return fmt.Errorf("failed to refresh %s: %w", name, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Refreshed: %s\n", name)
	return nil
}

func deleteAllSubscriptions(cmd *cobra.Command, configDir, cacheDir string) error {
	sources, _ := subscription.LoadSources(configDir)
	for _, s := range sources {
		_ = subscription.DeleteSource(configDir, s.Name)
	}
	_ = os.RemoveAll(cacheDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted all subscriptions.\n")
	return nil
}

func deleteOneSubscription(cmd *cobra.Command, name, configDir, cacheDir string) error {
	sources, err := subscription.LoadSources(configDir)
	if err != nil { // Ignore not exist
		return err
	}

	var newSources []subscription.Source
	found := false
	for _, s := range sources {
		if s.Name == name {
			found = true
			continue
		}
		newSources = append(newSources, s)
	}

	if !found {
		return fmt.Errorf("subscription not found: %s", name)
	}

	_ = subscription.DeleteSource(configDir, name)

	// Delete cache
	_ = os.Remove(filepath.Join(cacheDir, name+".json"))

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted subscription: %s\n", name)
	return nil
}
