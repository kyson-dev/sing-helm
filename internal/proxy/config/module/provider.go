package module

// Node contains the raw struct for outbound before sing-box validation
type Node struct {
	Name     string
	Type     string
	Source   string
	Outbound map[string]any
}

// NodeProvider provides a list of unbound proxy nodes
type NodeProvider interface {
	Name() string
	GetNodes() ([]Node, error)
}
