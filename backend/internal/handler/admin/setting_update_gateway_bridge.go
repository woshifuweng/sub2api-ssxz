package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

// UpdateSettingsGateway preserves the production settings update path, including
// its step-up and session-binding checks, while admin routes still run on Gin.
func (h *SettingHandler) UpdateSettingsGateway(c gatewayctx.GatewayContext) {
	ginContext, ok := c.Native().(*gin.Context)
	if !ok || ginContext == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotImplemented, "settings update is unavailable on this transport")
		return
	}
	h.UpdateSettings(ginContext)
}
