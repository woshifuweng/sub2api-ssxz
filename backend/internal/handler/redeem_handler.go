package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RedeemHandler handles redeem code-related requests
type RedeemHandler struct {
	redeemService *service.RedeemService
	authService   *service.AuthService
}

type redeemGatewayResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g redeemGatewayResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g redeemGatewayResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

// NewRedeemHandler creates a new RedeemHandler
func NewRedeemHandler(redeemService *service.RedeemService, authService *service.AuthService) *RedeemHandler {
	return &RedeemHandler{
		redeemService: redeemService,
		authService:   authService,
	}
}

// RedeemRequest represents the redeem code request payload
type RedeemRequest struct {
	Code           string `json:"code" binding:"required"`
	TurnstileToken string `json:"turnstile_token"`
}

// RedeemResponse represents the redeem response
type RedeemResponse struct {
	Message        string   `json:"message"`
	Type           string   `json:"type"`
	Value          float64  `json:"value"`
	NewBalance     *float64 `json:"new_balance,omitempty"`
	NewConcurrency *int     `json:"new_concurrency,omitempty"`
}

// Redeem handles redeeming a code
// POST /api/v1/redeem
func (h *RedeemHandler) Redeem(c *gin.Context) {
	h.RedeemGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) RedeemGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(redeemGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req RedeemRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(redeemGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	code := strings.TrimSpace(req.Code)
	if !service.IsValidRedeemCodeFormat(code) {
		response.ErrorFromContext(redeemGatewayResponder{ctx: c}, infraerrors.BadRequest("REDEEM_CODE_INVALID", "redeem code format is invalid"))
		return
	}
	result, err := h.redeemService.Redeem(c.Request().Context(), subject.UserID, code)
	if err != nil {
		response.ErrorFromContext(redeemGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(redeemGatewayResponder{ctx: c}, dto.RedeemCodeFromService(result))
}

// GetHistory returns the user's redemption history
// GET /api/v1/redeem/history
func (h *RedeemHandler) GetHistory(c *gin.Context) {
	h.GetHistoryGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) GetHistoryGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(redeemGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePaginationValues(c)

	codes, result, err := h.redeemService.GetUserHistory(c.Request().Context(), subject.UserID, pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFromContext(redeemGatewayResponder{ctx: c}, err)
		return
	}

	out := make([]dto.RedeemCode, 0, len(codes))
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromService(&codes[i]))
	}
	response.PaginatedContext(redeemGatewayResponder{ctx: c}, out, result.Total, page, pageSize)
}
