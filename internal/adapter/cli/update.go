package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/core/updater"
	"github.com/spf13/cobra"
)


func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update rules",
		Short: "Update geoip.db and geosite.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateRules()
		},
	}
}

func updateRules() error {
	// 1. 确定下载目录 (当前目录)
	// 也可以做的更高级：读取配置里的 WorkingDir
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	logger.Info("Updating rules...", "dir", dir)

	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()	

	// 下载GeoIP
	if err := updater.Download(ctx, updater.GeoIPURL, dir, updater.GeoIPFilename, printProgress("GeoIP")); err != nil {
		fmt.Println("GeoIP downloaded failed")
	}else {
		fmt.Println("GeoIP downloaded successfully")
	}

	// 下载GeoSite
	if err := updater.Download(ctx, updater.GeoSiteURL, dir, updater.GeoSiteFilename, printProgress("GeoSite")); err != nil {
		fmt.Println("GeoSite downloaded failed")
	}else {
		fmt.Println("GeoSite downloaded successfully")
	}

	return nil
}

func printProgress(name string) updater.ProgressCallback {
	return func(current, total int64) {
		// 简单的命令行覆写技巧：\r 回到行首
		if total > 0 {
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rDownloading %s: %.1f%% (%d/%d bytes)", name, percent, current, total)
		} else {
			fmt.Printf("\rDownloading %s: %d bytes", name, current)
		}
	}
}