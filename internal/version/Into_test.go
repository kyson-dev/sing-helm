package version_test

import (
	"testing"

	"github.com/kysonzou/sing-helm/internal/version"

	"github.com/stretchr/testify/assert"
)

func TestInfo_String(t *testing.T) {
	// Test Version info
	v := version.Info{}
	resut := v.String()

	assert.Contains(t, resut, "sing-helm")
	assert.Contains(t, resut, "dev")
}
