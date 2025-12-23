package updater

const(
	// 使用 SagerNet 官方发布的资源
	// 注意：如果你在国内，可能需要配置 HTTP_PROXY 环境变量才能顺利下载
	GeoIPURL   = "https://github.com/SagerNet/sing-geoip/releases/latest/download/geoip.db"
	GeoSiteURL = "https://github.com/SagerNet/sing-geosite/releases/latest/download/geosite.db"

	GeoIPFilename   = "geoip.db"
	GeoSiteFilename = "geosite.db"
)