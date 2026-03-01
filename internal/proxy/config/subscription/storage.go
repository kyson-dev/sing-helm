package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func EnsureDirs(configDir, cacheDir string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.MkdirAll(cacheDir, 0755)
}

func SourceFilePath(dir, name string) string {
	return filepath.Join(dir, name+".json")
}

func CacheFilePath(dir, name string) string {
	return filepath.Join(dir, name+".json")
}

func LoadSources(dir string) ([]Source, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Source{}, nil
		}
		return nil, err
	}

	var sources []Source
	var loadErrs []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		source, err := LoadSourceFile(path)
		if err != nil {
			loadErrs = append(loadErrs, err)
			continue
		}
		sources = append(sources, source)
	}

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Priority == sources[j].Priority {
			return sources[i].Name < sources[j].Name
		}
		return sources[i].Priority > sources[j].Priority
	})

	if len(loadErrs) > 0 {
		return sources, fmt.Errorf("failed to load %d subscription file(s)", len(loadErrs))
	}
	return sources, nil
}

func LoadSourceFile(path string) (Source, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}

	var source Source
	if err := json.Unmarshal(content, &source); err != nil {
		return Source{}, fmt.Errorf("invalid subscription file %s: %w", path, err)
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	source.NormalizeDefaults(name)
	if source.Name != name {
		source.Name = name
	}

	return source, nil
}

func SaveSourceFile(path string, source Source) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	source.NormalizeDefaults(name)
	source.Name = name

	data, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func LoadCache(path string) (Cache, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Cache{}, err
	}

	var cache Cache
	if err := json.Unmarshal(content, &cache); err != nil {
		return Cache{}, fmt.Errorf("invalid cache file %s: %w", path, err)
	}

	return cache, nil
}

func SaveCache(path string, cache Cache) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
