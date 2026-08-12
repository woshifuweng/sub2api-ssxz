package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

func executableUserFeatureRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}
	mw := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"jwt_auth",
		"backend_mode_user_guard",
	}
	out := make([]gatewayctx.RouteDef, 0, 24)
	if h.User != nil {
		out = append(out,
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/user/aff",
				Handler:    h.User.GetAffiliateGateway,
				Middleware: mw,
			},
			gatewayctx.RouteDef{
				Method:     http.MethodPost,
				Path:       "/api/v1/user/aff/transfer",
				Handler:    h.User.TransferAffiliateQuotaGateway,
				Middleware: mw,
			},
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/user/affiliate/stats",
				Handler:    h.User.GetAffiliateStatsGateway,
				Middleware: mw,
			},
		)
	}
	if h.Reseller != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/role", Handler: h.Reseller.GetMyRoleGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/dashboard", Handler: h.Reseller.GetMyDashboardGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/recruits", Handler: h.Reseller.GetMyRecruitsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/recruits/:userId", Handler: h.Reseller.GetRecruitDetailGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/recruits/:userId/logs", Handler: h.Reseller.GetRecruitLogsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/recruits/:userId/recharges", Handler: h.Reseller.GetRecruitRechargesGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/commission", Handler: h.Reseller.CommissionHandlerGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/invite", Handler: h.Reseller.InviteHandlerGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/withdrawals", Handler: h.Reseller.GetMyWithdrawalsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/reseller/withdraw", Handler: h.Reseller.RequestWithdrawGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/reseller/withdrawals", Handler: h.Reseller.RequestWithdrawGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/reseller/withdrawals/:id/cancel", Handler: h.Reseller.CancelWithdrawalGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/manager/dashboard", Handler: h.Reseller.GetManagerDashboardGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/manager/agents", Handler: h.Reseller.ManagerListAgentsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/manager/agents/:id", Handler: h.Reseller.ManagerGetAgentDetailGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/reseller/manager/agents/:id/grant", Handler: h.Reseller.ManagerGrantAgentGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/user/reseller/manager/agents/:id/role", Handler: h.Reseller.ManagerRevokeAgentGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/reseller/manager/withdrawals", Handler: h.Reseller.ManagerListWithdrawalsGateway, Middleware: mw},
		)
	}
	if h.AvailableChannel != nil {
		out = append(out, gatewayctx.RouteDef{
			Method:     http.MethodGet,
			Path:       "/api/v1/channels/available",
			Handler:    h.AvailableChannel.ListGateway,
			Middleware: mw,
		})
	}
	if h.ChannelMonitor != nil {
		out = append(out,
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/channel-monitors",
				Handler:    h.ChannelMonitor.ListGateway,
				Middleware: mw,
			},
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/channel-monitors/:id/status",
				Handler:    h.ChannelMonitor.GetStatusGateway,
				Middleware: mw,
			},
		)
	}
	return out
}

func registerUserFeatureRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	if authenticated == nil || h == nil {
		return
	}
	if h.User != nil {
		user := authenticated.Group("/user")
		{
			user.GET("/aff", h.User.GetAffiliate)
			user.POST("/aff/transfer", h.User.TransferAffiliateQuota)
			user.GET("/affiliate/stats", h.User.GetAffiliateStats)
		}
	}
	if h.Reseller != nil {
		reseller := authenticated.Group("/user/reseller")
		{
			reseller.GET("/role", h.Reseller.GetMyRole)
			reseller.GET("/dashboard", h.Reseller.GetMyDashboard)
			reseller.GET("/recruits", h.Reseller.GetMyRecruits)
			reseller.GET("/recruits/:userId", h.Reseller.GetRecruitDetail)
			reseller.GET("/recruits/:userId/logs", h.Reseller.GetRecruitLogs)
			reseller.GET("/recruits/:userId/recharges", h.Reseller.GetRecruitRecharges)
			reseller.GET("/commission", h.Reseller.CommissionHandler)
			reseller.GET("/invite", h.Reseller.InviteHandler)
			reseller.GET("/withdrawals", h.Reseller.GetMyWithdrawals)
			reseller.POST("/withdraw", h.Reseller.RequestWithdraw)
			reseller.POST("/withdrawals", h.Reseller.RequestWithdraw)
			reseller.POST("/withdrawals/:id/cancel", h.Reseller.CancelWithdrawal)
			manager := reseller.Group("/manager")
			{
				manager.GET("/dashboard", h.Reseller.GetManagerDashboard)
				manager.GET("/agents", h.Reseller.ManagerListAgents)
				manager.GET("/agents/:id", h.Reseller.ManagerGetAgentDetail)
				manager.POST("/agents/:id/grant", h.Reseller.ManagerGrantAgent)
				manager.DELETE("/agents/:id/role", h.Reseller.ManagerRevokeAgent)
				manager.GET("/withdrawals", h.Reseller.ManagerListWithdrawals)
			}
		}
	}
	if h.AvailableChannel != nil {
		authenticated.GET("/channels/available", h.AvailableChannel.List)
	}
	if h.ChannelMonitor != nil {
		monitors := authenticated.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
	}
}
