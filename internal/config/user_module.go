package config

import (
	"bytes"
	"context"
	"os"

	"github.com/kyson-dev/sing-helm/internal/env"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
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

	var opts_user option.Options
	// 使用 sing-box 的 JSON 解析器，include.Context 确保正确解析 Outbound 类型
	tx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(tx, content, &opts_user); err != nil {
		return err
	}

	// 将用户配置的 outbounds 添加到 opts.Outbounds
	// 其他配置项（如 log, dns, inbounds 等）会被后续模块覆盖或合并
	if len(opts_user.Outbounds) > 0 {
		opts.Outbounds = append(opts.Outbounds, opts_user.Outbounds...)
	}

	return nil
}
