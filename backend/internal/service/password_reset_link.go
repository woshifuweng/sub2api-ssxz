package service

import (
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// ginReleaseMode 与 gin.ReleaseMode 取值一致。此处用字面量而不是 import gin，
// 是为了不把 web 框架依赖引进 service 层。
const ginReleaseMode = "release"

// ResolvePasswordResetBaseURL 解析密码重置链接的 base URL，是该问题的**唯一入口**。
//
// 解析顺序：
//  1. configuredURL（DB 的 settings.frontend_url，为空时回落到 config.yaml 的 server.frontend_url）
//     非空即直接采用，与请求 Origin 无关；
//  2. 否则尝试用请求 Origin 兜底：Origin 必须是合法的绝对 http(s) URL 且不含
//     用户名/路径/查询/片段；
//  3. release 模式下额外要求 Origin 精确出现在 cfg.CORS.AllowedOrigins 中
//     （且不接受通配符 "*"），否则返回空串。
//
// 返回空串意味着**重置邮件无法生成链接**，调用方会跳过发信。生产上
// frontend_url 为空且 cors.allowed_origins 也为空时，该函数恒返回空串 ——
// 这正是密码重置静默失效的根因，因此管理端设置页需要复用本函数来判断配置是否可用，
// 而不是自行复刻这套回落规则（复刻必然漂移，等于制造第二个事实来源）。
func ResolvePasswordResetBaseURL(configuredURL, requestOrigin string, cfg *config.Config) string {
	if configuredURL = strings.TrimSpace(configuredURL); configuredURL != "" {
		return configuredURL
	}

	requestOrigin = strings.TrimSpace(requestOrigin)
	if cfg == nil || config.ValidateAbsoluteHTTPURL(requestOrigin) != nil {
		return ""
	}

	originURL, err := url.Parse(requestOrigin)
	if err != nil || originURL.User != nil || originURL.Opaque != "" || originURL.Path != "" ||
		originURL.RawPath != "" || originURL.RawQuery != "" || originURL.ForceQuery || originURL.Fragment != "" ||
		strings.TrimSpace(originURL.Hostname()) == "" {
		return ""
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Server.Mode), ginReleaseMode) {
		for _, allowedOrigin := range cfg.CORS.AllowedOrigins {
			if strings.TrimSpace(allowedOrigin) == requestOrigin && requestOrigin != "*" {
				return requestOrigin
			}
		}
		return ""
	}

	return requestOrigin
}
