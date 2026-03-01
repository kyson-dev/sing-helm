package adapter

import (
	"fmt"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
)

// ProtocolAdapter describes how to parse different node formats into a standard Node.
type ProtocolAdapter interface {
	FromClash(m map[string]any) (node.Node, error)
	FromURI(uri string) (node.Node, error)
}

var registry = make(map[string]ProtocolAdapter)

// Register registers an adapter for a protocol name (e.g., "vmess", "vless").
func Register(name string, adapter ProtocolAdapter) {
	registry[name] = adapter
}

// Get returns the protocol adapter.
func Get(name string) (ProtocolAdapter, error) {
	adapter, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported protocol: %s", name)
	}
	return adapter, nil
}

// Has checks if an adapter exists.
func Has(name string) bool {
	_, ok := registry[name]
	return ok
}
