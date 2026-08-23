package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeRegistrationLimiter struct {
	calls  []string
	denyAt int
	errAt  int
}

func (f *fakeRegistrationLimiter) Allow(_ context.Context, key string, _ int, _ time.Duration) (appmiddleware.AllowResult, error) {
	f.calls = append(f.calls, key)
	call := len(f.calls)
	if f.errAt == call {
		return appmiddleware.AllowResult{}, errors.New("redis unavailable")
	}
	if f.denyAt == call {
		return appmiddleware.AllowResult{Allowed: false, RetryAfter: time.Minute}, nil
	}
	return appmiddleware.AllowResult{Allowed: true}, nil
}

func TestRegistrationRateLimitKeysAreStableAndDoNotContainPII(t *testing.T) {
	keysA := registrationRateLimitKeys("  Alice.Example@Example.COM ")
	keysB := registrationRateLimitKeys("alice.example@example.com")

	require.Equal(t, keysA, keysB)
	require.Len(t, keysA, 3)
	require.Equal(t, "auth-register-global", keysA[0])
	for _, key := range keysA[1:] {
		require.NotContains(t, strings.ToLower(key), "alice")
		require.NotContains(t, strings.ToLower(key), "example.com")
	}
}

func TestAllowRegistrationAttemptAppliesAllThreeBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	limiter := &fakeRegistrationLimiter{}
	h := &AuthHandler{registrationLimiter: limiter}

	require.True(t, h.allowRegistrationAttempt(gatewayctx.FromGin(ctx), "alice@example.com"))
	require.Len(t, limiter.calls, 3)
	require.False(t, ctx.IsAborted())
}

func TestAllowRegistrationAttemptFailsClosedOnLimitOrRedisError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		denyAt int
		errAt  int
	}{
		{name: "limit exceeded", denyAt: 2},
		{name: "redis error", errAt: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
			limiter := &fakeRegistrationLimiter{denyAt: tc.denyAt, errAt: tc.errAt}
			h := &AuthHandler{registrationLimiter: limiter}

			require.False(t, h.allowRegistrationAttempt(gatewayctx.FromGin(ctx), "alice@example.com"))
			require.True(t, ctx.IsAborted())
			require.Equal(t, http.StatusTooManyRequests, recorder.Code)
			require.NotContains(t, recorder.Body.String(), "alice@example.com")
		})
	}
}
