package config

import (
	"context"
	"os"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// UserProfile 是用户实际维护的配置文件
type UserProfile struct {
	Outbounds []option.Outbound    `json:"outbounds"`
	Route     *option.RouteOptions `json:"route,omitempty"`
	// 用户可以定义自己喜欢的 DNS 服务器，但劫持规则由我们生成
	//RemoteDNS []option.DNSServerOptions `json:"dns_servers,omitempty"`
}

// LoadProfile 加载用户配置
func LoadProfile(path string) (*UserProfile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p UserProfile
	// 使用 sing-box 的 JSON 解析器，带 include.Context 来正确解析 Outbound 类型
	ctx := include.Context(context.Background())
	if err := singboxjson.UnmarshalContext(ctx, content, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
