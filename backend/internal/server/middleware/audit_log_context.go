package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// BeginAuditLogContext captures a native Gateway request and returns a
// completion callback that records the authenticated admin operation.
func BeginAuditLogContext(
	auditService *service.AuditLogService,
	c gatewayctx.GatewayContext,
	routePath string,
) func() {
	if auditService == nil || c == nil {
		return func() {}
	}

	req := c.Request()
	method := c.Method()
	var bodyRedacted string
	if req != nil && req.Body != nil && method != http.MethodGet {
		original := req.Body
		raw, err := io.ReadAll(io.LimitReader(original, service.AuditRequestBodyCaptureLimit+1))
		if err == nil {
			req.Body = &restoredBody{
				Reader: io.MultiReader(bytes.NewReader(raw), original),
				closer: original,
			}
			bodyRedacted = service.RedactAuditBody(raw, req.Header.Get("Content-Type"))
		}
	}

	startedAt := time.Now()
	return func() {
		status := gatewayResponseStatus(c)
		if status == 0 && !c.ResponseWritten() {
			return
		}
		if status == 0 {
			status = http.StatusOK
		}

		entry := &service.AuditLog{
			CreatedAt:        time.Now().UTC(),
			Action:           deriveAuditAction(method, routePath),
			Method:           method,
			Path:             routePath,
			ClientIP:         c.ClientIP(),
			RequestBody:      bodyRedacted,
			StatusCode:       status,
			LatencyMs:        time.Since(startedAt).Milliseconds(),
			ActorEmail:       gatewayStringValue(c, ContextKeyAuthEmail),
			ActorRole:        gatewayStringValue(c, string(ContextKeyUserRole)),
			AuthMethod:       gatewayStringValue(c, "auth_method"),
			CredentialMasked: maskedGatewayRequestCredential(c),
		}
		if req != nil {
			entry.UserAgent = req.UserAgent()
			if requestID, ok := req.Context().Value(ctxkey.RequestID).(string); ok {
				entry.RequestID = requestID
			}
		}
		if subject, ok := GetAuthSubjectFromGatewayContext(c); ok && subject.UserID > 0 {
			userID := subject.UserID
			entry.ActorUserID = &userID
		}

		extra := map[string]any{}
		if id := strings.TrimSpace(c.PathParam("id")); id != "" {
			extra["params"] = map[string]string{"id": id}
		}
		if req != nil {
			if query := service.RedactAuditQuery(req.URL.RawQuery); query != "" {
				extra["query"] = query
			}
		}
		if len(extra) > 0 {
			entry.Extra = extra
		}
		auditService.Record(entry)
	}
}

func gatewayResponseStatus(c gatewayctx.GatewayContext) int {
	if c == nil {
		return 0
	}
	if provider, ok := c.Native().(interface{ StatusCode() int }); ok {
		return provider.StatusCode()
	}
	return 0
}

func gatewayStringValue(c gatewayctx.GatewayContext, key string) string {
	value, ok := c.Value(key)
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return text
}

func maskedGatewayRequestCredential(c gatewayctx.GatewayContext) string {
	if apiKey := strings.TrimSpace(c.HeaderValue("x-api-key")); apiKey != "" {
		return "x-api-key " + service.MaskAuditCredential(apiKey)
	}
	authHeader := strings.TrimSpace(c.HeaderValue("Authorization"))
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 {
		return parts[0] + " " + service.MaskAuditCredential(strings.TrimSpace(parts[1]))
	}
	return service.MaskAuditCredential(authHeader)
}
