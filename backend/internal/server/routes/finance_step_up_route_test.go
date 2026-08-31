//go:build unit

package routes

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestFinanceMutationRoutesRequireStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlers := &handler.Handlers{}
	fillRouteHandlerPointers(reflect.ValueOf(handlers))

	pass := func(c *gin.Context) { c.Next() }
	stepUpMarker := func(c *gin.Context) { c.AbortWithStatus(428) }
	router := gin.New()
	router.Use(gin.Recovery())
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		handlers,
		middleware.JWTAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		middleware.StepUpAuthMiddleware(stepUpMarker),
		nil,
		nil,
	)
	RegisterAdminRoutes(
		v1,
		handlers,
		middleware.AdminAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		middleware.StepUpAuthMiddleware(stepUpMarker),
		nil,
		nil,
	)
	RegisterPaymentRoutes(
		v1,
		handlers.Payment,
		handlers.PaymentWebhook,
		handlers.Admin.Payment,
		middleware.JWTAuthMiddleware(pass),
		middleware.AdminAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		middleware.StepUpAuthMiddleware(stepUpMarker),
		nil,
		nil,
	)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/user/aff/transfer"},
		{http.MethodPost, "/api/v1/user/reseller/withdrawals"},
		{http.MethodPost, "/api/v1/payment/orders/1/refund-request"},
		{http.MethodPost, "/api/v1/admin/users/1/balance"},
		{http.MethodPost, "/api/v1/admin/redeem-codes/generate"},
		{http.MethodPost, "/api/v1/admin/redeem-codes/create-and-redeem"},
		{http.MethodPost, "/api/v1/admin/affiliates/users/batch-rate"},
		{http.MethodPut, "/api/v1/admin/affiliates/users/1"},
		{http.MethodDelete, "/api/v1/admin/affiliates/users/1"},
		{http.MethodPatch, "/api/v1/admin/reseller/agents/1"},
		{http.MethodPost, "/api/v1/admin/reseller/agents/1/disable"},
		{http.MethodPost, "/api/v1/admin/reseller/agents/1/enable"},
		{http.MethodPost, "/api/v1/admin/reseller/agents/1/role"},
		{http.MethodDelete, "/api/v1/admin/reseller/agents/1/role"},
		{http.MethodPost, "/api/v1/admin/reseller/withdrawals/1/review"},
		{http.MethodPut, "/api/v1/admin/settings"},
		{http.MethodPut, "/api/v1/admin/payment/config"},
		{http.MethodPost, "/api/v1/admin/payment/orders/1/cancel"},
		{http.MethodPost, "/api/v1/admin/payment/orders/1/retry"},
		{http.MethodPost, "/api/v1/admin/payment/orders/1/refund"},
		{http.MethodPost, "/api/v1/admin/payment/orders/1/refund/query"},
		{http.MethodPost, "/api/v1/admin/payment/plans"},
		{http.MethodPut, "/api/v1/admin/payment/plans/1"},
		{http.MethodDelete, "/api/v1/admin/payment/plans/1"},
		{http.MethodPost, "/api/v1/admin/payment/providers"},
		{http.MethodPut, "/api/v1/admin/payment/providers/1"},
		{http.MethodDelete, "/api/v1/admin/payment/providers/1"},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			require.Equal(t, 428, recorder.Code)
		})
	}
}
