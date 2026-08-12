package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type openAISlotAcquireResult int

const (
	openAISlotAcquireOK openAISlotAcquireResult = iota
	openAISlotAcquireFailed
	openAISlotAcquireProfitVetoed
)

func (h *OpenAIGatewayHandler) handleOpenAIProfitVetoExhausted(c *gin.Context, streamStarted bool, reqLog *zap.Logger, vetoCount int) {
	reqLog.Warn("openai.profit_veto_attempts_exhausted", zap.Int("profit_veto_count", vetoCount))
	markOpsRoutingCapacityLimited(c)
	h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", profitVetoExhaustedMessage, streamStarted)
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlot(
	c *gin.Context,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), openAISlotAcquireResult) {
	if selection == nil || selection.Account == nil {
		markOpsRoutingCapacityLimited(c)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, openAISlotAcquireFailed
	}

	ctx := service.ContextWithSelectionProfitGate(c.Request.Context(), selection)
	account := selection.Account
	admit := func(release func()) (func(), openAISlotAcquireResult) {
		latest, vetoed, reason := h.gatewayService.ProfitControlVetoLatest(ctx, account)
		if vetoed {
			if release != nil {
				release()
			}
			reqLog.Debug("openai.account_slot_profit_vetoed", zap.Int64("account_id", account.ID), zap.String("reason", reason))
			return nil, openAISlotAcquireProfitVetoed
		}
		account = latest
		selection.Account = latest
		if selection.ProfitGateActive() {
			if err := h.gatewayService.BindStickySessionAfterProfitAdmission(ctx, groupID, sessionHash, account.ID); err != nil {
				reqLog.Warn("openai.bind_sticky_session_after_profit_admission_failed", zap.Int64("account_id", account.ID), zap.Error(err))
			}
		}
		return wrapReleaseOnDone(ctx, release), openAISlotAcquireOK
	}

	if selection.Acquired {
		return admit(selection.ReleaseFunc)
	}
	if selection.WaitPlan == nil {
		markOpsRoutingCapacityLimited(c)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, openAISlotAcquireFailed
	}

	fastRelease, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(ctx, account.ID, selection.WaitPlan.MaxConcurrency)
	if err != nil {
		reqLog.Warn("openai.account_slot_quick_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyError(c, err, "account", *streamStarted)
		return nil, openAISlotAcquireFailed
	}
	if fastAcquired {
		return admit(fastRelease)
	}

	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(ctx, account.ID, selection.WaitPlan.MaxWaiting)
	if waitErr != nil {
		reqLog.Warn("openai.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
	} else if !canWait {
		reqLog.Info("openai.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", *streamStarted)
		return nil, openAISlotAcquireFailed
	}

	waitCounted := waitErr == nil && canWait
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
		}
	}()

	release, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("openai.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyError(c, err, "account", *streamStarted)
		return nil, openAISlotAcquireFailed
	}
	if waitCounted {
		h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
		waitCounted = false
	}
	return admit(release)
}
