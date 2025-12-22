package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/option"
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

	//3. 解析配置文件（JSON）
	var options option.Options
	if err := json.Unmarshal(content, &options); err != nil {
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
