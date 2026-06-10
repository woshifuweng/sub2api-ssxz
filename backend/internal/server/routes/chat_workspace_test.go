package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestExecutableUserRoutesChatWorkspacePathsAreRegistered(t *testing.T) {
	routes := ExecutableUserRoutes(&handler.Handlers{
		ChatWorkspace: handler.NewChatWorkspaceHandler(nil),
	})

	expected := map[string]string{
		"GET /api/v1/chat-workspace/conversations":               http.MethodGet,
		"POST /api/v1/chat-workspace/conversations":              http.MethodPost,
		"GET /api/v1/chat-workspace/conversations/:id":           http.MethodGet,
		"GET /api/v1/chat-workspace/conversations/:id/messages":  http.MethodGet,
		"POST /api/v1/chat-workspace/conversations/:id/messages": http.MethodPost,
	}

	found := make(map[string]bool, len(expected))
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; ok {
			found[key] = true
			require.Contains(t, route.Middleware, "request_logger")
			require.Contains(t, route.Middleware, "client_request_id")
			require.Contains(t, route.Middleware, "jwt_auth")
			require.Contains(t, route.Middleware, "backend_mode_user_guard")
		}
	}

	for key := range expected {
		require.True(t, found[key], "route %s should be present", key)
	}
}
