package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Storage handles loading and saving subscription sources
func LoadSources(configDir string) ([]Source, error) {
	configPath := filepath.Join(configDir, "sources.yaml")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open sources.yaml failed: %w", err)
	}
	defer file.Close()

	var doc struct {
		Sources []Source `yaml:"sources"`
	}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode sources.yaml failed: %w", err)
	}

	// Set default values if not explicitly configured
	for i := range doc.Sources {
		doc.Sources[i].NormalizeDefaults(fmt.Sprintf("source-%d", i+1))
	}

	// Sort sources by priority descending (higher integer = higher priority)
	sort.SliceStable(doc.Sources, func(i, j int) bool {
		return doc.Sources[i].Priority > doc.Sources[j].Priority
	})

	return doc.Sources, nil
}

// SaveSources saves the given list of sources to the configuration directory
func SaveSources(configDir string, sources []Source) error {
	configPath := filepath.Join(configDir, "sources.yaml")

	doc := struct {
		Sources []Source `yaml:"sources"`
	}{
		Sources: sources,
	}

	data, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal sources.yaml failed: %w", err)
	}

	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("create config dir failed: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write sources.yaml failed: %w", err)
	}

	return nil
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
