package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadSources reads all .json subscription definitions from the config directory
func LoadSources(configDir string) ([]Source, error) {
	var sources []Source

	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read subscription config dir failed: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(configDir, entry.Name()))
		if err != nil {
			continue
		}

		var s Source
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}

		s.NormalizeDefaults(name)
		sources = append(sources, s)
	}

	// Sort sources by priority descending
	sort.SliceStable(sources, func(i, j int) bool {
		return sources[i].Priority > sources[j].Priority
	})

	return sources, nil
}

// SaveSource saves a single subscription source to its own .json file
func SaveSource(configDir string, source Source) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir failed: %w", err)
	}

	configPath := filepath.Join(configDir, source.Name+".json")
	data, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal subscription source failed: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// DeleteSource removes a subscription source's .json definition
func DeleteSource(configDir string, name string) error {
	configPath := filepath.Join(configDir, name+".json")
	return os.Remove(configPath)
}

// LoadCache loads parsed nodes strictly from cache file without verification
func LoadCache(cachePath string) (*Cache, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("read cache file failed: %w", err)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("unmarshal cache file failed: %w", err)
	}

	return &cache, nil
}

// SaveCache saves parsed nodes back into the cache
func SaveCache(cachePath string, cache Cache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache file failed: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("write cache file failed: %w", err)
	}

	return nil
}
