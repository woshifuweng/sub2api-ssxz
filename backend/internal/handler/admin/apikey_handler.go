package admin

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

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

func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{adminService: adminService}
}

func (h *AdminAPIKeyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	userID, ok := parseAdminAPIKeyOptionalPositiveID(c.Query("user_id"))
	if !ok {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	groupID, ok := parseAdminAPIKeyOptionalPositiveID(c.Query("group_id"))
	if !ok {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	switch status {
	case "", "active", "inactive", "expired":
	default:
		response.BadRequest(c, "Invalid API key status")
		return
	}
	search := strings.TrimSpace(c.Query("search"))
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}
	sortBy, sortOrder := service.NormalizeAdminAPIKeySort(c.Query("sort_by"), c.Query("sort_order"))
	inventoryService, ok := h.adminService.(service.AdminAPIKeyInventoryService)
	if !ok {
		response.InternalError(c, "Admin API key inventory is unavailable")
		return
	}
	result, err := inventoryService.ListAdminAPIKeys(c.Request.Context(), service.AdminAPIKeyListParams{
		Pagination: pagination.PaginationParams{
			Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder,
		},
		Filters: service.AdminAPIKeyListFilters{
			Search: search, UserID: userID, GroupID: groupID, Status: status,
		},
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.AdminAPIKeyListItem, 0, len(result.Items))
	now := time.Now()
	for _, item := range result.Items {
		items = append(items, dto.AdminAPIKeyListItemFromService(item, now))
	}
	response.Success(c, AdminAPIKeyListResponse{
		Items: items, Total: result.Pagination.Total, Page: result.Pagination.Page,
		PageSize: result.Pagination.PageSize, Pages: result.Pagination.Pages,
		Summary: dto.AdminAPIKeyListSummary{
			Total: result.Summary.Total, Active: result.Summary.Active,
			Inactive: result.Summary.Inactive, Expired: result.Summary.Expired,
			Last30DaysActualCost: result.Summary.Last30DaysActualCost,
		},
	})
}

func parseAdminAPIKeyOptionalPositiveID(raw string) (*int64, bool) {
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

type AdminUpdateAPIKeyGroupRequest struct {
	GroupID             *int64 `json:"group_id"`
	ResetRateLimitUsage *bool  `json:"reset_rate_limit_usage"`
}

type adminSetAPIKeyEnabledRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := parseAdminAPIKeyID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	var req AdminUpdateAPIKeyGroupRequest
	if err := decodeStrictAdminAPIKeyJSON(c.Request.Body, &req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	resetRequested := req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage
	if req.GroupID == nil && !resetRequested {
		response.BadRequest(c, "No API key update requested")
		return
	}
	var resetKey *service.APIKey
	if resetRequested {
		resetKey, err = h.adminService.AdminResetAPIKeyRateLimitUsage(c.Request.Context(), keyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	result := &service.AdminUpdateAPIKeyGroupIDResult{APIKey: resetKey}
	if req.GroupID != nil {
		actorUserID, ok := adminAPIKeyActorUserID(c)
		if !ok {
			response.Forbidden(c, "Admin access required")
			return
		}
		mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
		if ok {
			result, err = mutationService.AdminChangeAPIKeyGroup(c.Request.Context(), keyID, *req.GroupID, actorUserID)
		} else {
			// Compatibility path for older implementations and focused test doubles.
			// Production uses AdminAPIKeyMutationService so the audited actor is retained.
			result, err = h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID)
		}
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	response.Success(c, struct {
		APIKey                 *dto.APIKey `json:"api_key"`
		AutoGrantedGroupAccess bool        `json:"auto_granted_group_access"`
		GrantedGroupID         *int64      `json:"granted_group_id,omitempty"`
		GrantedGroupName       string      `json:"granted_group_name,omitempty"`
	}{
		APIKey:                 dto.APIKeyFromService(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	})
}

func (h *AdminAPIKeyHandler) SetEnabled(c *gin.Context) {
	keyID, err := parseAdminAPIKeyID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	var req adminSetAPIKeyEnabledRequest
	if err := decodeStrictAdminAPIKeyJSON(c.Request.Body, &req); err != nil || req.Enabled == nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	actorUserID, ok := adminAPIKeyActorUserID(c)
	if !ok {
		response.Forbidden(c, "Admin access required")
		return
	}
	mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
	if !ok {
		response.InternalError(c, "Admin API key mutation is unavailable")
		return
	}
	apiKey, err := mutationService.AdminSetAPIKeyEnabled(c.Request.Context(), keyID, *req.Enabled, actorUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, struct {
		APIKey *dto.APIKey `json:"api_key"`
	}{APIKey: dto.APIKeyFromService(apiKey)})
}

func (h *AdminAPIKeyHandler) Delete(c *gin.Context) {
	keyID, err := parseAdminAPIKeyID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	actorUserID, ok := adminAPIKeyActorUserID(c)
	if !ok {
		response.Forbidden(c, "Admin access required")
		return
	}
	mutationService, ok := h.adminService.(service.AdminAPIKeyMutationService)
	if !ok {
		response.InternalError(c, "Admin API key mutation is unavailable")
		return
	}
	if err := mutationService.AdminDeleteAPIKey(c.Request.Context(), keyID, actorUserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, struct {
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

func adminAPIKeyActorUserID(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return 0, false
	}
	role, ok := servermiddleware.GetUserRoleFromContext(c)
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
