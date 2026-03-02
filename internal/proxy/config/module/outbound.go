package module

import (
	nodeProvider "github.com/kyson-dev/sing-helm/internal/proxy/config/module/node"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

// OutboundModule 出站模块
// 负责组装和构建 proxy, direct, block 以及各种出站节点群
type OutboundModule struct {
	providers []nodeProvider.NodeProvider
}

// NewOutboundModule creates a new outbound module with the given providers.
func NewOutboundModule(providers ...nodeProvider.NodeProvider) *OutboundModule {
	return &OutboundModule{providers: providers}
}

func (m *OutboundModule) Name() string {
	return "outbound"
}

func (m *OutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	processor := nodeProvider.NewOutboundProcessor()
	providers := make([]nodeProvider.NodeProvider, 0, len(m.providers)+1)
	providers = append(providers, nodeProvider.NewUserNodeProvider(opts.Outbounds))
	providers = append(providers, m.providers...)

	// 1. 从所有 Provider 获取节点
	for _, provider := range providers {
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
		"type": moduleUtils.TagDirect,
		"tag":  moduleUtils.TagDirect,
	}
	moduleUtils.ApplyMapToOutbound(&directOutbound, directOutboundMap)
	filteredOutbounds = append(filteredOutbounds, directOutbound)

	// 6. 添加 block 出站
	blockOutbound := option.Outbound{}
	blockOutboundMap := map[string]any{
		"type": moduleUtils.TagBlock,
		"tag":  moduleUtils.TagBlock,
	}
	moduleUtils.ApplyMapToOutbound(&blockOutbound, blockOutboundMap)
	filteredOutbounds = append(filteredOutbounds, blockOutbound)

	// 根据是否有实际节点决定如何配置 auto 和 proxy 策略组
	if len(actualNodes) > 0 {

		// 7. 添加 proxy selector
		proxyNodes := append([]string{moduleUtils.TagAuto}, actualNodes...)
		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       moduleUtils.TagProxy,
			"outbounds": proxyNodes,
			"default":   moduleUtils.TagAuto,
		}
		moduleUtils.ApplyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)

		// 8. 添加 auto urltest
		autoOutbound := option.Outbound{}
		autoOutboundMap := map[string]any{
			"type":      "urltest",
			"tag":       moduleUtils.TagAuto,
			"outbounds": actualNodes,
		}
		moduleUtils.ApplyMapToOutbound(&autoOutbound, autoOutboundMap)
		filteredOutbounds = append(filteredOutbounds, autoOutbound)
	} else {
		// 无节点时的逻辑：
		// - proxy: selector [direct]
		proxyOutbound := option.Outbound{}
		proxyOutboundMap := map[string]any{
			"type":      "selector",
			"tag":       moduleUtils.TagProxy,
			"outbounds": []string{moduleUtils.TagDirect},
			"default":   moduleUtils.TagDirect,
		}
		moduleUtils.ApplyMapToOutbound(&proxyOutbound, proxyOutboundMap)
		filteredOutbounds = append(filteredOutbounds, proxyOutbound)
	}

	// 4. 将合并后的出站回填
	// 规则：
	// - 硬编码/生成出站优先：与用户同名时舍弃用户定义
	// - 用户其他自定义出站保留
	userGeneratedTags := make(map[string]bool)
	for _, tag := range processor.GetGroups()["user"] {
		userGeneratedTags[tag] = true
	}

	generatedByTag := make(map[string]bool, len(filteredOutbounds))
	for _, fo := range filteredOutbounds {
		generatedByTag[fo.Tag] = true
	}

	preservedUserOutbounds := make([]option.Outbound, 0, len(opts.Outbounds))
	for _, out := range opts.Outbounds {
		// 同名时硬编码优先，丢弃用户定义
		if generatedByTag[out.Tag] && !userGeneratedTags[out.Tag] {
			continue
		}

		// 用户 selector/urltest 的 outbounds 为空数组时，自动填充全部实际节点。
		if len(actualNodes) > 0 {
			switch outOpts := out.Options.(type) {
			case *option.SelectorOutboundOptions:
				if len(outOpts.Outbounds) == 0 {
					outOpts.Outbounds = append([]string(nil), actualNodes...)
				}
			case *option.URLTestOutboundOptions:
				if len(outOpts.Outbounds) == 0 {
					outOpts.Outbounds = append([]string(nil), actualNodes...)
				}
			}
		}

		preservedUserOutbounds = append(preservedUserOutbounds, out)
	}

	// 先保留用户剩余配置，再追加硬编码/生成出站（同名已在上面剔除用户项）。
	opts.Outbounds = preservedUserOutbounds
	for _, fo := range filteredOutbounds {
		if userGeneratedTags[fo.Tag] {
			continue
		}
		opts.Outbounds = append(opts.Outbounds, fo)
	}

	return nil
}
