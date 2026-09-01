package export

import (
	"encoding/json"
	"fmt"

	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// Target controls platform-specific transforms for exported configs.
type Target struct {
	Platform string
}

// Export serializes options using the current sing-box schema and applies
// platform-specific transforms when needed.
func Export(opts *option.Options, target Target) ([]byte, error) {
	data, err := singboxjson.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyPlatformCompat(root, target.Platform)

	return json.MarshalIndent(root, "", "  ")
}
