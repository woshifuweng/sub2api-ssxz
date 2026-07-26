package admin

import "strings"

// 系统设置的非阻断配置提示。
//
// 这些提示只描述「保存成功、但该配置会让某个功能静默失效」的组合，
// 保存流程本身照常返回 2xx —— 管理员可以分步配置，不能因为某项还没填就拒绝保存。
//
// Code 是稳定的机器可读标识：前端按 code 渲染本地化文案，
// 不要去匹配 Message 文本（Message 仅作为无本地化时的兜底）。
const (
	// SettingWarningPasswordResetMissingFrontendURL 密码重置已开启但前端地址为空。
	// 后果：ForgotPassword 解析不出重置链接基址时会静默跳过发信并回通用成功文案
	// （防账号枚举），用户永远收不到重置邮件，页面还显示成功。
	SettingWarningPasswordResetMissingFrontendURL = "PASSWORD_RESET_MISSING_FRONTEND_URL"
	// SettingWarningContactInfoEmpty 未配置联系方式。
	// 后果：被封禁/被锁定的用户没有任何申诉入口。
	SettingWarningContactInfoEmpty = "CONTACT_INFO_EMPTY"
)

// SettingConfigWarning 单条非阻断配置提示。
type SettingConfigWarning struct {
	// Code 机器可读标识，见上方常量。
	Code string `json:"code"`
	// Field 触发提示的设置字段名（与请求/响应中的 JSON 字段一致），便于前端定位输入框。
	Field string `json:"field"`
	// Message 英文兜底文案，说明后果而不仅仅是「未配置」。
	Message string `json:"message"`
}

// settingConfigWarningInput 计算提示所需的最小输入，独立于 DTO 以便单测直接构造。
type settingConfigWarningInput struct {
	// PasswordResetEnabled 取管理员本次保存的原始意图（未与 email_verify_enabled 取与），
	// 这样「邮件验证关 + 密码重置开」这种组合也能拿到提示。
	PasswordResetEnabled bool
	// FrontendURL 取生效值（DB 优先、fallback 配置文件），而非请求原始值，
	// 避免配置文件里已经配好时误报。
	FrontendURL string
	ContactInfo string
}

// collectSettingConfigWarnings 汇总命中的非阻断提示；无命中时返回空切片（非 nil），
// 保证响应里的 warnings 始终序列化为 JSON 数组，前端不必区分 null 与 []。
func collectSettingConfigWarnings(in settingConfigWarningInput) []SettingConfigWarning {
	warnings := make([]SettingConfigWarning, 0, 2)

	if in.PasswordResetEnabled && strings.TrimSpace(in.FrontendURL) == "" {
		warnings = append(warnings, SettingConfigWarning{
			Code:  SettingWarningPasswordResetMissingFrontendURL,
			Field: "frontend_url",
			Message: "Password reset is enabled but the frontend URL is not configured. " +
				"Reset emails are skipped silently, so users will never receive a reset link " +
				"even though the page reports success.",
		})
	}

	if strings.TrimSpace(in.ContactInfo) == "" {
		warnings = append(warnings, SettingConfigWarning{
			Code:  SettingWarningContactInfoEmpty,
			Field: "contact_info",
			Message: "Contact info is not configured. Users who are locked out or blocked have " +
				"no way to reach support.",
		})
	}

	return warnings
}
