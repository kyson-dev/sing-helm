package version_test

import (
	"testing"

	"github.com/kyson/minibox/internal/core/version"

	"github.com/stretchr/testify/assert"
)

func TestInfo_String(t *testing.T) {
	// Test Version info
	v := version.Info{}
	resut := v.String()

	assert.Contains(t, resut, "Proxy Client")
	assert.Contains(t, resut, "dev")
}
