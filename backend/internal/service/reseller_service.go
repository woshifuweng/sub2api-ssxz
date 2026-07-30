// Package service — ResellerService manages the 3-tier reseller hierarchy.
// Owner → Manager (agent_manager) → Agent (agent) → Customers
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// ResellerRoleAgent marks a user as a registered affiliate agent (推广代理).
	ResellerRoleAgent = "agent"
	// ResellerRoleManager marks a user as an operations manager (运营管理员).
	ResellerRoleManager = "agent_manager"

	WithdrawStatusPending   = "pending"
	WithdrawStatusApproved  = "approved"
	WithdrawStatusRejected  = "rejected"
	WithdrawStatusCancelled = "cancelled"

	WithdrawReviewActionApprove = "approve"
	WithdrawReviewActionReject  = "reject"
)

var (
	ErrResellerRoleNotFound        = infraerrors.NotFound("RESELLER_ROLE_NOT_FOUND", "reseller role not found")
	ErrResellerNotAgent            = infraerrors.Forbidden("NOT_AN_AGENT", "user is not a registered agent")
	ErrResellerNotManager          = infraerrors.Forbidden("NOT_A_MANAGER", "user is not a manager")
	ErrResellerInvalidRole         = infraerrors.BadRequest("INVALID_RESELLER_ROLE", "role must be 'agent' or 'agent_manager'")
	ErrWithdrawInsufficientBalance = infraerrors.BadRequest("INSUFFICIENT_BALANCE", "insufficient affiliate balance for withdrawal")
	ErrWithdrawRequestNotFound     = infraerrors.NotFound("WITHDRAW_REQUEST_NOT_FOUND", "withdraw request not found")
	ErrWithdrawAlreadyReviewed     = infraerrors.Conflict("WITHDRAW_ALREADY_REVIEWED", "request already reviewed")
	ErrWithdrawInvalidMethod       = infraerrors.BadRequest("INVALID_WITHDRAW_METHOD", "unsupported withdrawal method")
	ErrWithdrawInvalidAccount      = infraerrors.BadRequest("INVALID_WITHDRAW_ACCOUNT", "withdrawal account is required")
	ErrWithdrawInvalidStatus       = infraerrors.BadRequest("INVALID_WITHDRAW_STATUS", "status must be 'approved' or 'rejected'")
	ErrWithdrawInvalidAction       = infraerrors.BadRequest("INVALID_WITHDRAW_ACTION", "action must be 'approve' or 'reject'")
	ErrWithdrawReasonRequired      = infraerrors.BadRequest("WITHDRAW_REASON_REQUIRED", "reason is required when rejecting a withdrawal")
	ErrWithdrawNotOwner            = infraerrors.Forbidden("WITHDRAW_NOT_OWNER", "withdrawal does not belong to the current user")
	ErrWithdrawNotPending          = infraerrors.Conflict("WITHDRAW_NOT_PENDING", "only pending withdrawals can be changed")
	ErrResellerAgentNotManaged     = infraerrors.Forbidden("AGENT_NOT_MANAGED", "agent is not managed by this manager")
	ErrResellerCannotManageSelf    = infraerrors.BadRequest("CANNOT_MANAGE_SELF", "manager cannot grant or revoke their own role")
)

// --- Domain types ---

// ResellerRoleRecord is one row from user_reseller_roles.
type ResellerRoleRecord struct {
	UserID         int64      `json:"user_id"`
	Role           string     `json:"role"`
	GrantedBy      *int64     `json:"granted_by,omitempty"`
	GrantedAt      time.Time  `json:"granted_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	Notes          string     `json:"notes"`
	CommissionRate float64    `json:"commission_rate"`
}

// AgentSummary is used by manager to list all agents.
type AgentSummary struct {
	UserID         int64     `json:"user_id"`
	Email          string    `json:"email"`
	Username       string    `json:"username"`
	Role           string    `json:"role"`
	CommissionRate float64   `json:"commission_rate"`
	AffCode        string    `json:"aff_code"`
	RecruitCount   int       `json:"recruit_count"`
	AffQuota       float64   `json:"aff_quota"`
	GrantedAt      time.Time `json:"granted_at"`
	GrantedBy      *int64    `json:"granted_by,omitempty"`
}

// RecruitRecord is a single customer under an agent.
type RecruitRecord struct {
	UserID      int64      `json:"user_id"`
	Email       string     `json:"email"` // masked for agent, real for manager
	Username    string     `json:"username"`
	JoinedAt    *time.Time `json:"joined_at,omitempty"`
	TotalRebate float64    `json:"total_rebate"`
	IsActive    bool       `json:"is_active"` // any usage in last 30 days
}

// AgentDetail is manager view of a single agent with their recruits.
type AgentDetail struct {
	AgentSummary
	AffHistoryQuota float64         `json:"aff_history_quota"`
	Recruits        []RecruitRecord `json:"recruits"`
}

// AgentDashboard is the agent's own view of their stats and balance.
type AgentDashboard struct {
	UserID           int64   `json:"user_id"`
	AffCode          string  `json:"aff_code"`
	AffQuota         float64 `json:"aff_quota"`        // available to withdraw
	AffFrozenQuota   float64 `json:"aff_frozen_quota"` // pending maturation
	AffHistoryQuota  float64 `json:"aff_history_quota"`
	RecruitCount     int     `json:"recruit_count"`
	RebateRate       float64 `json:"rebate_rate"`
	PendingWithdraw  float64 `json:"pending_withdraw"` // sum of pending requests
	CommissionEarned float64 `json:"commission_earned"`
}

// ManagerDashboard is the manager's top-level stats overview.
type ManagerDashboard struct {
	TotalAgents        int `json:"total_agents"`
	TotalRecruits      int `json:"total_recruits"`
	PendingWithdrawals int `json:"pending_withdrawals"`
}

// WithdrawRequest is one row from affiliate_withdraw_requests.
type WithdrawRequest struct {
	ID          int64          `json:"id"`
	UserID      int64          `json:"user_id"`
	UserEmail   string         `json:"user_email,omitempty"`
	Username    string         `json:"username,omitempty"`
	Amount      float64        `json:"amount"`
	Method      string         `json:"method"`
	AccountInfo map[string]any `json:"account_info"`
	Status      string         `json:"status"` // pending / approved / rejected / cancelled
	RequestedAt time.Time      `json:"requested_at"`
	ReviewedAt  *time.Time     `json:"reviewed_at,omitempty"`
	ReviewedBy  *int64         `json:"reviewed_by,omitempty"`
	Note        string         `json:"note"`
}

// WithdrawInput is sent by the agent when requesting a withdrawal.
type WithdrawInput struct {
	Amount      float64        `json:"amount"`
	Method      string         `json:"method"`
	AccountInfo map[string]any `json:"account_info"`
}

// AgentFilter is used by manager when listing agents.
type AgentFilter struct {
	ManagerID       int64
	IncludeAllRoles bool
	Search          string
	Page            int
	PageSize        int
}

// WithdrawFilter is used for listing withdraw requests.
type WithdrawFilter struct {
	UserID    int64  // 0 = all users
	ManagerID int64  // 0 = no manager scope
	Status    string // "" = all statuses
	Page      int
	PageSize  int
}

// --- Repository interface ---

// ResellerRepository is the data-access interface for the reseller subsystem.
type ResellerRepository interface {
	GetRole(ctx context.Context, userID int64) (*ResellerRoleRecord, error)
	GrantRole(ctx context.Context, userID int64, role string, grantedBy int64, notes string) error
	GrantManagedAgent(ctx context.Context, userID, managerID int64, notes string) error
	RevokeRole(ctx context.Context, userID int64) error
	RevokeManagedAgent(ctx context.Context, userID, managerID int64) error
	ListAgents(ctx context.Context, filter AgentFilter) ([]AgentSummary, int64, error)
	GetAgentDetail(ctx context.Context, agentUserID, managerID int64) (*AgentDetail, error)
	GetAgentDashboard(ctx context.Context, agentUserID int64) (*AgentDashboard, error)
	ListMyRecruits(ctx context.Context, agentUserID int64, page, pageSize int, maskEmail bool) ([]RecruitRecord, int64, error)
	CreateWithdrawRequest(ctx context.Context, userID int64, input WithdrawInput) (*WithdrawRequest, error)
	ListWithdrawRequests(ctx context.Context, filter WithdrawFilter) ([]WithdrawRequest, int64, error)
	ReviewWithdrawRequest(ctx context.Context, id, reviewerID int64, status, note string) error
	CancelWithdrawRequest(ctx context.Context, withdrawalID, userID int64) error
	GetManagerDashboard(ctx context.Context, managerID int64) (*ManagerDashboard, error)
}

// --- Service ---

// ResellerService manages role assignment, agent dashboards, and withdrawals.
type ResellerService struct {
	repo ResellerRepository
}

// NewResellerService constructs the service.
func NewResellerService(repo ResellerRepository) *ResellerService {
	return &ResellerService{repo: repo}
}

// GetUserRole returns the active reseller role for a user, or "" if none.
func (s *ResellerService) GetUserRole(ctx context.Context, userID int64) (string, error) {
	rec, err := s.repo.GetRole(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrResellerRoleNotFound) {
			return "", nil
		}
		return "", err
	}
	return rec.Role, nil
}

// GrantRole assigns agent or manager status. Admin/manager calls this.
// Only admin (owner) can grant agent_manager; managers can only grant agent.
func (s *ResellerService) GrantRole(ctx context.Context, targetUserID, grantedBy int64, role, notes string) error {
	if role != ResellerRoleAgent && role != ResellerRoleManager {
		return ErrResellerInvalidRole
	}
	return s.repo.GrantRole(ctx, targetUserID, role, grantedBy, notes)
}

// RevokeRole removes the active reseller role.
func (s *ResellerService) RevokeRole(ctx context.Context, targetUserID int64) error {
	return s.repo.RevokeRole(ctx, targetUserID)
}

// GrantManagedAgent lets a manager grant only an agent role within their own scope.
func (s *ResellerService) GrantManagedAgent(ctx context.Context, targetUserID, managerID int64, notes string) error {
	if targetUserID == managerID {
		return ErrResellerCannotManageSelf
	}
	return s.repo.GrantManagedAgent(ctx, targetUserID, managerID, strings.TrimSpace(notes))
}

// RevokeManagedAgent only revokes an agent that was granted by the caller.
func (s *ResellerService) RevokeManagedAgent(ctx context.Context, targetUserID, managerID int64) error {
	if targetUserID == managerID {
		return ErrResellerCannotManageSelf
	}
	return s.repo.RevokeManagedAgent(ctx, targetUserID, managerID)
}

// ManagerDashboard returns aggregate stats for the manager overview page.
func (s *ResellerService) ManagerDashboard(ctx context.Context, managerID int64) (*ManagerDashboard, error) {
	return s.repo.GetManagerDashboard(ctx, managerID)
}

// ListAgents returns paginated agent list for manager view.
func (s *ResellerService) ListAgents(ctx context.Context, filter AgentFilter) ([]AgentSummary, int64, error) {
	return s.repo.ListAgents(ctx, filter)
}

// ListManagedAgents returns only agents granted by the manager.
func (s *ResellerService) ListManagedAgents(ctx context.Context, managerID int64, filter AgentFilter) ([]AgentSummary, int64, error) {
	filter.ManagerID = managerID
	return s.repo.ListAgents(ctx, filter)
}

// GetAgentDetail returns a single agent with their full recruit list (real emails).
func (s *ResellerService) GetAgentDetail(ctx context.Context, agentUserID int64) (*AgentDetail, error) {
	return s.repo.GetAgentDetail(ctx, agentUserID, 0)
}

// GetManagedAgentDetail restricts manager access to their own agent scope.
func (s *ResellerService) GetManagedAgentDetail(ctx context.Context, managerID, agentUserID int64) (*AgentDetail, error) {
	return s.repo.GetAgentDetail(ctx, agentUserID, managerID)
}

// AgentDashboard returns the agent's own stats and balance view.
func (s *ResellerService) AgentDashboard(ctx context.Context, agentUserID int64) (*AgentDashboard, error) {
	return s.repo.GetAgentDashboard(ctx, agentUserID)
}

// ListMyRecruits returns the agent's own recruit list with masked emails.
func (s *ResellerService) ListMyRecruits(ctx context.Context, agentUserID int64, page, pageSize int) ([]RecruitRecord, int64, error) {
	return s.repo.ListMyRecruits(ctx, agentUserID, page, pageSize, true)
}

// RequestWithdraw creates a pending withdrawal request after validating balance.
func (s *ResellerService) RequestWithdraw(ctx context.Context, agentUserID int64, input WithdrawInput) (*WithdrawRequest, error) {
	role, err := s.GetUserRole(ctx, agentUserID)
	if err != nil {
		return nil, err
	}
	if role != ResellerRoleAgent && role != ResellerRoleManager {
		return nil, ErrResellerNotAgent
	}
	if input.Amount <= 0 {
		return nil, ErrWithdrawInsufficientBalance
	}
	input.Method = strings.ToLower(strings.TrimSpace(input.Method))
	if input.Method == "" {
		input.Method = "balance_transfer"
	}
	if !validWithdrawMethod(input.Method) {
		return nil, ErrWithdrawInvalidMethod
	}
	if input.Method == "balance_transfer" {
		input.AccountInfo = map[string]any{}
		return s.repo.CreateWithdrawRequest(ctx, agentUserID, input)
	}
	account, ok := input.AccountInfo["account"].(string)
	if !ok || strings.TrimSpace(account) == "" || len(strings.TrimSpace(account)) > 200 {
		return nil, ErrWithdrawInvalidAccount
	}
	input.AccountInfo["account"] = strings.TrimSpace(account)
	accountJSON, err := json.Marshal(input.AccountInfo)
	if err != nil || len(accountJSON) > 4096 {
		return nil, ErrWithdrawInvalidAccount
	}
	return s.repo.CreateWithdrawRequest(ctx, agentUserID, input)
}

func validWithdrawMethod(method string) bool {
	switch method {
	case "alipay", "wechat", "bank", "manual", "balance_transfer":
		return true
	default:
		return false
	}
}

// GetWithdrawHistory returns the agent's own withdrawal history.
func (s *ResellerService) GetWithdrawHistory(ctx context.Context, agentUserID int64, page, pageSize int) ([]WithdrawRequest, int64, error) {
	return s.repo.ListWithdrawRequests(ctx, WithdrawFilter{
		UserID:   agentUserID,
		Page:     page,
		PageSize: pageSize,
	})
}

// CancelWithdrawal cancels a pending request owned by the current user.
func (s *ResellerService) CancelWithdrawal(ctx context.Context, userID, withdrawalID int64) error {
	return s.repo.CancelWithdrawRequest(ctx, withdrawalID, userID)
}

// ListAllWithdrawRequests is for manager/admin: list all or filtered requests.
func (s *ResellerService) ListAllWithdrawRequests(ctx context.Context, filter WithdrawFilter) ([]WithdrawRequest, int64, error) {
	return s.repo.ListWithdrawRequests(ctx, filter)
}

// ListManagedWithdrawRequests returns withdrawals belonging to the manager's agents.
func (s *ResellerService) ListManagedWithdrawRequests(ctx context.Context, managerID int64, filter WithdrawFilter) ([]WithdrawRequest, int64, error) {
	filter.ManagerID = managerID
	items, total, err := s.repo.ListWithdrawRequests(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	// Managers monitor request status but do not approve payouts, so payment
	// account details remain restricted to the owner and platform admins.
	for i := range items {
		items[i].AccountInfo = nil
	}
	return items, total, nil
}

// ReviewWithdrawRequest approves or rejects a pending request.
// On approval the aff_quota is transferred to the user's balance.
func (s *ResellerService) ReviewWithdrawRequest(ctx context.Context, requestID, reviewerID int64, action, reason string) error {
	reason = strings.TrimSpace(reason)

	var status string
	switch action {
	case WithdrawReviewActionApprove:
		status = WithdrawStatusApproved
	case WithdrawReviewActionReject:
		if reason == "" {
			return ErrWithdrawReasonRequired
		}
		status = WithdrawStatusRejected
	default:
		return ErrWithdrawInvalidAction
	}

	return s.repo.ReviewWithdrawRequest(ctx, requestID, reviewerID, status, reason)
}
