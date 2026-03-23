package config

import (
	"context"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// LoadOptionsWithContext 从配置文件加载 sing-box 配置
func LoadOptionsWithContext(ctx context.Context, configPath string) (*option.Options, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var opts option.Options
	includeCtx := include.Context(ctx)
	if err := singboxjson.UnmarshalContext(includeCtx, data, &opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &opts, nil
}

