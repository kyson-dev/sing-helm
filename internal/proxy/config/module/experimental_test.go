package module

import (
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/sagernet/sing-box/option"
)

func TestExperimentalApply_InvalidExternalControllerFails(t *testing.T) {
	opts := &option.Options{
		Experimental: &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: "invalid-controller",
			},
		},
	}
	err := (&ExperimentalModule{}).Apply(opts, NewBuildContext(&model.RunOptions{}))
	if err == nil {
		t.Fatalf("expected parse error for invalid external_controller")
	}
}

func TestExperimentalApply_BackfillFromExternalController(t *testing.T) {
	run := &model.RunOptions{ListenAddr: "0.0.0.0", APIPort: 1}
	opts := &option.Options{
		Experimental: &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: "127.0.0.1:9090",
			},
		},
	}
	err := (&ExperimentalModule{ListenAddr: "10.0.0.1", APIPort: 9999}).Apply(opts, NewBuildContext(run))
	if err != nil {
		t.Fatalf("apply experimental: %v", err)
	}
	if run.ListenAddr != "127.0.0.1" || run.APIPort != 9090 {
		t.Fatalf("expected backfill from external_controller, got %+v", run)
	}
}
