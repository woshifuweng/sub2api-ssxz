package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/stretchr/testify/require"
)

func TestExecutableAdminRoutesIncludesBatchAccountUsage(t *testing.T) {
	routeDefs := ExecutableAdminRoutes(&handler.Handlers{
		Admin: &handler.AdminHandlers{Account: &adminhandler.AccountHandler{}},
	})

	for _, routeDef := range routeDefs {
		if routeDef.Method == http.MethodPost && routeDef.Path == "/api/v1/admin/accounts/usage/batch" {
			require.Contains(t, routeDef.Middleware, "admin_auth")
			require.NotNil(t, routeDef.Handler)
			return
		}
	}

	t.Fatal("POST /api/v1/admin/accounts/usage/batch is not registered")
}
