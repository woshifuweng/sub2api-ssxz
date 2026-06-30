package admin

import (
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
)

const fallbackAdminAuditOperator = "admin"

func adminAuditOperatorFromGateway(c gatewayctx.GatewayContext) string {
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		return fallbackAdminAuditOperator
	}
	return fmt.Sprintf("admin:%d", subject.UserID)
}

func appendAdminAuditOperatorNote(notes string, operator string) string {
	notes = strings.TrimSpace(notes)
	operator = strings.TrimSpace(operator)
	if operator == "" {
		operator = fallbackAdminAuditOperator
	}
	operatorNote := "operator=" + operator
	if notes == "" {
		return operatorNote
	}
	return notes + "\n" + operatorNote
}

func logAdminAudit(component string, format string, args ...any) {
	component = strings.TrimSpace(component)
	if component == "" {
		component = "general"
	}
	logger.LegacyPrintf("handler.admin."+component, "[AdminAudit] "+format, args...)
}

func adminAuditErrorReason(err error) string {
	reason := strings.TrimSpace(infraerrors.Reason(err))
	if reason == "" {
		return "internal_error"
	}
	return reason
}
