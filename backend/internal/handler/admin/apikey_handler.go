package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

type AdminAPIKeyListResponse struct {
	Items    []dto.AdminAPIKeyListItem  `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
	Pages    int                        `json:"pages"`
	Summary  dto.AdminAPIKeyListSummary `json:"summary"`
}

// List returns a read-only, masked inventory of API keys across all users.
// GET /api/v1/admin/api-keys
func (h *AdminAPIKeyHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *AdminAPIKeyHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	userID, ok := parseOptionalPositiveID(c.QueryValue("user_id"))
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user ID")
		return
	}
	groupID, ok := parseOptionalPositiveID(c.QueryValue("group_id"))
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group ID")
		return
	}

	status := strings.TrimSpace(c.QueryValue("status"))
	switch status {
	case "", "active", "inactive", "expired":
	default:
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid API key status")
		return
	}

	search := strings.TrimSpace(c.QueryValue("search"))
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}
	sortBy, sortOrder := service.NormalizeAdminAPIKeySort(
		c.QueryValue("sort_by"),
		c.QueryValue("sort_order"),
	)
	result, err := h.adminService.ListAdminAPIKeys(c.Request().Context(), service.AdminAPIKeyListParams{
		Pagination: pagination.PaginationParams{
			Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder,
		},
		Filters: service.AdminAPIKeyListFilters{
			Search: search, UserID: userID, GroupID: groupID, Status: status,
		},
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	items := make([]dto.AdminAPIKeyListItem, 0, len(result.Items))
	now := time.Now()
	for _, item := range result.Items {
		items = append(items, dto.AdminAPIKeyListItemFromService(item, now))
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, AdminAPIKeyListResponse{
		Items: items,
		Total: result.Pagination.Total, Page: result.Pagination.Page,
		PageSize: result.Pagination.PageSize, Pages: result.Pagination.Pages,
		Summary: dto.AdminAPIKeyListSummary{
			Total: result.Summary.Total, Active: result.Summary.Active,
			Inactive: result.Summary.Inactive, Expired: result.Summary.Expired,
			Last30DaysActualCost: result.Summary.Last30DaysActualCost,
		},
	})
}

func parseOptionalPositiveID(raw string) (*int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, false
	}
	return &id, true
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
	}
}

// AdminUpdateAPIKeyGroupRequest represents the request to update an API key.
type AdminUpdateAPIKeyGroupRequest struct {
	GroupID             *int64 `json:"group_id"`
	ResetRateLimitUsage *bool  `json:"reset_rate_limit_usage"`
}

// UpdateGroup handles updating an API key's admin-managed fields.
type adminSetAPIKeyEnabledRequest struct {
	Enabled *bool `json:"enabled"`
}

// UpdateGroup handles updating an API key's group binding
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	h.UpdateGroupGateway(gatewayctx.FromGin(c))
}

func (h *AdminAPIKeyHandler) UpdateGroupGateway(c gatewayctx.GatewayContext) {
	keyID, err := parseAdminAPIKeyID(c.PathParam("id"))
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyGroupRequest
	if err := decodeStrictAdminAPIKeyJSON(c.Request().Body, &req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request")
		return
	}
	resetRequested := req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage
	if req.GroupID == nil && !resetRequested {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "No API key update requested")
		return
	}
	ctx := c.Request().Context()
	var resetKey *service.APIKey
	if resetRequested {
		resetKey, err = h.adminService.AdminResetAPIKeyRateLimitUsage(ctx, keyID)
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	}

	operator := adminAuditOperatorFromGateway(c)
	result := &service.AdminUpdateAPIKeyGroupIDResult{APIKey: resetKey}
	if req.GroupID != nil {
		actorUserID, ok := adminAPIKeyActorUserID(c)
		if !ok {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusForbidden, "Admin access required")
			return
		}
		mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
		if !ok {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Admin API key mutation is unavailable")
			return
		}
		groupID := strconv.FormatInt(*req.GroupID, 10)
		result, err = mutationService.AdminChangeAPIKeyGroup(ctx, keyID, *req.GroupID, actorUserID)
		if err != nil {
			logAdminAudit("apikey", "update_group failed operator=%s api_key_id=%d group_id=%s error_reason=%s", operator, keyID, groupID, adminAuditErrorReason(err))
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
		logAdminAudit("apikey", "update_group succeeded operator=%s api_key_id=%d group_id=%s", operator, keyID, groupID)
	}
	if resetRequested {
		logAdminAudit("apikey", "reset_rate_limit_usage succeeded operator=%s api_key_id=%d", operator, keyID)
	}

	resp := struct {
		APIKey                 *dto.APIKey `json:"api_key"`
		AutoGrantedGroupAccess bool        `json:"auto_granted_group_access"`
		GrantedGroupID         *int64      `json:"granted_group_id,omitempty"`
		GrantedGroupName       string      `json:"granted_group_name,omitempty"`
	}{
		APIKey:                 dto.APIKeyFromService(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, resp)
}

// SetEnabled toggles an API key through the audited admin-only mutation path.
// PATCH /api/v1/admin/api-keys/:id/status
func (h *AdminAPIKeyHandler) SetEnabled(c *gin.Context) {
	h.SetEnabledGateway(gatewayctx.FromGin(c))
}

func (h *AdminAPIKeyHandler) SetEnabledGateway(c gatewayctx.GatewayContext) {
	keyID, err := parseAdminAPIKeyID(c.PathParam("id"))
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid API key ID")
		return
	}
	var req adminSetAPIKeyEnabledRequest
	if err := decodeStrictAdminAPIKeyJSON(c.Request().Body, &req); err != nil || req.Enabled == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request")
		return
	}
	actorUserID, ok := adminAPIKeyActorUserID(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusForbidden, "Admin access required")
		return
	}
	mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Admin API key mutation is unavailable")
		return
	}
	apiKey, err := mutationService.AdminSetAPIKeyEnabled(c.Request().Context(), keyID, *req.Enabled, actorUserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, struct {
		APIKey *dto.APIKey `json:"api_key"`
	}{APIKey: dto.APIKeyFromService(apiKey)})
}

// Delete soft-deletes an API key and persists the audit event atomically.
// DELETE /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *AdminAPIKeyHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	keyID, err := parseAdminAPIKeyID(c.PathParam("id"))
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid API key ID")
		return
	}
	actorUserID, ok := adminAPIKeyActorUserID(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusForbidden, "Admin access required")
		return
	}
	mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Admin API key mutation is unavailable")
		return
	}
	if err := mutationService.AdminDeleteAPIKey(c.Request().Context(), keyID, actorUserID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, struct {
		Deleted bool `json:"deleted"`
	}{Deleted: true})
}

func parseAdminAPIKeyID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid API key ID")
	}
	return id, nil
}

func adminAPIKeyActorUserID(c gatewayctx.GatewayContext) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		return 0, false
	}
	role, ok := servermiddleware.GetUserRoleFromGatewayContext(c)
	if !ok || role != service.RoleAdmin {
		return 0, false
	}
	return subject.UserID, true
}

func decodeStrictAdminAPIKeyJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
