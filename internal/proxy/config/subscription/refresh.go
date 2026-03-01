package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kyson-dev/sing-helm/internal/sys/logger"
)

// Refresh downloads a subscription and updates its cache
func Refresh(ctx context.Context, source Source, cacheDir string) error {
	logger.Info("Refreshing subscription", "name", source.Name, "url", source.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Some providers block standard go user agent
	req.Header.Set("User-Agent", "sing-box/1.10.x")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	nodes, err := Parse(content, source.Format)
	if err != nil {
		return fmt.Errorf("parse subscription failed: %w", err)
	}

	logger.Info("Successfully parsed nodes", "count", len(nodes))

	cache := Cache{
		Source:    source,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Nodes:     nodes,
	}

	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("create cache dir failed: %w", err)
	}

	cachePath := filepath.Join(cacheDir, source.Name+".json")
	return SaveCache(cachePath, cache)
}
