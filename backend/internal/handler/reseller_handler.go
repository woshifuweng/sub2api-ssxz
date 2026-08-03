// Package handler — ResellerHandler implements the 3-tier affiliate reseller system.
//
// Role routing:
//   - Agent routes    /api/v1/user/reseller/           jwt_auth + agent role check in handler
//   - Manager routes  /api/v1/user/reseller/manager/   jwt_auth + manager role check in handler
//   - Owner routes    /api/v1/admin/reseller/           admin_auth (no extra role check)
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ResellerHandler handles reseller/affiliate hierarchy endpoints.
type ResellerHandler struct {
	svc            *service.ResellerService
	totpService    *service.TotpService
	userService    *service.UserService
	settingService *service.SettingService
}

// NewResellerHandler constructs the handler. Called from wire.go.
func NewResellerHandler(
	svc *service.ResellerService,
	totpService *service.TotpService,
	userService *service.UserService,
	settingService *service.SettingService,
) *ResellerHandler {
	return &ResellerHandler{
		svc:            svc,
		totpService:    totpService,
		userService:    userService,
		settingService: settingService,
	}
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
func parseCommissionDate(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24 * time.Hour)
	}
	return &parsed, nil
}

func fallbackInviteCode(userID int64) string {
	return fmt.Sprintf("AGENT-%X", uint64(userID))
}

// InviteHandler GET /api/v1/user/reseller/invite.
func (h *ResellerHandler) InviteHandler(c *gin.Context) {
	h.InviteHandlerGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) InviteHandlerGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	summary, err := h.svc.GetInviteSummary(c.Request().Context(), userID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	inviteCode := strings.TrimSpace(summary.InviteCode)
	if inviteCode == "" {
		inviteCode = fallbackInviteCode(userID)
	}

	baseURL := ""
	if h.settingService != nil {
		baseURL = strings.TrimRight(strings.TrimSpace(h.settingService.GetFrontendURLLegacy(c.Request().Context())), "/")
	}
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(c.HeaderValue("Origin")), "/")
	}
	if baseURL == "" {
		if req := c.Request(); req != nil {
			scheme := "http"
			if req.TLS != nil {
				scheme = "https"
			}
			baseURL = scheme + "://" + req.Host
		}
	}
	if baseURL == "" {
		response.ErrorContext(h.r(c), http.StatusInternalServerError, "frontend URL is not configured")
		return
	}

	response.SuccessContext(h.r(c), gin.H{
		"invite_code":          inviteCode,
		"invite_link":          baseURL + "/register?ref=" + url.QueryEscape(inviteCode),
		"total_recruited":      summary.TotalRecruited,
		"recruited_this_month": summary.RecruitedThisMonth,
	})
}

// CommissionHandler GET /api/v1/user/reseller/commission.
func (h *ResellerHandler) CommissionHandler(c *gin.Context) {
	h.CommissionHandlerGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) CommissionHandlerGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	startAt, err := parseCommissionDate(c.QueryValue("start_date"), false)
	if err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid start_date")
		return
	}
	endAt, err := parseCommissionDate(c.QueryValue("end_date"), true)
	if err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid end_date")
		return
	}
	items, total, totalCommission, err := h.svc.ListCommission(c.Request().Context(), service.CommissionFilter{
		AgentUserID: userID,
		Page:        page,
		PageSize:    pageSize,
		StartAt:     startAt,
		EndAt:       endAt,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	pages := (total + int64(pageSize) - 1) / int64(pageSize)
	if pages < 1 {
		pages = 1
	}
	response.SuccessContext(h.r(c), gin.H{
		"items":                items,
		"total":                total,
		"total_commission_usd": totalCommission,
		"page":                 page,
		"page_size":            pageSize,
		"pages":                pages,
	})
}

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

func parseRecruitUserID(c gatewayctx.GatewayContext) (int64, bool) {
	recruitUserID, err := strconv.ParseInt(c.PathParam("userId"), 10, 64)
	if err != nil || recruitUserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid recruit user id")
		return 0, false
	}
	return recruitUserID, true
}

// GetRecruitDetail GET /api/v1/user/reseller/recruits/:userId.
func (h *ResellerHandler) GetRecruitDetail(c *gin.Context) {
	h.GetRecruitDetailGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetRecruitDetailGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	recruitUserID, ok := parseRecruitUserID(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetRecruitDetail(c.Request().Context(), userID, recruitUserID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

// GetRecruitLogs GET /api/v1/user/reseller/recruits/:userId/logs.
func (h *ResellerHandler) GetRecruitLogs(c *gin.Context) {
	h.GetRecruitLogsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetRecruitLogsGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	recruitUserID, ok := parseRecruitUserID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListRecruitUsageLogs(c.Request().Context(), userID, recruitUserID, page, pageSize)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// GetRecruitRecharges GET /api/v1/user/reseller/recruits/:userId/recharges.
func (h *ResellerHandler) GetRecruitRecharges(c *gin.Context) {
	h.GetRecruitRechargesGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) GetRecruitRechargesGateway(c gatewayctx.GatewayContext) {
	userID, ok := h.requireAgent(c)
	if !ok {
		return
	}
	recruitUserID, ok := parseRecruitUserID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.ListRecruitRecharges(c.Request().Context(), userID, recruitUserID, page, pageSize)
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
	if !middleware2.EnforceStepUpGateway(c, h.totpService, h.userService, h.settingService) {
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
	updatedBy, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	if !middleware2.EnforceStepUpGateway(c, h.totpService, h.userService, h.settingService) {
		return
	}
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.svc.RevokeRole(c.Request().Context(), targetID, updatedBy); err != nil {
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
	userID, err := parseOptionalPositiveInt64(c.QueryValue("user_id"))
	if err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid user id")
		return
	}
	items, total, err := h.svc.ListAllWithdrawRequests(c.Request().Context(), service.WithdrawFilter{
		UserID:   userID,
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
	managerID, err := parseOptionalPositiveInt64(c.QueryValue("manager_id"))
	if err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid manager id")
		return
	}
	items, total, err := h.svc.ListAgents(c.Request().Context(), service.AgentFilter{
		IncludeAllRoles: true,
		Search:          c.QueryValue("search"),
		Status:          c.QueryValue("status"),
		Role:            c.QueryValue("role"),
		ManagerID:       managerID,
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

// AdminGetAgentDetail GET /api/v1/admin/reseller/agents/:id
func (h *ResellerHandler) AdminGetAgentDetail(c *gin.Context) {
	h.AdminGetAgentDetailGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminGetAgentDetailGateway(c gatewayctx.GatewayContext) {
	targetID, ok := h.adminAgentTargetID(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetAdminAgentDetail(c.Request().Context(), targetID)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

// AdminGetAgentRecruits GET /api/v1/admin/reseller/agents/:id/recruits
func (h *ResellerHandler) AdminGetAgentRecruits(c *gin.Context) {
	h.AdminGetAgentRecruitsGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminGetAgentRecruitsGateway(c gatewayctx.GatewayContext) {
	targetID, ok := h.adminAgentTargetID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	items, total, err := h.svc.AdminListAgentRecruits(c.Request().Context(), targetID, page, pageSize)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.PaginatedContext(h.r(c), items, total, page, pageSize)
}

type adminUpdateAgentBody struct {
	Role         *string                    `json:"role"`
	ManagerID    service.OptionalInt64      `json:"-"`
	Notes        *string                    `json:"notes"`
	RebatePolicy *service.RebatePolicyInput `json:"rebate_policy"`
	Reason       string                     `json:"reason"`
}

func (b *adminUpdateAgentBody) UnmarshalJSON(data []byte) error {
	type bodyAlias adminUpdateAgentBody
	var alias bodyAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*b = adminUpdateAgentBody(alias)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	raw, exists := fields["manager_id"]
	if !exists {
		return nil
	}
	b.ManagerID.Set = true
	if string(raw) == "null" {
		return nil
	}
	var managerID int64
	if err := json.Unmarshal(raw, &managerID); err != nil {
		return err
	}
	b.ManagerID.Value = &managerID
	return nil
}

// AdminUpdateAgent PATCH /api/v1/admin/reseller/agents/:id
func (h *ResellerHandler) AdminUpdateAgent(c *gin.Context) {
	h.AdminUpdateAgentGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminUpdateAgentGateway(c gatewayctx.GatewayContext) {
	updatedBy, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	targetID, ok := h.adminAgentTargetID(c)
	if !ok {
		return
	}
	var body adminUpdateAgentBody
	if err := c.BindJSON(&body); err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid request")
		return
	}
	if body.Role != nil || body.ManagerID.Set || body.RebatePolicy != nil {
		if !middleware2.EnforceStepUpGateway(c, h.totpService, h.userService, h.settingService) {
			return
		}
	}
	detail, err := h.svc.UpdateAgent(c.Request().Context(), targetID, updatedBy, service.UpdateAgentInput{
		Role:         body.Role,
		ManagerID:    body.ManagerID,
		Notes:        body.Notes,
		RebatePolicy: body.RebatePolicy,
		Reason:       body.Reason,
	})
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

type adminDisableAgentBody struct {
	Reason string `json:"reason"`
}

// AdminDisableAgent POST /api/v1/admin/reseller/agents/:id/disable
func (h *ResellerHandler) AdminDisableAgent(c *gin.Context) {
	h.AdminDisableAgentGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminDisableAgentGateway(c gatewayctx.GatewayContext) {
	updatedBy, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	targetID, ok := h.adminAgentTargetID(c)
	if !ok {
		return
	}
	var body adminDisableAgentBody
	if err := c.BindJSON(&body); err != nil {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid request")
		return
	}
	detail, err := h.svc.DisableAgent(c.Request().Context(), targetID, updatedBy, body.Reason)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

// AdminEnableAgent POST /api/v1/admin/reseller/agents/:id/enable
func (h *ResellerHandler) AdminEnableAgent(c *gin.Context) {
	h.AdminEnableAgentGateway(gatewayctx.FromGin(c))
}

func (h *ResellerHandler) AdminEnableAgentGateway(c gatewayctx.GatewayContext) {
	updatedBy, ok := h.requireAdminUserSession(c)
	if !ok {
		return
	}
	targetID, ok := h.adminAgentTargetID(c)
	if !ok {
		return
	}
	detail, err := h.svc.EnableAgent(c.Request().Context(), targetID, updatedBy)
	if err != nil {
		response.ErrorFromContext(h.r(c), err)
		return
	}
	response.SuccessContext(h.r(c), detail)
}

func (h *ResellerHandler) adminAgentTargetID(c gatewayctx.GatewayContext) (int64, bool) {
	targetID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.ErrorContext(h.r(c), http.StatusBadRequest, "invalid agent id")
		return 0, false
	}
	return targetID, true
}

func parseOptionalPositiveInt64(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}
