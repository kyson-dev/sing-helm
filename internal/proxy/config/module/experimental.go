package module

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
	"github.com/sagernet/sing-box/option"
)

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
	// 如果用户已经在 profile.json 中完全配置了 experimental，尤其是 clash_api
	// 我们就直接跳过（依赖 TemplateModule 前置早已完成了参数提取）
	if opts.Experimental != nil && opts.Experimental.ClashAPI != nil && opts.Experimental.ClashAPI.ExternalController != "" {
		if ctx != nil && ctx.RunOptions != nil {
			listenAddr, apiPort, ok := parseExternalController(opts.Experimental.ClashAPI.ExternalController)
			if !ok {
				return fmt.Errorf("invalid experimental.clash_api.external_controller: %q", opts.Experimental.ClashAPI.ExternalController)
			}
			ctx.RunOptions.ListenAddr = listenAddr
			ctx.RunOptions.APIPort = apiPort
		}
		return nil
	}
	// 确定监听地址
	listenAddr := m.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	// 确定 API 端口
	apiPort := m.APIPort
	if apiPort == 0 {
		var err error
		apiPort, err = moduleUtils.GetFreePort()
		if err != nil {
			return err
		}
	}

	// 更新 context 中的端口信息
	ctx.RunOptions.APIPort = apiPort
	ctx.RunOptions.ListenAddr = listenAddr

	// 创建或追加 Clash API 配置
	if opts.Experimental == nil {
		opts.Experimental = &option.ExperimentalOptions{}
	}
	opts.Experimental.ClashAPI = &option.ClashAPIOptions{
		ExternalController: fmt.Sprintf("%s:%d", listenAddr, apiPort),
	}

	if opts.Experimental.CacheFile == nil {
		opts.Experimental.CacheFile = &option.CacheFileOptions{
			Enabled: true,
			Path:    paths.Get().CacheFile,
		}
	}

	return nil
}

func parseExternalController(externalController string) (string, int, bool) {
	controller := strings.TrimSpace(externalController)
	if controller == "" {
		return "", 0, false
	}

	host, portStr, err := net.SplitHostPort(controller)
	if err != nil {
		// Backward-compatible fallback for plain "host:port" forms.
		lastColon := strings.LastIndex(controller, ":")
		if lastColon <= 0 || lastColon == len(controller)-1 {
			return "", 0, false
		}
		host = controller[:lastColon]
		portStr = controller[lastColon+1:]
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, false
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", 0, false
	}

	return host, port, true
}
