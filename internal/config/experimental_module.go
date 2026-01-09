package config

import (
	"fmt"

	"github.com/kyson/sing-helm/internal/env"
	"github.com/kyson/sing-helm/internal/pkg/netutil"
	"github.com/sagernet/sing-box/option"
)

const testAPIPortEnv = "MINIBOX_TEST_API_PORT"

// ExperimentalModule 实验性模块
// 负责配置 Clash API 和缓存
type ExperimentalModule struct {
	ListenAddr string
	APIPort    int
}

func (m *ExperimentalModule) Name() string {
	return "experimental"
}

func (m *ExperimentalModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 确定监听地址
	listenAddr := m.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	// 确定 API 端口
	apiPort := m.APIPort
	if apiPort == 0 {
		if override, ok := getPortOverride(testAPIPortEnv); ok {
			apiPort = override
		} else {
			var err error
			apiPort, err = netutil.GetFreePort()
			if err != nil {
				return err
			}
		}
	}

	// 更新 context 中的端口信息
	ctx.RunOptions.APIPort = apiPort
	ctx.RunOptions.ListenAddr = listenAddr

	// 创建 Clash API 配置
	opts.Experimental = &option.ExperimentalOptions{
		ClashAPI: &option.ClashAPIOptions{
			ExternalController: fmt.Sprintf("%s:%d", listenAddr, apiPort),
		},
		CacheFile: &option.CacheFileOptions{
			Enabled: true,
			Path:    env.Get().CacheFile,
		},
	}

	return nil
}
