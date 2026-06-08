package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestExecutableChatWorkspaceRoutesDisabledByDefault(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "")

	routes := ExecutableChatWorkspaceRoutes(&handler.Handlers{})
	require.Empty(t, routes)
}

func TestExecutableChatWorkspaceRoutesRequireHandler(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "true")

	require.Empty(t, ExecutableChatWorkspaceRoutes(nil))
	require.Empty(t, ExecutableChatWorkspaceRoutes(&handler.Handlers{}))
}

func TestExecutableChatWorkspaceRoutesEnabled(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "true")

	h := &handler.Handlers{
		ChatWorkspace: &handler.ChatWorkspaceHandler{},
	}
	routes := ExecutableChatWorkspaceRoutes(h)

	require.Len(t, routes, 9)
	require.Equal(t, "/api/v1/chat-workspace/conversations", routes[0].Path)
	require.Equal(t, "/api/v1/chat-workspace/image-tasks/:id", routes[len(routes)-1].Path)
	for _, route := range routes {
		require.Contains(t, route.Middleware, "jwt_auth")
		require.Contains(t, route.Middleware, "backend_mode_user_guard")
	}
}
