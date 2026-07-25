package middleware

import "github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

// AbortWithErrorContext mirrors AbortWithError for the native gateway context.
// It keeps context-based handlers on the same public error contract as Gin handlers.
func AbortWithErrorContext(c gatewayctx.GatewayContext, statusCode int, code, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}
