package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// AuthSubject is the minimal authenticated identity stored in gin context.
// Decision: {UserID int64, Concurrency int, AllowedGroupIDs []int64}
type AuthSubject struct {
	UserID          int64
	Concurrency     int
	AllowedGroupIDs []int64
}

func GetAuthSubjectFromContext(c *gin.Context) (AuthSubject, bool) {
	return GetAuthSubjectFromGatewayContext(gatewayctx.FromGin(c))
}

func GetAuthSubjectFromGatewayContext(c gatewayctx.GatewayContext) (AuthSubject, bool) {
	if c == nil {
		return AuthSubject{}, false
	}
	value, exists := c.Value(string(ContextKeyUser))
	if !exists {
		return AuthSubject{}, false
	}
	subject, ok := value.(AuthSubject)
	return subject, ok
}

func GetUserRoleFromContext(c *gin.Context) (string, bool) {
	return GetUserRoleFromGatewayContext(gatewayctx.FromGin(c))
}

func GetUserRoleFromGatewayContext(c gatewayctx.GatewayContext) (string, bool) {
	if c == nil {
		return "", false
	}
	value, exists := c.Value(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

func cloneAuthSubjectGroupIDs(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			out = append(out, value)
		}
	}
	return out
}
