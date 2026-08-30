package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResellerAdminMutationsRejectAdminAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
		call func(*ResellerHandler, *gin.Context)
	}{
		{
			name: "grant role",
			path: "/api/v1/admin/reseller/agents/7/role",
			call: func(h *ResellerHandler, c *gin.Context) { h.AdminGrantRole(c) },
		},
		{
			name: "revoke role",
			path: "/api/v1/admin/reseller/agents/7/role",
			call: func(h *ResellerHandler, c *gin.Context) { h.AdminRevokeRole(c) },
		},
		{
			name: "review withdrawal",
			path: "/api/v1/admin/reseller/withdrawals/12/review",
			call: func(h *ResellerHandler, c *gin.Context) { h.AdminReviewWithdrawal(c) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)
			ctx.Params = gin.Params{{Key: "id", Value: "7"}}
			ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
			ctx.Set("auth_method", service.AuditAuthMethodAdminAPIKey)

			tt.call(&ResellerHandler{}, ctx)

			require.Equal(t, http.StatusForbidden, recorder.Code)
		})
	}
}
