package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func (s *RateLimitService) shouldAutoDeleteAccountOn401(ctx context.Context) bool {
	return s != nil && s.settingService != nil && s.settingService.IsAutoDelete401AccountsEnabled(ctx)
}

func (s *RateLimitService) shouldAutoDeleteAccountOn429(ctx context.Context) bool {
	return s != nil && s.settingService != nil && s.settingService.IsAutoDelete429AccountsEnabled(ctx)
}

func (s *RateLimitService) maybeAutoDeleteAccountOn429(ctx context.Context, account *Account, responseBody []byte) bool {
	if !s.shouldAutoDeleteAccountOn429(ctx) {
		return false
	}
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	return s.autoDeleteAccountForStatus(account, http.StatusTooManyRequests, upstreamMsg)
}

func (s *RateLimitService) autoDeleteAccountForStatus(account *Account, statusCode int, upstreamMsg string) bool {
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 {
		return true
	}
	reason := strings.TrimSpace(upstreamMsg)
	if reason == "" {
		reason = fmt.Sprintf("auto delete on upstream status %d", statusCode)
	}
	deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.accountRepo.Delete(deleteCtx, account.ID); err != nil {
		slog.Warn("auto_delete_account_failed", "account_id", account.ID, "status_code", statusCode, "error", err)
		return true
	}
	slog.Warn("auto_delete_account_succeeded", "account_id", account.ID, "status_code", statusCode, "reason", reason)
	return true
}
