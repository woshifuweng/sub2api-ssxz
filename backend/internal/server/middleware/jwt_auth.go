package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// NewJWTAuthMiddleware creates the Gin adapter for the shared JWT policy.
func NewJWTAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuth(authService, userService, userService, settingService, auditService))
}

type jwtUserReader interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

type userActivityToucher interface {
	TouchLastActiveForUser(ctx context.Context, user *service.User)
}

func jwtAuth(
	authService *service.AuthService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyJWTAuthContext(
			authService,
			userService,
			activityToucher,
			settingService,
			auditService,
			gatewayctx.FromGin(c),
		) {
			c.Next()
		}
	}
}

// ApplyJWTAuthContext applies the same JWT, token-version, and session-binding
// policy to Gin and native gateway contexts.
func ApplyJWTAuthContext(
	authService *service.AuthService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
	c gatewayctx.GatewayContext,
) bool {
	if c == nil || c.Request() == nil {
		return false
	}

	authHeader := c.HeaderValue("Authorization")
	if authHeader == "" {
		AbortWithErrorContext(c, 401, "UNAUTHORIZED", "Authorization header is required")
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		AbortWithErrorContext(c, 401, "INVALID_AUTH_HEADER", "Authorization header format must be 'Bearer {token}'")
		return false
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		AbortWithErrorContext(c, 401, "EMPTY_TOKEN", "Token cannot be empty")
		return false
	}

	claims, err := authService.ValidateToken(tokenString)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			AbortWithErrorContext(c, 401, "TOKEN_EXPIRED", "Token has expired")
			return false
		}
		AbortWithErrorContext(c, 401, "INVALID_TOKEN", "Invalid token")
		return false
	}

	user, err := userService.GetByID(c.Request().Context(), claims.UserID)
	if err != nil {
		AbortWithErrorContext(c, 401, "USER_NOT_FOUND", "User not found")
		return false
	}
	if !user.IsActive() {
		AbortWithErrorContext(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}
	if claims.TokenVersion != user.TokenVersion {
		AbortWithErrorContext(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
		return false
	}
	if !enforceSessionBindingContext(c, authService, settingService, auditService, claims) {
		return false
	}

	c.SetValue(string(ContextKeyUser), AuthSubject{
		UserID:          user.ID,
		Concurrency:     user.Concurrency,
		AllowedGroupIDs: cloneAuthSubjectGroupIDs(user.AllowedGroups),
	})
	c.SetValue(string(ContextKeyUserRole), user.Role)
	c.SetValue(ContextKeyAuthEmail, user.Email)
	c.SetValue(ContextKeySessionID, claims.SessionID)
	if activityToucher != nil {
		activityToucher.TouchLastActiveForUser(c.Request().Context(), user)
	}
	return true
}

// Deprecated: prefer GetAuthSubjectFromContext in auth_subject.go.
