package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/logger"
	"github.com/kyson-dev/sing-helm/internal/updater"
	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update rules",
		Short: "Update geoip.db and geosite.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := dispatchToDaemon(cmd.Context(), "update", nil); err != nil {
				if errors.Is(err, errDaemonUnavailable) {
					logger.Info("Daemon unavailable, updating locally")
					return updateRules()
				}
				return err
			}
			fmt.Println("Update job submitted to daemon; check logs for progress")
			return nil
		},
	}
}

func updateRules() error {
	dir := env.Get().AssetDir
	logger.Info("Updating rules...", "dir", dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := updater.Download(ctx, updater.GeoIPURL, dir, updater.GeoIPFilename, printProgress("GeoIP")); err != nil {
		fmt.Println("GeoIP downloaded failed")
	} else {
		fmt.Println("GeoIP downloaded successfully")
	}

	if err := updater.Download(ctx, updater.GeoSiteURL, dir, updater.GeoSiteFilename, printProgress("GeoSite")); err != nil {
		fmt.Println("GeoSite downloaded failed")
	} else {
		fmt.Println("GeoSite downloaded successfully")
	}

	return nil
}

func printProgress(name string) updater.ProgressCallback {
	return func(current, total int64) {
		if total > 0 {
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rDownloading %s: %.1f%% (%d/%d bytes)", name, percent, current, total)
			return
		}
		fmt.Printf("\rDownloading %s: %d bytes", name, current)
	}
}
