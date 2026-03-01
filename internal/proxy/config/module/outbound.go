package module

import (
	"github.com/sagernet/sing-box/option"
)

// OutboundModule 出站模块
// 负责组装和构建 proxy, direct, block 以及各种出站节点群
type OutboundModule struct {
	providers []NodeProvider
}

// NewOutboundModule creates a new outbound module with the given providers.
func NewOutboundModule(providers ...NodeProvider) *OutboundModule {
	return &OutboundModule{providers: providers}
}

func (m *OutboundModule) Name() string {
	return "outbound"
}

func (m *OutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	processor := NewOutboundProcessor()

	// 1. 从所有 Provider 获取节点
	for _, provider := range m.providers {
		nodes, err := provider.GetNodes()
		if err != nil {
			return err
		}
		processor.AddNodes(nodes)
	}

	// 2. 获取去重且正确命名后的 proxy 出站节点
	filteredOutbounds := make([]option.Outbound, 0)
	filteredOutbounds = append(filteredOutbounds, processor.GetProcessedOutbounds()...)

	actualNodes := processor.GetActualTags()

	// 3. 构建内置出站
	// 5. 添加 direct 出站
	directOutbound := option.Outbound{}
	directOutboundMap := map[string]any{
		"type": TagDirect,
		"tag":  TagDirect,
	}
	ApplyMapToOutbound(&directOutbound, directOutboundMap)
	filteredOutbounds = append(filteredOutbounds, directOutbound)

	// 6. 添加 block 出站
	blockOutbound := option.Outbound{}
	blockOutboundMap := map[string]any{
		"type": TagBlock,
		"tag":  TagBlock,
	}
	ApplyMapToOutbound(&blockOutbound, blockOutboundMap)
	filteredOutbounds = append(filteredOutbounds, blockOutbound)

	// 根据是否有实际节点决定如何配置 auto 和 proxy 策略组
	if len(actualNodes) > 0 {
		// 有节点时的逻辑：
		// - auto: urltest [all nodes]
		// - proxy: selector [auto, ...all nodes]

		// 7. 添加 proxy selector
		proxyNodes := append([]string{TagAuto}, actualNodes...)
		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       TagProxy,
			"outbounds": proxyNodes,
			"default":   TagAuto,
		}
		ApplyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)

		// 8. 添加 auto urltest
		autoOutbound := option.Outbound{}
		autoOutboundMap := map[string]any{
			"type":      "urltest",
			"tag":       TagAuto,
			"outbounds": actualNodes,
		}
		ApplyMapToOutbound(&autoOutbound, autoOutboundMap)
		filteredOutbounds = append(filteredOutbounds, autoOutbound)
	} else {
		// 无节点时的逻辑：
		// - proxy: selector [direct]
		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       TagProxy,
			"outbounds": []string{TagDirect},
			"default":   TagDirect,
		}
		ApplyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)
	}

	// 4. 将合并后的出站回填
	opts.Outbounds = append(opts.Outbounds, filteredOutbounds...)

	return nil
}
