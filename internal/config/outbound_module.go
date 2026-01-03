package config

import (
	"github.com/kyson/minibox/internal/logger"
	"github.com/sagernet/sing-box/option"
)

// OutboundModule 出站模块
// 负责补充 direct, block, proxy, auto 出站
type OutboundModule struct{}

func (m *OutboundModule) Name() string {
	return "outbound"
}

func (m *OutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 1. 过滤用户配置中的保留 tag
	reservedTags := map[string]bool{
		"direct": true,
		"block":  true,
		"proxy":  true,
		"auto":   true,
	}

	filteredOutbounds := []option.Outbound{}
	for _, out := range opts.Outbounds {
		if reservedTags[out.Tag] {
			logger.Info("Ignoring reserved outbound tag from user config", "tag", out.Tag)
			continue
		}
		filteredOutbounds = append(filteredOutbounds, out)
	}
	opts.Outbounds = filteredOutbounds

	// 2. 添加 direct 出站
	directOutbound := option.Outbound{}
	directOutboundMap := map[string]any{
		"type": "direct",
		"tag":  "direct",
	}
	applyMapToOutbound(&directOutbound, directOutboundMap)
	opts.Outbounds = append(opts.Outbounds, directOutbound)

	// 3. 添加 block 出站
	blockOutbound := option.Outbound{}
	blockOutboundMap := map[string]any{
		"type": "block",
		"tag":  "block",
	}
	applyMapToOutbound(&blockOutbound, blockOutboundMap)
	opts.Outbounds = append(opts.Outbounds, blockOutbound)

	// 4. 添加 proxy selector（包含所有实际节点 + auto）
	proxyNodes := append([]string{"auto"}, ctx.ActualNodes...)
	proxyOutbound := option.Outbound{}
	proxyOutboundMap := map[string]any{
		"type":      "selector",
		"tag":       "proxy",
		"outbounds": proxyNodes,
		"default":   "auto",
	}
	applyMapToOutbound(&proxyOutbound, proxyOutboundMap)
	opts.Outbounds = append(opts.Outbounds, proxyOutbound)

	// 5. 添加 auto urltest（包含所有实际节点）
	autoOutbound := option.Outbound{}
	autoOutboundMap := map[string]any{
		"type":      "urltest",
		"tag":       "auto",
		"outbounds": ctx.ActualNodes,
	}
	applyMapToOutbound(&autoOutbound, autoOutboundMap)
	opts.Outbounds = append(opts.Outbounds, autoOutbound)

	return nil
}
