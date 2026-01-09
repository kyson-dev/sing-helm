package config

import (
	"github.com/kyson/sing-helm/internal/logger"
	"github.com/sagernet/sing-box/option"
)

// OutboundModule 出站模块
// 负责处理所有 outbounds（用户配置 + 订阅节点），并补充系统 outbounds
type OutboundModule struct{}

func (m *OutboundModule) Name() string {
	return "outbound"
}

func (m *OutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 1. 过滤保留 tag，并统计节点信息
	filteredOutbounds := []option.Outbound{}
	userNodeTags := []string{}
	actualNodes := []string{}

	for _, out := range opts.Outbounds {
		if IsReservedOutboundTag(out.Tag) {
			logger.Info("Ignoring reserved outbound tag from user config", "tag", out.Tag)
			continue
		}
		filteredOutbounds = append(filteredOutbounds, out)
		if out.Tag != "" {
			userNodeTags = append(userNodeTags, out.Tag)
			if IsActualOutboundType(out.Type) {
				actualNodes = append(actualNodes, out.Tag)
			}
		}
	}

	// 2. 添加 direct 出站
	directOutbound := option.Outbound{}
	directOutboundMap := map[string]any{
		"type": "direct",
		"tag":  "direct",
	}
	applyMapToOutbound(&directOutbound, directOutboundMap)
	filteredOutbounds = append(filteredOutbounds, directOutbound)

	// 3. 添加 block 出站
	blockOutbound := option.Outbound{}
	blockOutboundMap := map[string]any{
		"type": "block",
		"tag":  "block",
	}
	applyMapToOutbound(&blockOutbound, blockOutboundMap)
	filteredOutbounds = append(filteredOutbounds, blockOutbound)

	// 4 & 5. 添加 proxy selector 和 auto urltest
	if len(actualNodes) > 0 {
		// 有节点时的逻辑：
		// - auto: urltest [all nodes]
		// - proxy: selector [auto, ...all nodes]

		// 4. 添加 proxy selector
		proxyNodes := append([]string{"auto"}, actualNodes...)
		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": proxyNodes,
			"default":   "auto",
		}
		applyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)

		// 5. 添加 auto urltest
		autoOutbound := option.Outbound{}
		autoOutboundMap := map[string]any{
			"type":      "urltest",
			"tag":       "auto",
			"outbounds": actualNodes,
		}
		applyMapToOutbound(&autoOutbound, autoOutboundMap)
		filteredOutbounds = append(filteredOutbounds, autoOutbound)
	} else {
		// 无节点时的逻辑：
		// - proxy: selector [direct] (降级为直连)
		// - 不创建 auto 组 (因为没有节点可以测速)

		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": []string{"direct"},
			"default":   "direct",
		}
		applyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)
	}

	// 6. 更新最终的 outbounds
	opts.Outbounds = filteredOutbounds

	return nil
}
