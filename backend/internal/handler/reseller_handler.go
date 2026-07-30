// Package handler — ResellerHandler implements the 3-tier affiliate reseller system.
//
// Role routing:
//   - Agent routes    /api/v1/user/reseller/           jwt_auth + agent role check in handler
//   - Manager routes  /api/v1/user/reseller/manager/   jwt_auth + manager role check in handler
//   - Owner routes    /api/v1/admin/reseller/           admin_auth (no extra role check)
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ResellerHandler handles reseller/affiliate hierarchy endpoints.
type ResellerHandler struct {
	svc *service.ResellerService
}

// NewResellerHandler constructs the handler. Called from wire.go.
func NewResellerHandler(svc *service.ResellerService) *ResellerHandler {
	return &ResellerHandler{svc: svc}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *ResellerHandler) r(c gatewayctx.GatewayContext) gatewayJSONResponder {
	return gatewayJSONResponder{ctx: c}
}

func (h *ResellerHandler) requireAdminUserSession(c gatewayctx.GatewayContext) (int64, bool) {
	if authMethod, ok := c.Value("auth_method"); ok && authMethod == service.AuditAuthMethodAdminAPIKey {
		response.ErrorContext(h.r(c), http.StatusForbidden, "admin API key cannot perform this operation")
		return 0, false
	}
	sub, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok || sub.UserID <= 0 {
		response.ErrorContext(h.r(c), http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return sub.UserID, true
}

// requireAgent returns the caller's userID if they hold the agent role.
func (h *ResellerHandler) requireAgent(c gatewayctx.GatewayContext) (int64, bool) {
	sub, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(h.r(c), http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	role, err := h.svc.GetUserRole(c.Request().Context(), sub.UserID)
	if err != nil || (role != service.ResellerRoleAgent && role != service.ResellerRoleManager) {
		response.ErrorContext(h.r(c), http.StatusForbidden, "requires agent role")
		return 0, false
	}
	return sub.UserID, true
}

// requireManager returns the caller's userID if they hold the manager role.
func (h *ResellerHandler) requireManager(c gatewayctx.GatewayContext) (int64, bool) {
	sub, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(h.r(c), http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	role, err := h.svc.GetUserRole(c.Request().Context(), sub.UserID)
	if err != nil || role != service.ResellerRoleManager {
		response.ErrorContext(h.r(c), http.StatusForbidden, "requires manager role")
		return 0, false
	}
	return sub.UserID, true
}

// ── Agent endpoints (jwt_auth + agent role) ───────────────────────────────────

// GetMyRole GET /api/v1/user/reseller/role — safe for any logged-in user.
func (h *ResellerHandler) GetMyRole(c *gin.Context) {
	h.GetMyRoleGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetMyRoleGateway(c gatewayctx.GatewayContext) {
	sub, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(h.r(c), http.StatusUnauthorized, "unauthorized")
		return
	}
	role, err := h.svc.GetUserRole(c.Request().Context(), sub.UserID)
	if errors.Is(err, service.ErrResellerRoleNotFound) {
		response.SuccessContext(h.r(c), gin.H{"role": nil})
		return
	}
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{"role": role})
}

// GetMyDashboard GET /api/v1/user/reseller/dashboard
func (h *ResellerHandler) GetMyDashboard(c *gin.Context) {
	h.GetMyDashboardGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetMyDashboardGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	dash, err := h.svc.AgentDashboard(c.Request().Context(), userID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), dash)
}

// GetMyRecruits GET /api/v1/user/reseller/recruits
func (h *ResellerHandler) GetMyRecruits(c *gin.Context) {
	h.GetMyRecruitsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetMyRecruitsGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListMyRecruits(c.Request().Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// GetMyWithdrawals GET /api/v1/user/reseller/withdrawals
func (h *ResellerHandler) GetMyWithdrawals(c *gin.Context) {
	h.GetMyWithdrawalsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetMyWithdrawalsGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.GetWithdrawHistory(c.Request().Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

type resellerWithdrawBody struct {
	Amount      float64                `json:"amount"      binding:"required,gt=0"`
	Method      string                 `json:"method"`
	AccountInfo map[string]interface{} `json:"account_info"`
}

// RequestWithdraw POST /api/v1/user/reseller/withdraw
func (h *ResellerHandler) RequestWithdraw(c *gin.Context) {
	h.RequestWithdrawGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) RequestWithdrawGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	var body resellerWithdrawBody
	if err := c.BindJSON(&body); err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid request")
		return
	}
	req, err := h.svc.RequestWithdraw(c.Request().Context(), userID, service.WithdrawInput{
		Amount:      body.Amount,
		Method:      body.Method,
		AccountInfo: body.AccountInfo,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), req)
}

// CancelWithdrawal POST /api/v1/user/reseller/withdrawals/:id/cancel
func (h *ResellerHandler) CancelWithdrawal(c *gin.Context) {
	h.CancelWithdrawalGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) CancelWithdrawalGateway(c gatewayctx.GatewayContext) {
	sub, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok || sub.UserID <= 0 {
		response.ErrorContext(h.r(c), http.StatusUnauthorized, "unauthorized")
		return
	}
	withdrawalID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || withdrawalID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid withdrawal id")
		return
	}
	if err := h.svc.CancelWithdrawal(c.Request().Context(), sub.UserID, withdrawalID); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{
		"id":     withdrawalID,
		"status": service.WithdrawStatusCancelled,
	})
}

// ── Manager endpoints (jwt_auth + manager role) ───────────────────────────────

// GetManagerDashboard GET /api/v1/user/reseller/manager/dashboard
func (h *ResellerHandler) GetManagerDashboard(c *gin.Context) {
	h.GetManagerDashboardGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetManagerDashboardGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	dash, err := h.svc.ManagerDashboard(c.Request().Context(), managerID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), dash)
}

// ManagerListAgents GET /api/v1/user/reseller/manager/agents
func (h *ResellerHandler) ManagerListAgents(c *gin.Context) {
	h.ManagerListAgentsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) ManagerListAgentsGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListManagedAgents(c.Request().Context(), managerID, service.AgentFilter{
		Search:   c.QueryValue("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// ManagerGetAgentDetail GET /api/v1/user/reseller/manager/agents/:id
func (h *ResellerHandler) ManagerGetAgentDetail(c *gin.Context) {
	h.ManagerGetAgentDetailGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) ManagerGetAgentDetailGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	agentID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || agentID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid agent id")
		return
	}
	detail, err := h.svc.GetManagedAgentDetail(c.Request().Context(), managerID, agentID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

type resellerGrantBody struct {
	Notes string `json:"notes"`
}

// ManagerGrantAgent POST /api/v1/user/reseller/manager/agents/:id/grant
// Manager can only grant the "agent" role.
func (h *ResellerHandler) ManagerGrantAgent(c *gin.Context) {
	h.ManagerGrantAgentGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) ManagerGrantAgentGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	var body resellerGrantBody
	_ = c.BindJSON(&body)
	if err := h.svc.GrantManagedAgent(c.Request().Context(), targetID, managerID, body.Notes); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{"user_id": targetID, "role": service.ResellerRoleAgent})
}

// ManagerRevokeAgent DELETE /api/v1/user/reseller/manager/agents/:id/role
func (h *ResellerHandler) ManagerRevokeAgent(c *gin.Context) {
	h.ManagerRevokeAgentGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) ManagerRevokeAgentGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.svc.RevokeManagedAgent(c.Request().Context(), targetID, managerID); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{"user_id": targetID})
}

// ManagerListWithdrawals GET /api/v1/user/reseller/manager/withdrawals (view only)
func (h *ResellerHandler) ManagerListWithdrawals(c *gin.Context) {
	h.ManagerListWithdrawalsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) ManagerListWithdrawalsGateway(c gatewayctx.GatewayContext) {
	managerID, ok := h.requireManager(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListManagedWithdrawRequests(c.Request().Context(), managerID, service.WithdrawFilter{
		Status:   c.QueryValue("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// ── Owner (admin_auth) endpoints ──────────────────────────────────────────────

type reviewWithdrawalBody struct {
	Action string `json:"action" binding:"required"`
	Reason string `json:"reason"`
}

// AdminReviewWithdrawal POST /api/v1/admin/reseller/withdrawals/:id/review
func (h *ResellerHandler) AdminReviewWithdrawal(c *gin.Context) {
	h.AdminReviewWithdrawalGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminReviewWithdrawalGateway(c gatewayctx.GatewayContext) {
	reviewerID, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	wid, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || wid <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid withdrawal id")
		return
	}
	var body reviewWithdrawalBody
	if err := c.BindJSON(&body); err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.svc.ReviewWithdrawRequest(
		c.Request().Context(),
		wid,
		reviewerID,
		body.Action,
		body.Reason,
	); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	status := service.WithdrawStatusRejected
	if body.Action == service.WithdrawReviewActionApprove {
		status = service.WithdrawStatusApproved
	}
	response.SuccessContext(h.r(c), gin.H{"id": wid, "status": status})
}

type adminGrantRoleBody struct {
	Role  string `json:"role"  binding:"required"`
	Notes string `json:"notes"`
}

// AdminGrantRole POST /api/v1/admin/reseller/agents/:id/role
// Owner can grant any valid role (agent or agent_manager).
func (h *ResellerHandler) AdminGrantRole(c *gin.Context) {
	h.AdminGrantRoleGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminGrantRoleGateway(c gatewayctx.GatewayContext) {
	grantedBy, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	var body adminGrantRoleBody
	if err := c.BindJSON(&body); err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.svc.GrantRole(c.Request().Context(), targetID, grantedBy, body.Role, body.Notes); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{"user_id": targetID, "role": body.Role})
}

// AdminRevokeRole DELETE /api/v1/admin/reseller/agents/:id/role
func (h *ResellerHandler) AdminRevokeRole(c *gin.Context) {
	h.AdminRevokeRoleGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminRevokeRoleGateway(c gatewayctx.GatewayContext) {
	if _, ok := h.requireAdminUserSession(c); !ok {
		return
	}
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.svc.RevokeRole(c.Request().Context(), targetID); err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), gin.H{"user_id": targetID})
}

// AdminListWithdrawals GET /api/v1/admin/reseller/withdrawals
func (h *ResellerHandler) AdminListWithdrawals(c *gin.Context) {
	h.AdminListWithdrawalsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminListWithdrawalsGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListAllWithdrawRequests(c.Request().Context(), service.WithdrawFilter{
		Status:   c.QueryValue("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// AdminListAgents GET /api/v1/admin/reseller/agents
func (h *ResellerHandler) AdminListAgents(c *gin.Context) {
	h.AdminListAgentsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminListAgentsGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListAgents(c.Request().Context(), service.AgentFilter{
		IncludeAllRoles: true,
		Search:          c.QueryValue("search"),
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}
