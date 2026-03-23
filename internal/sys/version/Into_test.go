package version_test

import (
	"testing"

	"github.com/kyson-dev/sing-helm/internal/sys/version"

	"github.com/stretchr/testify/assert"
)

func TestInfo_String(t *testing.T) {
	// Test Version info
	v := version.Info{}
	result := v.String()
	assert.Contains(t, result, "sing-helm")
	assert.Contains(t, result, version.Tag)
}
