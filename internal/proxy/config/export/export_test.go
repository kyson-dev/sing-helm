package export

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestExport_CurrentSchema(t *testing.T) {
	opts := &option.Options{}

	if _, err := Export(opts, Target{}); err != nil {
		t.Fatalf("current schema export failed: %v", err)
	}
}
