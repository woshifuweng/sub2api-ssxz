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

// password_reset_enabled 在 parseSettings 里与 email_verify_enabled 取与，
// 所以响应里的 password_reset_enabled 无法表达「配置开着但当前未生效」。
// password_reset_enabled_stored 是这一状态的唯一观测点：GET 与 PUT 两条路径都必须带上，
// 否则管理台只能在「邮箱验证开启」时才看得见空 frontend_url 的隐患。

func newPasswordResetStoredHandler(t *testing.T, stored map[string]string) (*SettingHandler, *settingHandlerRepoStub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: stored}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	return NewSettingHandler(svc, nil, nil, nil, nil, nil, nil), repo
}

func decodeSettingsResponseData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok, "response data should be an object")
	return data
}

func getSettingsData(t *testing.T, h *SettingHandler) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	h.GetSettings(c)
	return rec, decodeSettingsResponseData(t, rec)
}

func putSettingsData(t *testing.T, h *SettingHandler, body map[string]any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")
	h.UpdateSettings(c)
	return rec, decodeSettingsResponseData(t, rec)
}

// 状态 B（潜伏）：DB 里 password_reset_enabled=true，但 email_verify_enabled=false。
// 生效值被取与成 false，stored 必须仍然是 true —— 这是坑 A 的唯一可观测判据。
func TestGetSettings_ExposesStoredPasswordResetWhenEmailVerifyDisabled(t *testing.T) {
	h, _ := newPasswordResetStoredHandler(t, map[string]string{
		service.SettingKeyEmailVerifyEnabled:   "false",
		service.SettingKeyPasswordResetEnabled: "true",
		service.SettingKeyFrontendURL:          "",
	})

	rec, data := getSettingsData(t, h)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, false, data["email_verify_enabled"])
	require.Equal(t, false, data["password_reset_enabled"], "生效值必须保持取与语义不变")
	require.Equal(t, true, data["password_reset_enabled_stored"], "原始存储值必须原样透出")
}

// 状态 A（正在静默失败）：两个开关都开着，stored 与生效值一致。
func TestGetSettings_StoredMatchesEffectiveWhenEmailVerifyEnabled(t *testing.T) {
	h, _ := newPasswordResetStoredHandler(t, map[string]string{
		service.SettingKeyEmailVerifyEnabled:   "true",
		service.SettingKeyPasswordResetEnabled: "true",
	})

	rec, data := getSettingsData(t, h)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, true, data["password_reset_enabled"])
	require.Equal(t, true, data["password_reset_enabled_stored"])
}

// 未开启时 stored 也必须是 false，避免前端把「没配」误判成「配了没生效」。
func TestGetSettings_StoredFalseWhenPasswordResetNeverEnabled(t *testing.T) {
	h, _ := newPasswordResetStoredHandler(t, map[string]string{
		service.SettingKeyEmailVerifyEnabled:   "true",
		service.SettingKeyPasswordResetEnabled: "false",
	})

	rec, data := getSettingsData(t, h)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, false, data["password_reset_enabled"])
	require.Equal(t, false, data["password_reset_enabled_stored"])
}

// PUT 必须与 GET 对称：保存后立刻能从响应里读到落库的原始值，
// 前端不必为了刷新告警状态再发一次 GET。
func TestUpdateSettings_ReturnsStoredPasswordResetWhenEmailVerifyDisabled(t *testing.T) {
	h, repo := newPasswordResetStoredHandler(t, map[string]string{})

	rec, data := putSettingsData(t, h, map[string]any{
		"email_verify_enabled":   false,
		"password_reset_enabled": true,
		"frontend_url":           "",
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyPasswordResetEnabled], "原始值必须真的落库")
	require.Equal(t, false, data["password_reset_enabled"], "生效值仍与 email_verify_enabled 取与")
	require.Equal(t, true, data["password_reset_enabled_stored"])
}

// 保存后 stored 反映的是刚落库的值，而不是请求前的旧值。
func TestUpdateSettings_StoredReflectsNewlyPersistedValue(t *testing.T) {
	h, _ := newPasswordResetStoredHandler(t, map[string]string{
		service.SettingKeyEmailVerifyEnabled:   "true",
		service.SettingKeyPasswordResetEnabled: "true",
	})

	rec, data := putSettingsData(t, h, map[string]any{
		"email_verify_enabled":   true,
		"password_reset_enabled": false,
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, false, data["password_reset_enabled_stored"])
}

// warnings 与 password_reset_enabled_stored 必须同源：
// 状态 B 下既要给出 stored=true，也要给出 frontend_url 缺失的告警，两者不能互相打架。
func TestUpdateSettings_StoredAndWarningsAgreeInLatentState(t *testing.T) {
	h, _ := newPasswordResetStoredHandler(t, map[string]string{})

	rec, data := putSettingsData(t, h, map[string]any{
		"email_verify_enabled":   false,
		"password_reset_enabled": true,
		"frontend_url":           "",
		"contact_info":           "support@example.com",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, true, data["password_reset_enabled_stored"])

	encoded, err := json.Marshal(data["warnings"])
	require.NoError(t, err)
	var warnings []SettingConfigWarning
	require.NoError(t, json.Unmarshal(encoded, &warnings))
	require.Equal(t, []string{SettingWarningPasswordResetMissingFrontendURL}, warningCodes(warnings))
}
