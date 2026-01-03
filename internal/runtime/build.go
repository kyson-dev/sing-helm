package runtime

import (
	"fmt"

	"github.com/kyson/minibox/internal/config"
)

// BuildConfig loads the profile, applies runtime modules, and saves raw config.
func BuildConfig(profilePath, rawPath string, runops *config.RunOptions) error {
	base, err := config.LoadProfile(profilePath)
	if err != nil {
		return fmt.Errorf("failed to load profile file: %w", err)
	}

	builder := config.NewConfigBuilder(base, runops)
	for _, m := range config.DefaultModules(runops) {
		builder.With(m)
	}

	if err := builder.SaveToFile(rawPath); err != nil {
		return fmt.Errorf("failed to save raw config: %w", err)
	}

	return nil
}
