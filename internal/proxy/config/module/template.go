package module

import (
	"context"
	"encoding/json"
	"os"

	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// TemplateModule is responsible for loading the user's custom profile.json
// and setting it as the structural foundation of the configuration options.
type TemplateModule struct{}

func (m *TemplateModule) Name() string {
	return "template"
}

func (m *TemplateModule) Apply(opts *option.Options, ctx *BuildContext) error {
	profilePath := paths.Get().ConfigFile

	data, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("No profile.json found, skipping template injection")
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	// Read into a map first to selectively strip fields if necessary.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		logger.Error("Failed to parse profile.json", "error", err)
		return nil // Non-fatal, fallback to generated
	}

	// We KEEP "outbounds" so users can define extra custom groups in profile.json.
	// The OutboundModule will append the actual proxy nodes and system groups later.

	// Convert back to bytes for unmarshaling into option.Options with context
	cleanData, err := singboxjson.Marshal(raw)
	if err != nil {
		return err
	}

	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, cleanData, opts); err != nil {
		logger.Error("Failed to unmarshal profile.json into Sing-box options", "error", err)
		return nil // Non-fatal
	}

	logger.Info("Injected user profile template", "path", profilePath)
	return nil
}
