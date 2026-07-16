package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService      *service.UserService
	affiliateService *service.AffiliateService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService, affiliateService *service.AffiliateService) *UserHandler {
	return &UserHandler{
		userService:      userService,
		affiliateService: affiliateService,
	}
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateProfileRequest represents the update profile request payload
type UpdateProfileRequest struct {
	Username *string `json:"username"`
}

type UpdateAvatarRequest struct {
	Avatar string `json:"avatar"`
}

type AvatarResponse struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	ByteSize    int    `json:"byte_size,omitempty"`
}

const maxAvatarRequestBytes = 150 * 1024

func avatarResponse(avatar *service.UserAvatar) *AvatarResponse {
	if avatar == nil {
		return nil
	}
	return &AvatarResponse{
		URL:         avatar.URL,
		ContentType: avatar.ContentType,
		ByteSize:    avatar.ByteSize,
	}
}

func decodeAvatarRequest(c gatewayctx.GatewayContext, req *UpdateAvatarRequest) error {
	if c == nil || c.Request() == nil || c.Request().Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(io.LimitReader(c.Request().Body, maxAvatarRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

// GetProfile handles getting user profile
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	h.GetProfileGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) GetProfileGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userData, err := h.userService.GetByID(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.UserFromService(userData))
}

// GetAvatar returns the current user's avatar only.
// GET /api/v1/user/avatar
func (h *UserHandler) GetAvatar(c *gin.Context) {
	h.GetAvatarGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) GetAvatarGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	avatar, err := h.userService.GetAvatar(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, avatarResponse(avatar))
}

// UpdateAvatar changes or removes the current user's avatar. The target user
// ID is always taken from the authenticated session, never from the payload.
// PUT /api/v1/user/avatar
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	h.UpdateAvatarGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) UpdateAvatarGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req UpdateAvatarRequest
	if err := decodeAvatarRequest(c, &req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid avatar request: "+err.Error())
		return
	}
	if req.Avatar != "" {
		if err := service.ValidateUserAvatarUpload(req.Avatar); err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	}

	avatar, err := h.userService.SetAvatar(c.Request().Context(), subject.UserID, req.Avatar)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, avatarResponse(avatar))
}

// ChangePassword handles changing user password
// POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	h.ChangePasswordGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) ChangePasswordGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.ChangePasswordRequest{
		CurrentPassword: req.OldPassword,
		NewPassword:     req.NewPassword,
	}
	err := h.userService.ChangePassword(c.Request().Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "Password changed successfully"})
}

// UpdateProfile handles updating user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	h.UpdateProfileGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) UpdateProfileGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.UpdateProfileRequest{
		Username: req.Username,
	}
	updatedUser, err := h.userService.UpdateProfile(c.Request().Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.UserFromService(updatedUser))
}

// GetAffiliate returns the current user's affiliate details.
// GET /api/v1/user/aff
func (h *UserHandler) GetAffiliate(c *gin.Context) {
	h.GetAffiliateGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) GetAffiliateGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}
	if h.affiliateService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Affiliate service unavailable")
		return
	}

	detail, err := h.affiliateService.GetAffiliateDetail(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, detail)
}

// TransferAffiliateQuota transfers all available affiliate quota into current balance.
// POST /api/v1/user/aff/transfer
func (h *UserHandler) TransferAffiliateQuota(c *gin.Context) {
	h.TransferAffiliateQuotaGateway(gatewayctx.FromGin(c))
}

func (h *UserHandler) TransferAffiliateQuotaGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}
	if h.affiliateService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Affiliate service unavailable")
		return
	}

	transferred, balance, err := h.affiliateService.TransferAffiliateQuota(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{
		"transferred_quota": transferred,
		"balance":           balance,
	})
}

type gatewayJSONResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g gatewayJSONResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g gatewayJSONResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}
