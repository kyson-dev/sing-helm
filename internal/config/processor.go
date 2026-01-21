package config

import (
	"context"

	"github.com/kyson-dev/sing-helm/internal/logger"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// OutboundProcessor 处理出站节点的通用逻辑
type OutboundProcessor struct {
	UsedTags map[string]bool
}

// NewOutboundProcessor 创建处理器
func NewOutboundProcessor(existingTags map[string]bool) *OutboundProcessor {
	if existingTags == nil {
		existingTags = make(map[string]bool)
	}
	return &OutboundProcessor{
		UsedTags: existingTags,
	}
}

// RawOutbound 表示通用的出站配置 map
type RawOutbound map[string]any

// ProcessStandard 处理标准出站列表 (例如来自用户配置)
// source: 用于标识这批节点的来源，用于生成 Tag (例如 "user" 或订阅名)
func (p *OutboundProcessor) Process(outbounds []RawOutbound, source string) ([]option.Outbound, error) {
	if len(outbounds) == 0 {
		return nil, nil
	}

	// 1. Pass 1: 分配唯一 Tag，建立 Old -> New 映射
	tagMapping := make(map[string]string)
	newTags := make([]string, len(outbounds))

	for i, out := range outbounds {
		// 获取原始 Tag
		oldTag := ""
		if v, ok := out["tag"].(string); ok {
			oldTag = v
		}

		// 生成唯一 Tag (如果是用户配置，source 可以传空或是 "user")
		// 对于订阅，我们可能需要更复杂的命名逻辑，这里我们复用 MakeUniqueTag
		// 如果需要包含 source，可以在调用前把 oldTag 格式化好，或者修改 MakeUniqueTag

		// 为了兼容现有的订阅命名逻辑 (Name + Source)，我们需要更灵活的入参
		// 这里假设 out["tag"] 已经是基础名字了

		newTag := ""
		if source != "" && source != "user" {
			// 订阅模式：使用 Base + Source
			newTag = MakeUniqueOutboundTag(oldTag, source, p.UsedTags)
		} else {
			// 用户模式：直接使用 Base，冲突加后缀
			if oldTag == "" {
				oldTag = "node"
			}
			newTag = MakeUniqueTag(oldTag, p.UsedTags)
		}

		newTags[i] = newTag
		if oldTag != "" {
			tagMapping[oldTag] = newTag
		}
	}

	// 2. Pass 2: 应用 Tag 并修正 Detour，转换为对象
	results := make([]option.Outbound, 0, len(outbounds))

	for i, outMap := range outbounds {
		// 1. 修正 Detour 字段 (单引用)
		if detour, ok := outMap["detour"].(string); ok && detour != "" {
			if mapped, exists := tagMapping[detour]; exists {
				outMap["detour"] = mapped
			}
		}

		// 2. 修正 Outbounds 列表 (多引用，用于 selector/urltest/chain)
		// 注意：JSON 解析出来可能是 []any 或 []string
		if rawList, ok := outMap["outbounds"]; ok {
			var newList []string
			changed := false

			// 处理 []string
			if strList, ok := rawList.([]string); ok {
				newList = make([]string, len(strList))
				for j, tag := range strList {
					if mapped, exists := tagMapping[tag]; exists {
						newList[j] = mapped
						changed = true
					} else {
						newList[j] = tag
					}
				}
			} else if anyList, ok := rawList.([]any); ok {
				// 处理 []any
				newList = make([]string, len(anyList))
				for j, v := range anyList {
					if tag, ok := v.(string); ok {
						if mapped, exists := tagMapping[tag]; exists {
							newList[j] = mapped
							changed = true
						} else {
							newList[j] = tag
						}
					}
				}
			}

			if changed {
				outMap["outbounds"] = newList
			}
		}

		// 3. 应用新 Tag
		outMap["tag"] = newTags[i]

		// 转换为 option.Outbound
		var out option.Outbound
		if err := p.applyMapToOutbound(&out, outMap); err != nil {
			logger.Error("Failed to convert outbound", "tag", newTags[i], "error", err)
			continue
		}
		results = append(results, out)
	}

	return results, nil
}

// applyMapToOutbound 辅助函数：Map -> Outbound
func (p *OutboundProcessor) applyMapToOutbound(out *option.Outbound, m RawOutbound) error {
	// 先序列化回 JSON
	data, err := singboxjson.Marshal(m)
	if err != nil {
		return err
	}
	// 再反序列化为 Struct
	// 必须使用 include.Context，否则 interface{} 类型的字段无法正确解析
	ctx := include.Context(context.Background())
	return singboxjson.UnmarshalContext(ctx, data, out)
}
