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

	version := strings.ToLower(strings.TrimSpace(target.Version))
	switch version {
	case "", "latest":
		// Latest uses current schema directly.
	case "1.11.4":
		applyCompatForV1114(root)
	default:
		return nil, fmt.Errorf("unsupported target version %q, only supports: 1.11.4, latest", target.Version)
	}

	// Apply platform-specific compatibility transforms
	if strings.TrimSpace(target.Platform) != "" {
		applyPlatformCompat(root, target.Platform)
	}

	return json.MarshalIndent(root, "", "  ")
}
