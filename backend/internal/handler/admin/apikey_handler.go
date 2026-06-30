package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
	}
}

// AdminUpdateAPIKeyGroupRequest represents the request to update an API key's group
type AdminUpdateAPIKeyGroupRequest struct {
	GroupID *int64 `json:"group_id"` // nil=不修改, 0=解绑, >0=绑定到目标分组
}

// UpdateGroup handles updating an API key's group binding
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	h.UpdateGroupGateway(gatewayctx.FromGin(c))
}

func (h *AdminAPIKeyHandler) UpdateGroupGateway(c gatewayctx.GatewayContext) {
	keyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyGroupRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	operator := adminAuditOperatorFromGateway(c)
	groupID := "nil"
	if req.GroupID != nil {
		groupID = strconv.FormatInt(*req.GroupID, 10)
	}
	result, err := h.adminService.AdminUpdateAPIKeyGroupID(c.Request().Context(), keyID, req.GroupID)
	if err != nil {
		logAdminAudit("apikey", "update_group failed operator=%s api_key_id=%d group_id=%s error_reason=%s", operator, keyID, groupID, adminAuditErrorReason(err))
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logAdminAudit("apikey", "update_group succeeded operator=%s api_key_id=%d group_id=%s", operator, keyID, groupID)

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
