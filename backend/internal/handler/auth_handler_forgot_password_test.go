//go:build unit

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

const forgotPasswordSuccessMessage = "If your email is registered, you will receive a password reset link shortly."

func TestResolvePasswordResetBaseURL(t *testing.T) {
	releaseCfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"https://app.example.com", "http://localhost:5173"},
		},
	}

	t.Run("configured URL takes precedence", func(t *testing.T) {
		got := resolvePasswordResetBaseURL(
			" https://configured.example.com/app/ ",
			"https://evil.example.com",
			releaseCfg,
		)
		require.Equal(t, "https://configured.example.com/app/", got)
	})

	t.Run("release accepts explicitly allowed Origin", func(t *testing.T) {
		require.Equal(t, "https://app.example.com", resolvePasswordResetBaseURL(
			"",
			"https://app.example.com",
			releaseCfg,
		))
	})

	t.Run("non-release accepts a validated Origin", func(t *testing.T) {
		debugCfg := &config.Config{Server: config.ServerConfig{Mode: "debug"}}
		require.Equal(t, "http://localhost:3000", resolvePasswordResetBaseURL(
			"",
			"http://localhost:3000",
			debugCfg,
		))
	})

	tests := []struct {
		name   string
		origin string
		cfg    *config.Config
	}{
		{name: "missing Origin", origin: "", cfg: releaseCfg},
		{name: "non HTTP scheme", origin: "javascript://app.example.com", cfg: releaseCfg},
		{name: "missing host", origin: "https://", cfg: releaseCfg},
		{name: "userinfo", origin: "https://attacker@app.example.com", cfg: releaseCfg},
		{name: "path is not an Origin", origin: "https://app.example.com/reset", cfg: releaseCfg},
		{name: "query is not an Origin", origin: "https://app.example.com?next=evil", cfg: releaseCfg},
		{name: "release rejects unlisted Origin", origin: "https://evil.example.com", cfg: releaseCfg},
		{
			name:   "release does not treat CORS wildcard as trusted",
			origin: "https://evil.example.com",
			cfg: &config.Config{
				Server: config.ServerConfig{Mode: "release"},
				CORS:   config.CORSConfig{AllowedOrigins: []string{"*"}},
			},
		},
		{name: "missing config fails closed", origin: "https://app.example.com", cfg: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Empty(t, resolvePasswordResetBaseURL("", tt.origin, tt.cfg))
		})
	}
}

type forgotPasswordHarness struct {
	handler *AuthHandler
}

func newForgotPasswordHarness(t *testing.T, frontendURL string, allowedOrigins []string, withQueue bool) *forgotPasswordHarness {
	t.Helper()

	existing := &service.User{
		ID:     1,
		Email:  "existing@example.com",
		Role:   service.RoleUser,
		Status: service.StatusActive,
	}
	userRepo := &authHandlerUserRepoStub{
		usersByID:    map[int64]*service.User{existing.ID: existing},
		usersByEmail: map[string]*service.User{existing.Email: existing},
	}
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		CORS:   config.CORSConfig{AllowedOrigins: allowedOrigins},
	}
	settingRepo := &authHandlerSettingRepoStub{values: map[string]string{
		service.SettingKeyEmailVerifyEnabled:   "true",
		service.SettingKeyPasswordResetEnabled: "true",
		service.SettingKeyFrontendURL:          frontendURL,
	}}
	settingSvc := service.NewSettingService(settingRepo, cfg)

	var emailQueue *service.EmailQueueService
	if withQueue {
		emailQueue = service.NewEmailQueueServiceWithAutoStart(nil, 1, false)
	}
	authSvc := service.NewAuthService(nil, userRepo, nil, nil, cfg, settingSvc, nil, nil, emailQueue, nil, nil)

	return &forgotPasswordHarness{
		handler: NewAuthHandler(cfg, authSvc, nil, settingSvc, nil, nil, nil, nil),
	}
}

func runForgotPasswordRequest(t *testing.T, handler *AuthHandler, email, origin string, extraHeaders map[string]string) (int, authHandlerResponseEnvelope) {
	t.Helper()

	body, err := json.Marshal(ForgotPasswordRequest{Email: email})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	for key, value := range extraHeaders {
		if key == "Host" {
			req.Host = value
			continue
		}
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	ctx := gatewayctx.NewNative(req, rec, nil, req.RemoteAddr)

	handler.ForgotPasswordGateway(ctx)

	var envelope authHandlerResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return rec.Code, envelope
}

func TestForgotPasswordGateway_ConfiguredURLAndTrustedOriginDoNotEnumerateAccounts(t *testing.T) {
	tests := []struct {
		name        string
		frontendURL string
		origin      string
	}{
		{
			name:        "configured URL",
			frontendURL: "https://configured.example.com",
			origin:      "https://evil.example.com",
		},
		{
			name:   "trusted Origin fallback",
			origin: "https://app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			harness := newForgotPasswordHarness(t, tt.frontendURL, []string{"https://app.example.com"}, true)

			existingCode, existingResponse := runForgotPasswordRequest(t, harness.handler, "existing@example.com", tt.origin, nil)
			missingCode, missingResponse := runForgotPasswordRequest(t, harness.handler, "missing@example.com", tt.origin, nil)

			require.Equal(t, http.StatusOK, existingCode)
			require.Equal(t, existingCode, missingCode)
			require.Equal(t, existingResponse, missingResponse)
			require.Contains(t, string(existingResponse.Data), forgotPasswordSuccessMessage)
		})
	}
}

func TestForgotPasswordGateway_UntrustedOrMissingOriginDegradesToGenericSuccess(t *testing.T) {
	harness := newForgotPasswordHarness(t, "", []string{"https://app.example.com"}, false)
	tests := []struct {
		name         string
		origin       string
		extraHeaders map[string]string
	}{
		{name: "missing Origin"},
		{name: "malicious Origin", origin: "https://attacker@app.example.com"},
		{name: "unlisted Origin", origin: "https://evil.example.com"},
		{
			name: "Host headers are not fallback sources",
			extraHeaders: map[string]string{
				"Host":             "attacker.example.com",
				"X-Forwarded-Host": "attacker.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingCode, existingResponse := runForgotPasswordRequest(t, harness.handler, "existing@example.com", tt.origin, tt.extraHeaders)
			missingCode, missingResponse := runForgotPasswordRequest(t, harness.handler, "missing@example.com", tt.origin, tt.extraHeaders)

			require.Equal(t, http.StatusOK, existingCode)
			require.Equal(t, existingCode, missingCode)
			require.Equal(t, existingResponse, missingResponse)
			require.Contains(t, string(existingResponse.Data), forgotPasswordSuccessMessage)
		})
	}
}
