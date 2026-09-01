package engine

import (
	"context"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/module"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

// This exercises the sing-box lifecycle, not just JSON decoding. In particular,
// it catches invalid DNS dialer detours that are only rejected during Box.Start.
func TestStart_LocalDoHTransport(t *testing.T) {
	opts := &option.Options{DNS: &option.DNSOptions{}}
	if err := (&module.DNSModule{}).Apply(opts, module.NewBuildContext(nil)); err != nil {
		t.Fatalf("apply DNS module: %v", err)
	}

	// Isolate local_dns so the test does not require a proxy outbound or make a
	// DNS request. Starting the transport is enough to validate its dialer.
	opts.DNS.Servers = opts.DNS.Servers[:1]
	opts.DNS.Rules = nil
	opts.DNS.Final = "local_dns"

	instance, err := box.New(box.Options{
		Context: include.Context(context.Background()),
		Options: *opts,
	})
	if err != nil {
		t.Fatalf("create sing-box with local DoH transport: %v", err)
	}
	defer instance.Close()
	if err := instance.Start(); err != nil {
		t.Fatalf("start sing-box with local DoH transport: %v", err)
	}
}
