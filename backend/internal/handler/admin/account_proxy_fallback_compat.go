package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

// RevertProxyFallback handles reverting account proxy to original before fallback.
// POST /api/v1/admin/accounts/:id/revert-proxy-fallback
func (h *AccountHandler) RevertProxyFallback(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if err := h.adminService.RevertAccountProxyFallback(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "reverted"})
}
