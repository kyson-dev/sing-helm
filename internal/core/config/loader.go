package config

import (
	"context"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// Load 从指定路径读取并解析配置文件
// 返回 sing-box 官方定义的 Options 结构体指针
func Load(configPath string) (*option.Options, error) {
	//1. 检查文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config file not found at: %s", configPath)
	}

	//2. 读取文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	//3. 使用 sing-box 的 JSON 解析器解析配置文件
	// 需要使用 include.Context 初始化 context，以便正确解析类型特定的选项
	ctx := include.Context(context.Background())
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	//4. 基础校验
	if err := checkOptions(&options); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &options, nil
}

func checkOptions(options *option.Options) error {
	if len(options.Inbounds) == 0 && len(options.Outbounds) == 0 {
		return fmt.Errorf("config must have at least one inbound or outbound")
	}
	return nil
}
