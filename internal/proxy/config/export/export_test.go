package export

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestExport_OnlySupportsTwoTargetVersions(t *testing.T) {
	opts := &option.Options{}

	if _, err := Export(opts, Target{Version: "latest"}); err != nil {
		t.Fatalf("latest should be supported: %v", err)
	}
	if _, err := Export(opts, Target{Version: "1.11.4"}); err != nil {
		t.Fatalf("1.11.4 should be supported: %v", err)
	}
	if _, err := Export(opts, Target{Version: "1.12.3"}); err == nil {
		t.Fatalf("unexpected success for unsupported version")
	}
}
