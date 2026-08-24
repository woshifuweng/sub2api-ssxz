package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// BatchAccountUsageRequest is the request body for fetching usage snapshots
// for the accounts currently visible in the admin table.
type BatchAccountUsageRequest struct {
	AccountIDs []int64 `json:"account_ids"`
	Force      bool    `json:"force"`
}

// GetBatchUsage returns account usage snapshots without failing the whole
// request when one account cannot be queried.
// POST /api/v1/admin/accounts/usage/batch
func (h *AccountHandler) GetBatchUsage(c *gin.Context) {
	h.GetBatchUsageGateway(gatewayctx.FromGin(c))
}

func (h *AccountHandler) GetBatchUsageGateway(c gatewayctx.GatewayContext) {
	var req BatchAccountUsageRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
			"usage":  map[int64]any{},
			"errors": map[int64]string{},
		})
		return
	}

	usage, errorsByAccount, err := h.accountUsageService.GetUsageBatch(c.Request().Context(), accountIDs, req.Force)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"usage":  usage,
		"errors": errorsByAccount,
	})
}
