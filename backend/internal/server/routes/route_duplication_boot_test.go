package routes

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func fillRouteHandlerPointers(v reflect.Value) {
	if v.Kind() != reflect.Ptr {
		return
	}
	if v.IsNil() {
		if !v.CanSet() {
			return
		}
		v.Set(reflect.New(v.Type().Elem()))
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		if field.Kind() == reflect.Ptr && field.CanSet() {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			fillRouteHandlerPointers(field)
		}
	}
}

func TestNativeRouterBootHasNoDuplicateResellerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlers := &handler.Handlers{}
	fillRouteHandlerPointers(reflect.ValueOf(handlers))
	noop := func(c *gin.Context) { c.Next() }

	router := gin.New()
	v1 := router.Group("/api/v1")
	require.NotPanics(t, func() {
		RegisterUserRoutes(v1, handlers,
			middleware.JWTAuthMiddleware(noop),
			middleware.AuditLogMiddleware(noop),
			nil,
			nil,
		)
		RegisterAdminRoutes(v1, handlers,
			middleware.AdminAuthMiddleware(noop),
			middleware.AuditLogMiddleware(noop),
			middleware.StepUpAuthMiddleware(noop),
			nil,
			nil,
		)
	})

	counts := make(map[string]int)
	for _, route := range router.Routes() {
		counts[route.Method+" "+route.Path]++
	}
	for _, expected := range []string{
		"GET /api/v1/user/reseller/role",
		"POST /api/v1/user/reseller/withdrawals/:id/cancel",
		"GET /api/v1/user/reseller/manager/agents",
		"DELETE /api/v1/user/reseller/manager/agents/:id/role",
		"GET /api/v1/admin/reseller/agents",
		"PATCH /api/v1/admin/reseller/agents/:id",
		"POST /api/v1/admin/reseller/withdrawals/:id/review",
	} {
		require.Equalf(t, 1, counts[expected], "route must be registered exactly once: %s", expected)
	}
}
