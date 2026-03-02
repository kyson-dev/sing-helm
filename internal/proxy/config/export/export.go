package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// Target controls compatibility transforms for exported configs.
type Target struct {
	Version  string
	Platform string
}

// Export serializes options and applies compatibility transforms when needed.
func Export(opts *option.Options, target Target) ([]byte, error) {
	data, err := singboxjson.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Printf("DEBUG Raw JSON: %s\n", string(data)) // 检查这里的 JSON 是否有 

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	fmt.Printf("DEBUG Map Root: %+v\n", root) 

	// No transforms needed if no target specified
	if strings.TrimSpace(target.Version) == "" && strings.TrimSpace(target.Platform) == "" {
		return json.MarshalIndent(root, "", "  ")
	}

	// Apply version-specific compatibility transforms
	if strings.TrimSpace(target.Version) != "" {
		if err := applyVersionCompat(root, target.Version); err != nil {
			return nil, err
		}
	}

	// Apply platform-specific compatibility transforms
	if strings.TrimSpace(target.Platform) != "" {
		applyPlatformCompat(root, target.Platform)
	}

	return json.MarshalIndent(root, "", "  ")
}
