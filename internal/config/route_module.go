package config

import (
	"github.com/kyson/minibox/internal/runtime"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// RouteModule 路由模块
// 负责配置路由规则，支持 RouteMode
type RouteModule struct {
	RouteMode runtime.RouteMode
}

func (m *RouteModule) Name() string {
	return "route"
}

func (m *RouteModule) Apply(opts *option.Options, ctx *BuildContext) error {
	// 如果用户没有配置路由，使用默认路由
	if opts.Route == nil {
		defaultRoute, err := m.generateDefaultRoute()
		if err != nil {
			return err
		}
		opts.Route = defaultRoute
	}

	// 根据 RouteMode 调整路由
	switch m.RouteMode {
	case runtime.RouteModeGlobal:
		// 全局代理：清空所有路由规则，直接走 proxy
		// 保留 RuleSet 以供 DNS 规则使用
		opts.Route.Rules = nil
		opts.Route.Final = "proxy"
	case runtime.RouteModeDirect:
		// 全局直连：清空所有路由规则，直接走 direct
		// 保留 RuleSet 以供 DNS 规则使用
		opts.Route.Rules = nil
		opts.Route.Final = "direct"
	case runtime.RouteModeRule, "":
		// rule 模式保持用户配置的路由规则
		if opts.Route.Final == "" {
			opts.Route.Final = "proxy" // 默认 final 走代理
		}
	}

	return nil
}

// generateDefaultRoute 生成默认路由规则
// 当用户没有配置 Route 时使用
func (m *RouteModule) generateDefaultRoute() (*option.RouteOptions, error) {
	routeMap := map[string]any{
		"rule_set": []map[string]any{
			{
				"download_detour": "proxy",
				"format":          "binary",
				"tag":             "geosite-tld-cn",
				"type":            "remote",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-tld-cn.srs",
			},
			{
				"tag":             "geosite-google",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geoip-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-apple",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-apple.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-ads",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "anti-ad",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/privacy-protection-tools/anti-ad.github.io/master/docs/anti-ad-sing-box.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-geolocation-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-bilibili",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-bilibili.srs",
				"download_detour": "proxy",
			},
			// 中国媒体服务
			{
				"tag":             "geosite-category-media-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-media-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-category-entertainment-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-entertainment-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-category-social-media-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-social-media-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-tencent",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-tencent.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-iqiyi",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-iqiyi.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-youku",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-youku.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-baidu",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-baidu.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-netease",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-netease.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-douyin",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-douyin.srs",
				"download_detour": "proxy",
			},
			// 中国 CDN 服务商 (解决图片加载慢)
			{
				"tag":             "geosite-category-cdn-cn",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-cdn-cn.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-wangsu",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-wangsu.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-kingsoft",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-kingsoft.srs",
				"download_detour": "proxy",
			},
			{
				"tag":             "geosite-qiniu",
				"type":            "remote",
				"format":          "binary",
				"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-qiniu.srs",
				"download_detour": "proxy",
			},
		},
		"rules": []map[string]any{
			// 直连白名单
			{"domain_suffix": []string{"wise.com", "schwab.com", "interactivebrokers.com", "soulapp.cn", "soul.cn"}, "outbound": "direct"},
			// 广告屏蔽
			{"rule_set": []string{"anti-ad", "geosite-ads"}, "outbound": "block"},
			// 1. DNS 劫持
			{"protocol": []string{"dns"}, "action": "hijack-dns"},
			// 1.1 AliDNS upstream (avoid proxy latency)
			{"ip_cidr": []string{"223.5.5.5/32", "223.6.6.6/32", "2400:3200::/32"}, "outbound": "direct"},
			// 2. NTP 直连
			{"protocol": []string{"ntp"}, "outbound": "direct"},
			// 4. 私有 IP 直连
			{"ip_is_private": true, "outbound": "direct"},
			// 7. Apple 直连
			{"rule_set": []string{"geosite-apple"}, "outbound": "direct"},
			// 6. CN 直连 (含媒体、社交、娱乐)
			{"rule_set": []string{
				"geosite-cn", "geosite-tld-cn", "geoip-cn",
				"geosite-geolocation-cn", "geosite-bilibili",
				"geosite-category-media-cn", "geosite-category-entertainment-cn", "geosite-category-social-media-cn",
				"geosite-tencent", "geosite-iqiyi", "geosite-youku", "geosite-baidu", "geosite-netease", "geosite-douyin",
				"geosite-category-cdn-cn", "geosite-wangsu", "geosite-kingsoft", "geosite-qiniu",
			}, "outbound": "direct"},
		},
		"final":                 "proxy",
		"auto_detect_interface": true,
	}

	data, err := singboxjson.Marshal(routeMap)
	if err != nil {
		return nil, err
	}

	var routeOpts option.RouteOptions
	if err := singboxjson.Unmarshal(data, &routeOpts); err != nil {
		return nil, err
	}

	return &routeOpts, nil
}
