package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubStepUpGrantChecker struct {
	granted bool
	err     error
}

func (s stubStepUpGrantChecker) HasStepUpGrant(ctx context.Context, userID int64, sessionKey string) (bool, error) {
	return s.granted, s.err
}

type stubStepUpUserReader struct {
	user *service.User
	err  error
}

func (s stubStepUpUserReader) GetByID(ctx context.Context, id int64) (*service.User, error) {
	return s.user, s.err
}

type stubStepUpSettingReader struct {
	enabled bool
}

func (s stubStepUpSettingReader) IsStepUpEnabled(ctx context.Context) bool {
	return s.enabled
}

// stepUpEnabled 功能开关开启的设置桩，供既有门控分支测试使用。
var stepUpEnabled = stubStepUpSettingReader{enabled: true}

func newStepUpTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/sensitive", nil)
	return c, rec
}

func TestEnforceStepUpRejectsAdminAPIKey(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set("auth_method", service.AuditAuthMethodAdminAPIKey)

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: true}, stubStepUpUserReader{user: &service.User{TotpEnabled: true}}, stepUpEnabled)

	require.False(t, ok)
	require.True(t, c.IsAborted())
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_ADMIN_API_KEY_FORBIDDEN")
}

func TestEnforceStepUpDisabledRejectsAdminAPIKey(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set("auth_method", service.AuditAuthMethodAdminAPIKey)

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: false}, stubStepUpUserReader{err: errors.New("should not be called")}, stubStepUpSettingReader{enabled: false})

	require.False(t, ok)
	require.True(t, c.IsAborted())
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_ADMIN_API_KEY_FORBIDDEN")
}

func TestEnforceStepUpGatewayDisabledRejectsAdminAPIKey(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set("auth_method", service.AuditAuthMethodAdminAPIKey)
	settings := service.NewSettingService(stepUpSettingRepoStub{values: map[string]string{
		service.SettingKeyStepUpEnabled: "false",
	}}, nil)

	ok := EnforceStepUpGateway(gatewayctx.FromGin(c), nil, nil, settings)

	require.False(t, ok)
	require.True(t, c.IsAborted())
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_ADMIN_API_KEY_FORBIDDEN")
}

func TestEnforceStepUpRequiresAuthSubject(t *testing.T) {
	c, rec := newStepUpTestContext(t)

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: true}, stubStepUpUserReader{user: &service.User{TotpEnabled: true}}, stepUpEnabled)

	require.False(t, ok)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestEnforceStepUpRequiresTotpEnabled(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: true}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: false}}, stepUpEnabled)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_TOTP_NOT_ENABLED")
}

func TestEnforceStepUpFailsClosedOnGrantError(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

	ok := enforceStepUp(c, stubStepUpGrantChecker{err: errors.New("redis down")}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: true}}, stepUpEnabled)

	require.False(t, ok)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_UNAVAILABLE")
}

func TestEnforceStepUpRequiresGrant(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: false}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: true}}, stepUpEnabled)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_REQUIRED")
}

func TestEnforceStepUpPassesWithGrant(t *testing.T) {
	c, _ := newStepUpTestContext(t)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: true}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: true}}, stepUpEnabled)

	require.True(t, ok)
	require.False(t, c.IsAborted())
}

// 功能开关关闭时，普通用户会话跳过 step-up 检查，但机器凭证仍被拒绝。
func TestEnforceStepUpDisabledSkipsUserSessionChecks(t *testing.T) {
	disabled := stubStepUpSettingReader{enabled: false}

	t.Run("no totp, no grant", func(t *testing.T) {
		c, _ := newStepUpTestContext(t)
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

		ok := enforceStepUp(c, stubStepUpGrantChecker{granted: false}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: false}}, disabled)

		require.True(t, ok)
		require.False(t, c.IsAborted())
	})

}

type stepUpSettingRepoStub struct {
	values map[string]string
}

func (r stepUpSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	return nil, errors.New("not implemented")
}

func (r stepUpSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (r stepUpSettingRepoStub) Set(ctx context.Context, key, value string) error {
	return errors.New("not implemented")
}

func (r stepUpSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (r stepUpSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	return errors.New("not implemented")
}

func (r stepUpSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (r stepUpSettingRepoStub) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}

// settings 为 nil 时保持门控（fail-closed），避免装配缺陷静默关闭安全控制。
func TestEnforceStepUpNilSettingsFailsClosed(t *testing.T) {
	c, rec := newStepUpTestContext(t)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})

	ok := enforceStepUp(c, stubStepUpGrantChecker{granted: false}, stubStepUpUserReader{user: &service.User{ID: 1, TotpEnabled: true}}, nil)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "STEP_UP_REQUIRED")
}

// EnforceStepUp 收到 nil *service.SettingService 时不得因 typed-nil 装箱绕过门控：
// 未认证请求仍应被拦截（401），而不是当作"开关关闭"放行。
func TestEnforceStepUpTypedNilSettingServiceFailsClosed(t *testing.T) {
	require.Nil(t, stepUpSettingsOrNil(nil))

	c, rec := newStepUpTestContext(t)

	ok := EnforceStepUp(c, nil, nil, nil)

	require.False(t, ok)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
