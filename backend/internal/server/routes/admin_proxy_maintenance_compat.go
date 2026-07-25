package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerProxyMaintenanceRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	plans := admin.Group("/proxy-maintenance-plans")
	{
		plans.GET("", h.Admin.ProxyMaintenance.List)
		plans.POST("", h.Admin.ProxyMaintenance.Create)
		plans.PUT("/:id", h.Admin.ProxyMaintenance.Update)
		plans.DELETE("/:id", h.Admin.ProxyMaintenance.Delete)
		plans.GET("/:id/results", h.Admin.ProxyMaintenance.ListResults)
	}
	admin.POST("/proxy-maintenance/run-now", h.Admin.ProxyMaintenance.RunNow)
	admin.GET("/proxy-maintenance/tasks/:task_id", h.Admin.ProxyMaintenance.GetTask)
}
