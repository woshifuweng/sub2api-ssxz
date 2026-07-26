//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// UpdateSettings 的非阻断配置提示：命中时保存仍然成功（HTTP 200 + code 0），
// 只在成功响应里追加 warnings 字段。

func newSettingWarningTestHandler(t *testing.T, stored map[string]string) *SettingHandler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: stored}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	return NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
}

// updateSettingsWarnings 执行一次保存并解析响应中的 warnings。
// 返回 recorder 供断言状态码，warnings 可能为空切片。
func updateSettingsWarnings(t *testing.T, h *SettingHandler, body map[string]any) (*httptest.ResponseRecorder, []SettingConfigWarning) {
	t.Helper()
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateSettings(c)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok, "response data should be an object")

	rawWarnings, ok := data["warnings"]
	require.True(t, ok, "warnings field must always be present")
	encoded, err := json.Marshal(rawWarnings)
	require.NoError(t, err)
	var warnings []SettingConfigWarning
	require.NoError(t, json.Unmarshal(encoded, &warnings))
	return rec, warnings
}

func warningCodes(warnings []SettingConfigWarning) []string {
	codes := make([]string, 0, len(warnings))
	for _, w := range warnings {
		codes = append(codes, w.Code)
	}
	return codes
}

// 命中：密码重置开启但 frontend_url 为空，且 contact_info 为空 —— 两条提示都要回，
// 且保存本身必须成功（非阻断）。
func TestUpdateSettingsReturnsConfigWarningsWithoutBlockingSave(t *testing.T) {
	h := newSettingWarningTestHandler(t, map[string]string{})

	rec, warnings := updateSettingsWarnings(t, h, map[string]any{
		"email_verify_enabled":   true, // 状态 A：重置真正生效，此刻正在丢邮件
		"password_reset_enabled": true,
		"frontend_url":           "",
		"contact_info":           "",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.ElementsMatch(t, []string{
		SettingWarningPasswordResetLinkUnresolvable,
		SettingWarningContactInfoEmpty,
	}, warningCodes(warnings))

	// 契约：只回 code + field，不回任何面向用户的文案（文案属于前端 i18n）。
	for _, w := range warnings {
		switch w.Code {
		case SettingWarningPasswordResetLinkUnresolvable:
			require.Equal(t, "frontend_url", w.Field)
		case SettingWarningContactInfoEmpty:
			require.Equal(t, "contact_info", w.Field)
		}
	}
}

// 状态 B（潜伏）：email_verify_enabled=false 时生效值被取与成 false，重置接口直接 403、
// 没有任何邮件在丢。提示仍要基于原始存储值给出，但必须用 LATENT 码 ——
// 套用状态 A 的「用户收不到邮件」在这里是假的。
func TestUpdateSettingsWarnsWhenPasswordResetOnWithEmailVerifyOff(t *testing.T) {
	h := newSettingWarningTestHandler(t, map[string]string{})

	rec, warnings := updateSettingsWarnings(t, h, map[string]any{
		"email_verify_enabled":   false,
		"password_reset_enabled": true,
		"frontend_url":           "",
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{SettingWarningPasswordResetLinkUnresolvableLatent}, warningCodes(warnings))
}

// 不命中：frontend_url 已配置且 contact_info 非空 —— warnings 为空数组，不是 null。
func TestUpdateSettingsReturnsNoConfigWarningsWhenConfigured(t *testing.T) {
	h := newSettingWarningTestHandler(t, map[string]string{})

	rec, warnings := updateSettingsWarnings(t, h, map[string]any{
		"email_verify_enabled":   true,
		"password_reset_enabled": true,
		"frontend_url":           "https://console.example.com",
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, warnings)
	require.Contains(t, rec.Body.String(), `"warnings":[]`)
}

// 不命中：密码重置关闭时不因 frontend_url 为空报警（只有 contact_info 那条）。
func TestUpdateSettingsSkipsFrontendURLWarningWhenPasswordResetDisabled(t *testing.T) {
	h := newSettingWarningTestHandler(t, map[string]string{})

	rec, warnings := updateSettingsWarnings(t, h, map[string]any{
		"password_reset_enabled": false,
		"frontend_url":           "",
		"contact_info":           "",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{SettingWarningContactInfoEmpty}, warningCodes(warnings))
}

// 不命中：DB 未配 frontend_url 但配置文件已配时走 fallback，不应误报。
func TestUpdateSettingsSkipsFrontendURLWarningWhenConfigFileProvidesFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{
		Default: config.DefaultConfig{UserConcurrency: 5},
		Server:  config.ServerConfig{FrontendURL: "https://fallback.example.com"},
	})
	h := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	rec, warnings := updateSettingsWarnings(t, h, map[string]any{
		"password_reset_enabled": true,
		"frontend_url":           "",
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, warnings)
}

// 纯函数层面的边界：仅空白字符等同未配置。
func TestCollectSettingConfigWarningsTreatsBlankAsEmpty(t *testing.T) {
	warnings := collectSettingConfigWarnings(settingConfigWarningInput{
		PasswordResetStored:   true,
		EmailVerifyEnabled:    true,
		ResolvedResetLinkBase: "   ",
		ContactInfo:           "\t\n",
	})

	require.ElementsMatch(t, []string{
		SettingWarningPasswordResetLinkUnresolvable,
		SettingWarningContactInfoEmpty,
	}, warningCodes(warnings))
}
