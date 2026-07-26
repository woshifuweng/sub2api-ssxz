package routes

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
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
		nil)
	RegisterAdminRoutes(v1, h,
		middleware.AdminAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		middleware.StepUpAuthMiddleware(noop),
		nil)
}
