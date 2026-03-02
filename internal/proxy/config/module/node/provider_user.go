package node

import (
	"encoding/json"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// UserNodeProvider extracts actual proxy nodes from user-injected outbounds.
type UserNodeProvider struct {
	outbounds []option.Outbound
}

func NewUserNodeProvider(outbounds []option.Outbound) *UserNodeProvider {
	return &UserNodeProvider{outbounds: outbounds}
}

func (p *UserNodeProvider) Name() string {
	return "user"
}

func (p *UserNodeProvider) GetNodes() ([]model.Node, error) {
	nodes := make([]model.Node, 0, len(p.outbounds))
	for _, out := range p.outbounds {
		if out.Tag == "" || !IsActualOutboundType(out.Type) {
			continue
		}

		outboundMap, err := outboundToMap(out)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, model.Node{
			Name:     out.Tag,
			Type:     out.Type,
			Source:   "user",
			Outbound: outboundMap,
		})
	}
	return nodes, nil
}

func outboundToMap(out option.Outbound) (map[string]any, error) {
	data, err := singboxjson.Marshal(out)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
