package model

// Node is a normalized outbound entry representing a proxy node in a universal format.
type Node struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Source     string         `json:"source,omitempty"`
	SkipDedupe bool           `json:"-"`
	Outbound   map[string]any `json:"outbound"`
}
