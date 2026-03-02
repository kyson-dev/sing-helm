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

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

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
