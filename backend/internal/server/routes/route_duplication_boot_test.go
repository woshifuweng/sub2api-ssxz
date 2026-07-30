package routes

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fillPointers recursively allocates every nil pointer field in the value
// pointed to by v, so that the route-registration guards (which skip modules
// whose handler is nil) all fire and every route is actually registered.
func fillPointers(v reflect.Value) {
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
		f := elem.Field(i)
		if f.Kind() == reflect.Ptr && f.CanSet() {
			if f.IsNil() {
				f.Set(reflect.New(f.Type().Elem()))
			}
			fillPointers(f)
		}
	}
}

// TestNativeRouterBootHasNoDuplicateRoutes boots the full native gin router the
// same way production does. gin panics on a duplicate/conflicting route, so a
// clean boot proves there are no collisions across all Register*Routes.
func TestNativeRouterBootHasNoDuplicateRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &handler.Handlers{}
	fillPointers(reflect.ValueOf(h))

	noop := func(c *gin.Context) { c.Next() }

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("native router boot panicked (duplicate/conflicting route): %v", r)
		}
	}()

	r := gin.New()
	v1 := r.Group("/api/v1")

	RegisterUserRoutes(v1, h,
		middleware.JWTAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		nil,
		nil)
	RegisterAdminRoutes(v1, h,
		middleware.AdminAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		middleware.StepUpAuthMiddleware(noop),
		nil,
		nil)

	registered := make(map[string]struct{})
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /api/v1/user/reseller/role",
		"POST /api/v1/user/reseller/withdrawals/:id/cancel",
		"GET /api/v1/user/reseller/manager/agents",
		"GET /api/v1/admin/reseller/agents",
		"POST /api/v1/admin/reseller/withdrawals/:id/review",
	} {
		if _, ok := registered[expected]; !ok {
			t.Errorf("missing native reseller route %s", expected)
		}
	}
}

func TestExecutableAdminRoutesHaveNativeGinParity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &handler.Handlers{}
	fillPointers(reflect.ValueOf(h))

	noop := func(c *gin.Context) { c.Next() }
	r := gin.New()
	RegisterAdminRoutes(
		r.Group("/api/v1"),
		h,
		middleware.AdminAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		middleware.StepUpAuthMiddleware(noop),
		nil,
		nil,
	)

	registered := make(map[routeKey]struct{})
	for _, route := range r.Routes() {
		registered[routeKey{method: route.Method, path: route.Path}] = struct{}{}
	}

	var missing []routeKey
	for _, route := range ExecutableAdminRoutes(h) {
		key := routeKey{method: route.Method, path: route.Path}
		if _, ok := registered[key]; !ok {
			missing = append(missing, key)
		}
	}
	require.Empty(t, missing, "native Gin router must include every executable admin route")
}
