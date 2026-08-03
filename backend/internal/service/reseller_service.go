// Package service — ResellerService manages the 3-tier reseller hierarchy.
// Owner → Manager (agent_manager) → Agent (agent) → Customers
package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// ResellerRoleAgent marks a user as a registered affiliate agent (推广代理).
	ResellerRoleAgent = "agent"
	// ResellerRoleManager marks a user as an operations manager (运营管理员).
	ResellerRoleManager = "agent_manager"

	ResellerStatusActive   = "active"
	ResellerStatusDisabled = "disabled"
	ResellerStatusRevoked  = "revoked"
	ResellerStatusAll      = "all"

	RebateModeGlobal   = "global"
	RebateModeDisabled = "disabled"
	RebateModeCustom   = "custom"

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
	ErrResellerInvalidStatus       = infraerrors.BadRequest("INVALID_RESELLER_STATUS", "status must be active, disabled, revoked, or all")
	ErrResellerInvalidRebatePolicy = infraerrors.BadRequest("INVALID_REBATE_POLICY", "invalid rebate policy")
	ErrResellerManagerInvalid      = infraerrors.New(422, "INVALID_RESELLER_MANAGER", "manager must be an active agent manager")
	ErrResellerManagerCycle        = infraerrors.New(422, "RESELLER_MANAGER_CYCLE", "manager assignment creates a hierarchy cycle")
	ErrResellerHasDirectAgents     = infraerrors.Conflict("RESELLER_HAS_DIRECT_AGENTS", "agent manager still has direct agents")
	ErrResellerHasPendingWithdraw  = infraerrors.Conflict("RESELLER_HAS_PENDING_WITHDRAWALS", "agent still has pending balance conversions")
	ErrResellerStateConflict       = infraerrors.Conflict("RESELLER_STATE_CONFLICT", "reseller state does not allow this operation")
	ErrResellerDisableReason       = infraerrors.BadRequest("DISABLE_REASON_REQUIRED", "disable reason is required")
)

// --- Domain types ---

// ResellerRoleRecord is one row from user_reseller_roles.
type ResellerRoleRecord struct {
	UserID         int64      `json:"user_id"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	ManagerID      *int64     `json:"manager_id,omitempty"`
	GrantedBy      *int64     `json:"granted_by,omitempty"`
	GrantedAt      time.Time  `json:"granted_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	Notes          string     `json:"notes"`
	CommissionRate float64    `json:"-"`
}

// AgentSummary is shared by manager and admin list views.
type AgentSummary struct {
	UserID                     int64      `json:"user_id"`
	Email                      string     `json:"email"`
	Username                   string     `json:"username"`
	Role                       string     `json:"role"`
	Status                     string     `json:"status"`
	ManagerID                  *int64     `json:"manager_id"`
	ManagerEmail               *string    `json:"manager_email"`
	EffectiveRebateRatePercent *float64   `json:"effective_rebate_rate_percent"`
	RebateMode                 string     `json:"rebate_mode"`
	AffCode                    string     `json:"aff_code"`
	RecruitCount               int        `json:"recruit_count"`
	CommissionBalance          string     `json:"commission_balance"`
	CommissionTotal            string     `json:"commission_total"`
	Notes                      string     `json:"notes"`
	GrantedAt                  time.Time  `json:"granted_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	DisabledAt                 *time.Time `json:"disabled_at"`
	DisabledByEmail            *string    `json:"disabled_by_email"`
	DisabledReason             string     `json:"disabled_reason"`
	RevokedAt                  *time.Time `json:"revoked_at"`
	GrantedBy                  *int64     `json:"granted_by,omitempty"`
}

// RecruitRecord is a single customer under an agent.
type RecruitRecord struct {
	UserID         int64      `json:"user_id"`
	Email          string     `json:"email"` // masked for agent, real for manager
	Username       string     `json:"username"`
	Status         string     `json:"status"`
	ResellerRole   string     `json:"reseller_role"`
	CommissionRate float64    `json:"commission_rate"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	JoinedAt       *time.Time `json:"joined_at,omitempty"`
	TotalRebate    float64    `json:"total_rebate"`
	IsActive       bool       `json:"is_active"` // any usage in last 30 days
}

// RecruitUsageLog is a scoped usage record shown in an agent's recruit drawer.
type RecruitUsageLog struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Model       string    `json:"model"`
	RequestType int16     `json:"request_type"`
	TotalTokens int64     `json:"total_tokens"`
	ActualCost  float64   `json:"actual_cost"`
}

// RecruitRecharge is a positive balance ledger entry for a recruit.
type RecruitRecharge struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"`
	Amount    float64   `json:"amount"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// CommissionRecord is one usage-based commission row for an agent.
type CommissionRecord struct {
	ID                    int64     `json:"id"`
	Time                  time.Time `json:"time"`
	SourceUserMaskedEmail string    `json:"source_user_masked_email"`
	SourceConsumptionUSD  float64   `json:"source_consumption_usd"`
	CommissionUSD         float64   `json:"commission_usd"`
	CommissionRate        float64   `json:"commission_rate"`
}

// CommissionFilter controls the agent commission list range and pagination.
type CommissionFilter struct {
	AgentUserID int64
	Page        int
	PageSize    int
	StartAt     *time.Time
	EndAt       *time.Time
}

// InviteSummary contains the persisted invite code and recruit counters.
type InviteSummary struct {
	InviteCode         string
	TotalRecruited     int
	RecruitedThisMonth int
}

// AgentDetail is manager view of a single agent with their recruits.
type AgentDetail struct {
	AgentSummary
	AffHistoryQuota        float64         `json:"aff_history_quota"`
	PendingRedemptionCount int             `json:"pending_redemption_count"`
	Recruits               []RecruitRecord `json:"recruits,omitempty"`
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
	Status          string
	Role            string
	Page            int
	PageSize        int
}

// OptionalInt64 distinguishes an omitted manager_id from an explicit null.
type OptionalInt64 struct {
	Set   bool
	Value *int64
}

type RebatePolicyInput struct {
	Mode        string   `json:"mode"`
	RatePercent *float64 `json:"rate_percent,omitempty"`
}

type UpdateAgentInput struct {
	Role         *string
	ManagerID    OptionalInt64
	Notes        *string
	RebatePolicy *RebatePolicyInput
	Reason       string
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
	RevokeRole(ctx context.Context, userID, updatedBy int64) error
	RevokeManagedAgent(ctx context.Context, userID, managerID int64) error
	ListAgents(ctx context.Context, filter AgentFilter) ([]AgentSummary, int64, error)
	GetAgentDetail(ctx context.Context, agentUserID, managerID int64) (*AgentDetail, error)
	GetAdminAgentDetail(ctx context.Context, agentUserID int64) (*AgentDetail, error)
	UpdateAgent(ctx context.Context, agentUserID, updatedBy int64, input UpdateAgentInput) (*AgentDetail, error)
	DisableAgent(ctx context.Context, agentUserID, updatedBy int64, reason string) (*AgentDetail, error)
	EnableAgent(ctx context.Context, agentUserID, updatedBy int64) (*AgentDetail, error)
	GetAgentDashboard(ctx context.Context, agentUserID int64) (*AgentDashboard, error)
	ListMyRecruits(ctx context.Context, agentUserID int64, page, pageSize int, maskEmail bool) ([]RecruitRecord, int64, error)
	GetRecruitDetail(ctx context.Context, agentUserID, recruitUserID int64, maskEmail bool) (*RecruitRecord, error)
	ListRecruitUsageLogs(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]RecruitUsageLog, int64, error)
	ListRecruitRecharges(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]RecruitRecharge, int64, error)
	ListCommission(ctx context.Context, filter CommissionFilter) ([]CommissionRecord, int64, float64, error)
	GetInviteSummary(ctx context.Context, agentUserID int64) (*InviteSummary, error)
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
func (s *ResellerService) RevokeRole(ctx context.Context, targetUserID, updatedBy int64) error {
	return s.repo.RevokeRole(ctx, targetUserID, updatedBy)
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
	if !validResellerStatusFilter(filter.Status) {
		return nil, 0, ErrResellerInvalidStatus
	}
	if filter.Role != "" && filter.Role != ResellerRoleAgent && filter.Role != ResellerRoleManager {
		return nil, 0, ErrResellerInvalidRole
	}
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

func (s *ResellerService) GetAdminAgentDetail(ctx context.Context, agentUserID int64) (*AgentDetail, error) {
	return s.repo.GetAdminAgentDetail(ctx, agentUserID)
}

func (s *ResellerService) UpdateAgent(ctx context.Context, agentUserID, updatedBy int64, input UpdateAgentInput) (*AgentDetail, error) {
	if input.Role != nil && *input.Role != ResellerRoleAgent && *input.Role != ResellerRoleManager {
		return nil, ErrResellerInvalidRole
	}
	if input.ManagerID.Set && input.ManagerID.Value != nil {
		if *input.ManagerID.Value <= 0 || *input.ManagerID.Value == agentUserID {
			return nil, ErrResellerManagerInvalid
		}
	}
	if err := validateRebatePolicy(input.RebatePolicy); err != nil {
		return nil, err
	}
	if input.Notes != nil {
		trimmed := strings.TrimSpace(*input.Notes)
		input.Notes = &trimmed
	}
	input.Reason = strings.TrimSpace(input.Reason)
	return s.repo.UpdateAgent(ctx, agentUserID, updatedBy, input)
}

func (s *ResellerService) DisableAgent(ctx context.Context, agentUserID, updatedBy int64, reason string) (*AgentDetail, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrResellerDisableReason
	}
	return s.repo.DisableAgent(ctx, agentUserID, updatedBy, reason)
}

func (s *ResellerService) EnableAgent(ctx context.Context, agentUserID, updatedBy int64) (*AgentDetail, error) {
	return s.repo.EnableAgent(ctx, agentUserID, updatedBy)
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

// GetRecruitDetail returns one direct or nested recruit within the caller's downline.
func (s *ResellerService) GetRecruitDetail(ctx context.Context, agentUserID, recruitUserID int64) (*RecruitRecord, error) {
	return s.repo.GetRecruitDetail(ctx, agentUserID, recruitUserID, true)
}

// ListRecruitUsageLogs returns usage rows scoped to the caller's downline.
func (s *ResellerService) ListRecruitUsageLogs(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]RecruitUsageLog, int64, error) {
	return s.repo.ListRecruitUsageLogs(ctx, agentUserID, recruitUserID, page, pageSize)
}

// ListRecruitRecharges returns positive balance ledger rows scoped to the caller's downline.
func (s *ResellerService) ListRecruitRecharges(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]RecruitRecharge, int64, error) {
	return s.repo.ListRecruitRecharges(ctx, agentUserID, recruitUserID, page, pageSize)
}

// ListCommission returns usage-based commission rows with source emails masked by the repository.
func (s *ResellerService) ListCommission(ctx context.Context, filter CommissionFilter) ([]CommissionRecord, int64, float64, error) {
	return s.repo.ListCommission(ctx, filter)
}

// GetInviteSummary returns the agent's persisted invite code and recruit counters.
func (s *ResellerService) GetInviteSummary(ctx context.Context, agentUserID int64) (*InviteSummary, error) {
	return s.repo.GetInviteSummary(ctx, agentUserID)
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
	input.AccountInfo = map[string]any{}
	return s.repo.CreateWithdrawRequest(ctx, agentUserID, input)
}

func validWithdrawMethod(method string) bool {
	return method == "balance_transfer"
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

func validResellerStatusFilter(status string) bool {
	switch status {
	case "", ResellerStatusActive, ResellerStatusDisabled, ResellerStatusRevoked, ResellerStatusAll:
		return true
	default:
		return false
	}
}

func validateRebatePolicy(policy *RebatePolicyInput) error {
	if policy == nil {
		return nil
	}
	switch policy.Mode {
	case RebateModeGlobal, RebateModeDisabled:
		return nil
	case RebateModeCustom:
		if policy.RatePercent == nil || math.IsNaN(*policy.RatePercent) || math.IsInf(*policy.RatePercent, 0) ||
			*policy.RatePercent < 0 || *policy.RatePercent > 100 {
			return ErrResellerInvalidRebatePolicy
		}
		return nil
	default:
		return ErrResellerInvalidRebatePolicy
	}
}
