package module

import (
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/sagernet/sing-box/option"
)

// OutboundModule 出站模块
// 负责处理所有 outbounds（用户配置 + 订阅节点），并补充系统 outbounds
type OutboundModule struct {
	Providers []NodeProvider
}

func (m *OutboundModule) Name() string {
	return "outbound"
}

func (m *OutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 1. 获取已有的 tags
	usedTags := make(map[string]bool)
	for _, out := range opts.Outbounds {
		if out.Tag != "" {
			usedTags[out.Tag] = true
		}
	}

	processor := NewOutboundProcessor(usedTags)

	// 2. 收集所有节点
	var nodes []Node
	for _, p := range m.Providers {
		pNodes, err := p.GetNodes()
		if err != nil {
			logger.Error("Failed to get nodes from provider", "provider", p.Name(), "error", err)
			continue
		}
		nodes = append(nodes, pNodes...)
	}

	// 3. 按 Source 分组并放入 Processor
	nodesBySource := map[string][]RawOutbound{}
	for _, node := range nodes {
		if _, ok := nodesBySource[node.Source]; !ok {
			nodesBySource[node.Source] = make([]RawOutbound, 0)
		}
		outboundCopy := node.Outbound
		if node.Name != "" {
			outboundCopy["tag"] = node.Name
		}
		nodesBySource[node.Source] = append(nodesBySource[node.Source], RawOutbound(outboundCopy))
	}

	// 4. 处理分组的节点并收集最终生成的所有 outbounds 和真实的节点 tag 列表
	filteredOutbounds := []option.Outbound{}
	actualNodes := []string{}

	for source, rawOutbounds := range nodesBySource {
		processed, err := processor.Process(rawOutbounds, source)
		if err != nil {
			logger.Error("Failed to process outbounds", "source", source, "error", err)
			continue
		}
		for _, out := range processed {
			if IsReservedOutboundTag(out.Tag) {
				logger.Info("Ignoring reserved outbound tag from provider config", "tag", out.Tag, "source", source)
				continue
			}
			filteredOutbounds = append(filteredOutbounds, out)
			// 注意，这里的 IsActualOutboundType 需要处理
			if out.Type != "selector" && out.Type != "urltest" && out.Type != "direct" && out.Type != "block" && out.Type != "dns" {
				actualNodes = append(actualNodes, out.Tag)
			}
		}
	}

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

	// 7 & 8. 添加 proxy selector 和 auto urltest
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
		// - proxy: selector [direct] (降级为直连)
		// - 不创建 auto 组 (因为没有节点可以测速)

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

	// 9. 更新最终的 outbounds
	opts.Outbounds = append(opts.Outbounds, filteredOutbounds...)

	return nil
}
