package node

// Node is a normalized outbound entry representing a proxy node in a universal format.
type Node struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Source   string         `json:"source,omitempty"`
	Outbound map[string]any `json:"outbound"`
}
