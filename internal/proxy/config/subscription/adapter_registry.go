package subscription

import "fmt"

// ProtocolAdapter describes how to parse different node formats into a standard Node.
type ProtocolAdapter interface {
	FromClash(m map[string]any) (Node, error)
	FromURI(uri string) (Node, error)
}

var registry = make(map[string]ProtocolAdapter)

// RegisterAdapter registers an adapter for a protocol name (e.g., "vmess", "vless").
func RegisterAdapter(name string, adapter ProtocolAdapter) {
	registry[name] = adapter
}

// GetAdapter returns the protocol adapter.
func GetAdapter(name string) (ProtocolAdapter, error) {
	adapter, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported protocol: %s", name)
	}
	return adapter, nil
}

// HasAdapter checks if an adapter exists.
func HasAdapter(name string) bool {
	_, ok := registry[name]
	return ok
}
