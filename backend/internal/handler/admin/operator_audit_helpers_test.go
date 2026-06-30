package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAuditOperatorFromGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})

	require.Equal(t, "admin:42", adminAuditOperatorFromGateway(gatewayctx.FromGin(c)))
}

func TestAdminAuditOperatorFromGatewayFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	require.Equal(t, "admin", adminAuditOperatorFromGateway(gatewayctx.FromGin(c)))
}

func TestAppendAdminAuditOperatorNote(t *testing.T) {
	require.Equal(t, "operator=admin:42", appendAdminAuditOperatorNote("", "admin:42"))
	require.Equal(t, "manual credit\noperator=admin:42", appendAdminAuditOperatorNote(" manual credit ", "admin:42"))
	require.Equal(t, "manual credit\noperator=admin", appendAdminAuditOperatorNote("manual credit", ""))
}

func TestAdminAuditErrorReason(t *testing.T) {
	require.Equal(t, "INVALID_STATUS", adminAuditErrorReason(infraerrors.BadRequest("INVALID_STATUS", "details should not be logged")))
	require.Equal(t, "internal_error", adminAuditErrorReason(assertionError("raw provider text should not be logged")))
}

type assertionError string

func (e assertionError) Error() string {
	return string(e)
}
