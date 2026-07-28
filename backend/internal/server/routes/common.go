package routes

import (
	"database/sql"
	"net"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

const (
	middlewareRequestLogger = "request_logger"
	middlewareSecurity      = "security_headers"
	middlewareCORS          = "cors"
)

// RegisterCommonRoutes 注册通用路由（健康检查、状态等）
func RegisterCommonRoutes(r gin.IRoutes, dbs ...*sql.DB) {
	var db *sql.DB
	if len(dbs) > 0 {
		db = dbs[0]
	}

	// 健康检查
	r.GET("/health", gatewayctx.AdaptGinHandler(commonHealth))
	r.GET("/internal/readyz", gatewayctx.AdaptGinHandler(commonReadyz(db)))

	// Claude Code 遥测日志（忽略，直接返回200）
	r.POST("/api/event_logging/batch", gatewayctx.AdaptGinHandler(commonEventLoggingBatch))

	// Setup status endpoint (always returns needs_setup: false in normal mode)
	// This is used by the frontend to detect when the service has restarted after setup
	r.GET("/setup/status", gatewayctx.AdaptGinHandler(commonSetupStatus))
}

func ExecutableCommonRoutes() []gatewayctx.RouteDef {
	return []gatewayctx.RouteDef{
		{
			Method:     http.MethodGet,
			Path:       "/health",
			Handler:    commonHealth,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/event_logging/batch",
			Handler:    commonEventLoggingBatch,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
		{
			Method:     http.MethodGet,
			Path:       "/setup/status",
			Handler:    commonSetupStatus,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
	}
}

func commonHealth(c gatewayctx.GatewayContext) {
	c.WriteJSON(http.StatusOK, map[string]any{"status": "ok"})
}

const readyzSchemaQuery = `SELECT COUNT(*)::bigint, COALESCE((SELECT filename FROM schema_migrations ORDER BY applied_at DESC, filename DESC LIMIT 1), '') FROM schema_migrations`

func commonReadyz(db *sql.DB) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		req := c.Request()
		if req == nil || !isLoopbackRemoteAddr(req.RemoteAddr) {
			c.WriteJSON(http.StatusForbidden, map[string]any{"status": "forbidden"})
			return
		}
		if db == nil {
			c.WriteJSON(http.StatusServiceUnavailable, map[string]any{"status": "unavailable"})
			return
		}

		var count int64
		var latest string
		if err := db.QueryRowContext(req.Context(), readyzSchemaQuery).Scan(&count, &latest); err != nil {
			c.WriteJSON(http.StatusServiceUnavailable, map[string]any{"status": "unavailable"})
			return
		}
		c.WriteJSON(http.StatusOK, map[string]any{
			"status":                            "ok",
			"schema_migrations_count":           count,
			"schema_migrations_latest_filename": latest,
		})
	}
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteAddr = host
	}
	remoteAddr = strings.Trim(remoteAddr, "[]")
	ip := net.ParseIP(remoteAddr)
	return ip != nil && ip.IsLoopback()
}

func commonEventLoggingBatch(c gatewayctx.GatewayContext) {
	c.SetStatus(http.StatusOK)
}

func commonSetupStatus(c gatewayctx.GatewayContext) {
	c.WriteJSON(http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"needs_setup": false,
			"step":        "completed",
		},
	})
}
