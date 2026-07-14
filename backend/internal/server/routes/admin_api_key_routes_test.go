package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterAdminAPIKeyRoutesIncludesAuditedMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	registerAdminAPIKeyRoutes(admin, &handler.Handlers{
		Admin: &handler.AdminHandlers{
			APIKey: adminhandler.NewAdminAPIKeyHandler(nil),
		},
	})

	expected := map[string]bool{
		http.MethodGet + " /api/v1/admin/api-keys":              false,
		http.MethodPut + " /api/v1/admin/api-keys/:id":          false,
		http.MethodPatch + " /api/v1/admin/api-keys/:id/status": false,
		http.MethodDelete + " /api/v1/admin/api-keys/:id":       false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}
	for route, found := range expected {
		require.True(t, found, "route %s should be registered", route)
	}
}
