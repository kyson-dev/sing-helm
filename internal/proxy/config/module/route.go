package module

import (
	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
	singboxjson "github.com/sagernet/sing/common/json"
)

// RouteModule 路由模块
// 负责组装和构建动态路由协议栈，通过拼装不同的 RouteFragment 来实现灵活的规则扩展
type RouteModule struct {
	RouteMode model.RouteMode
}

func (m *RouteModule) Name() string {
	return "route"
}

func (m *RouteModule) Apply(opts *option.Options, ctx *BuildContext) error {
	if opts.Route == nil {
		opts.Route = &option.RouteOptions{}
	}

	// 1. 如果用户没有自定义 final 出站，设置默认
	if opts.Route.Final == "" {
		opts.Route.Final = moduleUtils.TagProxy
	}

	// 2. 将全局/直连模式转化为更高级别的劫持
	switch m.RouteMode {
	case model.RouteModeGlobal:
		// 全局代理：覆盖前面的默认 Final
		opts.Route.Final = moduleUtils.TagProxy
		// 但我们需要保留 DNS 和局域网绕过的规则，因此我们仍然应用 default 规则
	case model.RouteModeDirect:
		// 全局直连：所有流量默认直连
		opts.Route.Final = moduleUtils.TagDirect
	}

	// 3. 构建并应用默认扩展拼图 (当用户没有完全接管路由时)
	// 如果用户自己配了 rule_set，我们尽量把系统必备的加到后面。
	if len(opts.Route.Rules) == 0 {
		m.applyDefaultFragments(opts)
	}

	// 4. 清空特定模式下的所有非必要规则
	// 对于全局/直连，我们可以强制清空普通路由
	if m.RouteMode == model.RouteModeGlobal || m.RouteMode == model.RouteModeDirect {
		opts.Route.Rules = nil
	}

	return nil
}

// applyDefaultFragments 组装默认的开箱即用路由规则
func (m *RouteModule) applyDefaultFragments(opts *option.Options) error {
	var ruleSets []map[string]any
	var rules []map[string]any

	// 片段 1: 局域网直连 (必须最优先)
	rules = append(rules, map[string]any{"ip_is_private": true, "outbound": moduleUtils.TagDirect})

	// 片段 2: NTP 直连
	rules = append(rules, map[string]any{"protocol": []string{"ntp"}, "outbound": moduleUtils.TagDirect})

	// 片段 3: DNS 流量专门劫持 (在 TUN/Mixed 模式中，由 sing-box 本地解析)
	rules = append(rules, map[string]any{"protocol": []string{"dns"}, "action": "hijack-dns"})
	// 针对 ali dns 放行（因为在 DNS 模块中配置了国内直接去 ali 解析，避免循环）
	rules = append(rules, map[string]any{"ip_cidr": []string{"223.5.5.5/32", "223.6.6.6/32", "2400:3200::/32"}, "outbound": moduleUtils.TagDirect})

	// 片段 4: 去广告模块
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-ads",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "anti-ad",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/privacy-protection-tools/anti-ad.github.io/master/docs/anti-ad-sing-box.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-ads", "anti-ad"}, "outbound": moduleUtils.TagBlock})

	// 片段 5: 直连白名单
	rules = append(rules, map[string]any{
		"domain_suffix": []string{"wise.com", "schwab.com", "interactivebrokers.com", "cloudflare.com",
			"5e1f8y2z3l9.shop", "sky.money", "ethena.fi"},
		"outbound": moduleUtils.TagDirect,
	})

	// 片段 6: Apple 流量直连
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-apple",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-apple.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-apple"}, "outbound": moduleUtils.TagDirect})

	// 片段 7: 国内直连 (CN 路由分流)
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geosite-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	ruleSets = append(ruleSets, map[string]any{
		"tag":             "geoip-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
		"download_detour": moduleUtils.TagProxy,
	})
	rules = append(rules, map[string]any{"rule_set": []string{"geosite-cn", "geoip-cn"}, "outbound": moduleUtils.TagDirect})

	// 此时组合成一个整体 map 进行反序列化 (为了兼容 sing-box 的 rule 抽象类型)
	routeMap := map[string]any{
		"rule_set":              ruleSets,
		"rules":                 rules,
		"auto_detect_interface": true,
	}

	data, err := singboxjson.Marshal(routeMap)
	if err != nil {
		return err
	}

	var generatedRoute option.RouteOptions
	if err := singboxjson.Unmarshal(data, &generatedRoute); err != nil {
		return err
	}

	opts.Route.RuleSet = append(opts.Route.RuleSet, generatedRoute.RuleSet...)
	opts.Route.Rules = append(opts.Route.Rules, generatedRoute.Rules...)
	opts.Route.AutoDetectInterface = generatedRoute.AutoDetectInterface

	return nil
}
