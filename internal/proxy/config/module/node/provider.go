package node
import "github.com/kyson-dev/sing-helm/internal/proxy/config/model"

// NodeProvider is an interface for modules that provide proxy nodes.
// Examples: parsing from local user config, or reading from a subscription cache.
type NodeProvider interface {
	// Name returns the provider's logic name.
	Name() string
	// GetNodes fetches a list of normalized outbound nodes.
	GetNodes() ([]model.Node, error)
}
