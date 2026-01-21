package config

import (
	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/kyson-dev/sing-helm/internal/logger"
	"github.com/kyson-dev/sing-helm/internal/subscription"
	"github.com/sagernet/sing-box/option"
)

// SubscriptionModule merges cached subscription nodes into outbounds.
type SubscriptionModule struct{}

func (m *SubscriptionModule) Name() string {
	return "subscription"
}

func (m *SubscriptionModule) Apply(opts *option.Options, ctx *BuildContext) error {
	paths := env.Get()
	sources, err := subscription.LoadSources(paths.SubConfigDir)
	if err != nil {
		logger.Error("Failed to load subscription sources", "error", err)
	}

	nodes, err := subscription.LoadNodesFromCache(sources, paths.SubCacheDir)
	if err != nil {
		logger.Error("Failed to load subscription cache", "error", err)
		return nil
	}

	if len(nodes) == 0 {
		return nil
	}

	// 1. 收集已有的 tags
	usedTags := make(map[string]bool)
	for _, out := range opts.Outbounds {
		if out.Tag != "" {
			usedTags[out.Tag] = true
		}
	}

	// 2. 按 Source 分组节点
	nodesBySource := map[string][]RawOutbound{}
	// 为了保持顺序（可选），可以维护一个 source 列表，但 map 遍历顺序随机
	// 这里简单处理，因为不同 source 之间无依赖
	for _, node := range nodes {
		if node.Outbound == nil || node.Source == "" {
			continue
		}
		if _, ok := nodesBySource[node.Source]; !ok {
			nodesBySource[node.Source] = make([]RawOutbound, 0)
		}
		// 复制 Outbound map 并设置 tag 为节点名
		outboundCopy := make(map[string]any, len(node.Outbound)+1)
		for k, v := range node.Outbound {
			outboundCopy[k] = v
		}
		// 使用节点名作为 tag 的 base（Processor 会处理冲突）
		if node.Name != "" {
			outboundCopy["tag"] = node.Name
		}
		nodesBySource[node.Source] = append(nodesBySource[node.Source], RawOutbound(outboundCopy))
	}

	// 3. 使用 Processor 处理每个分组
	processor := NewOutboundProcessor(usedTags)
	for source, rawOutbounds := range nodesBySource {
		processed, err := processor.Process(rawOutbounds, source)
		if err != nil {
			logger.Error("Failed to process subscription outbounds", "source", source, "error", err)
			continue
		}
		opts.Outbounds = append(opts.Outbounds, processed...)
	}

	return nil
}
