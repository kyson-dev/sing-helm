package config

import (
	"github.com/kysonzou/sing-helm/internal/env"
	"github.com/kysonzou/sing-helm/internal/logger"
	"github.com/kysonzou/sing-helm/internal/subscription"
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

	// 收集已使用的 tags（包括用户配置的 outbounds）
	usedTags := map[string]bool{}
	for _, out := range opts.Outbounds {
		if out.Tag == "" || IsReservedOutboundTag(out.Tag) {
			continue
		}
		usedTags[out.Tag] = true
	}

	tagBySource := map[string]map[string]string{}
	assignedTags := make([]string, len(nodes))
	for i, node := range nodes {
		if node.Outbound == nil {
			continue
		}
		tag := MakeUniqueOutboundTag(node.Name, node.Source, usedTags)
		assignedTags[i] = tag
		if node.Name != "" {
			if _, ok := tagBySource[node.Source]; !ok {
				tagBySource[node.Source] = map[string]string{}
			}
			tagBySource[node.Source][node.Name] = tag
		}
	}

	// 将订阅节点追加到 opts.Outbounds
	for i, node := range nodes {
		if node.Outbound == nil {
			continue
		}
		outMap := make(map[string]any, len(node.Outbound)+1)
		for key, value := range node.Outbound {
			outMap[key] = value
		}
		if detour, ok := outMap["detour"].(string); ok {
			if mapped := tagBySource[node.Source][detour]; mapped != "" {
				outMap["detour"] = mapped
			}
		}
		tag := assignedTags[i]
		if tag == "" {
			tag = MakeUniqueOutboundTag(node.Name, node.Source, usedTags)
		}
		outMap["tag"] = tag

		out := option.Outbound{}
		if err := applyMapToOutbound(&out, outMap); err != nil {
			logger.Error("Failed to apply outbound from subscription", "name", node.Name, "error", err)
			continue
		}

		opts.Outbounds = append(opts.Outbounds, out)
	}

	return nil
}
