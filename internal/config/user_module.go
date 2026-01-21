package config

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/sagernet/sing-box/option"
)

// UserOutboundModule collects user outbounds into build context.
type UserOutboundModule struct{}

func (m *UserOutboundModule) Name() string {
	return "user_outbound"
}

func (m *UserOutboundModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 如果没有提供 ProfilePath，说明用户配置已经在 opts 中了（向后兼容）
	paths := env.Get()

	content, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// 如果文件为空或只包含空白字符，直接返回（允许用户不配置任何内容）
	if len(bytes.TrimSpace(content)) == 0 {
		return nil
	}

	// 1. 收集已有的 tags 以避免冲突
	usedTags := make(map[string]bool)
	for _, out := range opts.Outbounds {
		if out.Tag != "" {
			usedTags[out.Tag] = true
		}
	}

	// 2. 解析为通用 Map 以提取 outbounds
	var rawConfig map[string]any
	// 使用 singboxjson.Unmarshal 以支持注释等特性 (如果 json 包不支持)
	// 但这里我们用标准 json 包即可，profile.json 通常是标准 JSON
	if err := json.Unmarshal(content, &rawConfig); err != nil {
		return err
	}

	// 3. 提取并处理 Outbounds
	if rawOutboundsVal, ok := rawConfig["outbounds"]; ok {
		var typedOutbounds []RawOutbound

		// 处理 []any 类型 (标准 json 解析结果)
		if list, ok := rawOutboundsVal.([]any); ok {
			typedOutbounds = make([]RawOutbound, 0, len(list))
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					typedOutbounds = append(typedOutbounds, RawOutbound(m))
				}
			}
		}

		if len(typedOutbounds) > 0 {
			processor := NewOutboundProcessor(usedTags)
			outbounds, err := processor.Process(typedOutbounds, "")
			if err != nil {
				return err
			}
			opts.Outbounds = append(opts.Outbounds, outbounds...)
		}
	}

	return nil
}
