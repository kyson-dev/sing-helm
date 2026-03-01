package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func RefreshSource(ctx context.Context, source Source, cacheDir string) (Cache, error) {
	if source.URL == "" {
		return Cache{}, fmt.Errorf("missing subscription url for %s", source.Name)
	}

	content, err := fetchURL(ctx, source.URL)
	if err != nil {
		return Cache{}, err
	}

	nodes, err := Parse(content, source.Format)
	if err != nil {
		return Cache{}, err
	}

	for i := range nodes {
		if nodes[i].Source == "" {
			nodes[i].Source = source.Name
		}
	}

	cache := Cache{
		Source:    source,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Nodes:     nodes,
	}

	if err := SaveCache(CacheFilePath(cacheDir, source.Name), cache); err != nil {
		return Cache{}, err
	}

	return cache, nil
}

func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sing-helm/1.0")

	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
