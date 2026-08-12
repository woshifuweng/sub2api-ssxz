package admin

import "strings"

// 系统设置的非阻断配置提示。
//
// 这些提示只描述「保存成功、但该配置会让某个功能静默失效」的组合，
// 保存流程本身照常返回 2xx —— 管理员可以分步配置，不能因为某项还没填就拒绝保存。
//
// Code 是稳定的机器可读标识，也是本结构体对外的**全部**语义：
// 展示文案一律由前端按 code 从 i18n 取。后端不返回任何面向用户的文案 ——
// 返英文会让中文站显示英文（这正是本轮要修的问题之一），
// 返中文又把展示层职责搬进了 API。
const (
	// SettingWarningPasswordResetLinkUnresolvable 状态 A：密码重置**当前正在**静默失败。
	// 条件：重置生效（原始开关为真且邮箱验证已开）且 ResolvePasswordResetBaseURL 解析为空。
	// 后果：用户此刻点重置就收不到邮件，页面还显示发送成功。
	SettingWarningPasswordResetLinkUnresolvable = "PASSWORD_RESET_LINK_UNRESOLVABLE"
	// SettingWarningPasswordResetLinkUnresolvableLatent 状态 B：潜伏，尚未造成影响。
	// 条件：原始开关为真但邮箱验证关闭（生效值被取与成 false），且解析为空。
	// 此刻重置接口直接 403，**没有任何邮件在丢**；一旦开启邮箱验证就会立刻变成状态 A。
	// 与状态 A 分开是因为用同一句「用户收不到邮件」描述状态 B 是假的。
	SettingWarningPasswordResetLinkUnresolvableLatent = "PASSWORD_RESET_LINK_UNRESOLVABLE_LATENT"
	// SettingWarningContactInfoEmpty 未配置联系方式。
	// 后果：被封禁/被锁定的用户没有任何申诉入口。
	SettingWarningContactInfoEmpty = "CONTACT_INFO_EMPTY"
)

// SettingConfigWarning 单条非阻断配置提示。
type SettingConfigWarning struct {
	// Code 机器可读标识，见上方常量。前端据此渲染本地化文案。
	Code string `json:"code"`
	// Field 触发提示的设置字段名（与请求/响应中的 JSON 字段一致），便于前端定位输入框。
	Field string `json:"field"`
}

// settingConfigWarningInput 计算提示所需的最小输入，独立于 DTO 以便单测直接构造。
type settingConfigWarningInput struct {
	// PasswordResetStored 取 password_reset_enabled 的**原始存储值**（未与
	// email_verify_enabled 取与），这样「邮箱验证关 + 密码重置开」也能被观测到。
	PasswordResetStored bool
	// EmailVerifyEnabled 决定重置是否真正生效：生效值 = PasswordResetStored && EmailVerifyEnabled。
	// 用它来区分状态 A（正在失败）与状态 B（潜伏）。
	EmailVerifyEnabled bool
	// ResolvedResetLinkBase 是 service.ResolvePasswordResetBaseURL 的结果，即重置邮件
	// 实际会用的链接基址。为空表示解析不出来 —— 这是唯一可信的判据，
	// 不要拿 settings.frontend_url 原始值代替（它漏掉了配置文件回落与 Origin 兜底）。
	ResolvedResetLinkBase string
	ContactInfo           string
}

// collectSettingConfigWarnings 汇总命中的非阻断提示；无命中时返回空切片（非 nil），
// 保证响应里的 warnings 始终序列化为 JSON 数组，前端不必区分 null 与 []。
func collectSettingConfigWarnings(in settingConfigWarningInput) []SettingConfigWarning {
	warnings := make([]SettingConfigWarning, 0, 2)

	if in.PasswordResetStored && strings.TrimSpace(in.ResolvedResetLinkBase) == "" {
		code := SettingWarningPasswordResetLinkUnresolvableLatent
		if in.EmailVerifyEnabled {
			code = SettingWarningPasswordResetLinkUnresolvable
		}
		warnings = append(warnings, SettingConfigWarning{
			Code:  code,
			Field: "frontend_url",
		})
	}

	if strings.TrimSpace(in.ContactInfo) == "" {
		warnings = append(warnings, SettingConfigWarning{
			Code:  SettingWarningContactInfoEmpty,
			Field: "contact_info",
		})
	}

	return warnings
}
