package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kyson/minibox/internal/config"
	"github.com/kyson/minibox/internal/runtime"
	"github.com/kyson/minibox/internal/env"
)

func TestBuildConfig(t *testing.T) {
	env.ResetForTest()
	dir := t.TempDir()
	if err := env.Init(dir); err != nil {
		t.Fatalf("env.Init failed: %v", err)
	}

	t.Setenv("MINIBOX_TEST_MIXED_PORT", "18081")
	t.Setenv("MINIBOX_TEST_API_PORT", "18082")

	profilePath := filepath.Join(dir, "profile.json")
	rawPath := filepath.Join(dir, "raw.json")
	if err := os.WriteFile(profilePath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write profile.json: %v", err)
	}

	runops := config.DefaultRunOptions()
	if err := runtime.BuildConfig(profilePath, rawPath, &runops); err != nil {
		t.Fatalf("BuildConfig failed: %v", err)
	}

	if runops.MixedPort != 18081 {
		t.Fatalf("expected mixed port override, got %d", runops.MixedPort)
	}
	if runops.APIPort != 18082 {
		t.Fatalf("expected api port override, got %d", runops.APIPort)
	}
	if _, err := os.Stat(rawPath); err != nil {
		t.Fatalf("expected raw config to exist: %v", err)
	}
}
