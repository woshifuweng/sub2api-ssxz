package admin

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"log/slog"
)

// Duplicate POST /api/v1/admin/channel-monitors/:id/duplicate
func (h *ChannelMonitorHandler) Duplicate(c *gin.Context) {
	id, ok := ParseChannelMonitorID(c)
	if !ok {
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	actorScope := adminActorScope(c)

	result, err := executeAdminIdempotent(
		c,
		"admin.channel_monitors.duplicate",
		struct {
			MonitorID int64 `json:"monitor_id"`
		}{MonitorID: id},
		service.DefaultWriteIdempotencyTTL(),
		func(ctx context.Context) (any, error) {
			monitor, err := h.monitorService.Duplicate(
				ctx,
				id,
				subject.UserID,
				actorScope,
				c.GetHeader("Idempotency-Key"),
			)
			if err != nil {
				return nil, err
			}
			return channelMonitorToResponse(monitor), nil
		},
	)
	if err != nil {
		reason := infraerrors.Reason(err)
		if reason == infraerrors.Reason(service.ErrIdempotencyInProgress) || reason == infraerrors.Reason(service.ErrIdempotencyStoreUnavail) {
			recovered, recoverErr := h.monitorService.RecoverDuplicate(
				c.Request.Context(),
				id,
				actorScope,
				c.GetHeader("Idempotency-Key"),
			)
			if recoverErr != nil {
				slog.Warn("channel_monitor_duplicate_recovery_failed", "monitor_id", id, "actor_scope", actorScope, "reason", reason, "error", recoverErr)
			} else if recovered != nil {
				c.Header("X-Idempotency-Recovered", "true")
				response.Success(c, channelMonitorToResponse(recovered))
				return
			}
		}
		response.ErrorFrom(c, err)
		return
	}

	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}
