package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	emailSendTimeout = 30 * time.Second

	// Threshold type values
	thresholdTypeFixed      = "fixed"
	thresholdTypePercentage = "percentage"

	// Quota dimension labels
	quotaDimDaily  = "daily"
	quotaDimWeekly = "weekly"
	quotaDimTotal  = "total"

	defaultSiteName = "Sub2API"
)

// quotaDimLabels maps dimension names to display labels.
var quotaDimLabels = map[string]string{
	quotaDimDaily:  "日限额 / Daily",
	quotaDimWeekly: "周限额 / Weekly",
	quotaDimTotal:  "总限额 / Total",
}

// AccountQuotaReader provides read access to account quota data.
type AccountQuotaReader interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
}

type notificationEmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// BillingShortfallAlert contains only non-secret identifiers needed by operators.
type BillingShortfallAlert struct {
	RequestID string
	UserID    int64
	APIKeyID  int64
	Model     string
	Charged   float64
	Shortfall float64
}

// BillingShortfallNotifier is the best-effort alert boundary used by billing.
type BillingShortfallNotifier interface {
	NotifyBillingShortfall(ctx context.Context, alert BillingShortfallAlert)
}

// BalanceNotifyService handles balance and quota threshold notifications.
type BalanceNotifyService struct {
	emailService           notificationEmailSender
	settingRepo            SettingRepository
	accountRepo            AccountQuotaReader
	shortfallEmailLimiter  *slidingWindowLimiter
	alertMu                sync.Mutex
	upstreamAlertSent      map[string]time.Time
	dailySpendingAlertSent map[string]time.Time
}

// NewBalanceNotifyService creates a new BalanceNotifyService.
func NewBalanceNotifyService(emailService *EmailService, settingRepo SettingRepository, accountRepo AccountQuotaReader) *BalanceNotifyService {
	return &BalanceNotifyService{
		emailService:           emailService,
		settingRepo:            settingRepo,
		accountRepo:            accountRepo,
		shortfallEmailLimiter:  newSlidingWindowLimiter(0, time.Hour),
		upstreamAlertSent:      make(map[string]time.Time),
		dailySpendingAlertSent: make(map[string]time.Time),
	}
}

// NotifyUpstreamAccountState sends a best-effort operator alert when an
// upstream account becomes exhausted, banned, or otherwise unavailable.
// The in-memory key prevents duplicate alerts for the same account/status for
// 24 hours without adding another persistence table.
func (s *BalanceNotifyService) NotifyUpstreamAccountState(ctx context.Context, account *Account, status string) {
	if s == nil || account == nil || account.ID <= 0 {
		return
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return
	}

	emailCfg, err := s.getOpsEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil || !emailCfg.Alert.Enabled {
		return
	}
	recipients := normalizeAlertRecipients(emailCfg.Alert.Recipients)
	if len(recipients) == 0 || !shouldSendOpsAlertEmailByMinSeverity(emailCfg.Alert.MinSeverity, "critical") {
		return
	}

	now := time.Now().UTC()
	key := fmt.Sprintf("%d:%s", account.ID, status)
	if !s.claimAlertKey(s.upstreamAlertSent, key, now, 24*time.Hour) {
		return
	}

	siteName := sanitizeEmailHeader(s.getSiteName(ctx))
	subject := fmt.Sprintf("[%s][Critical] Upstream account %s", siteName, status)
	body := fmt.Sprintf(`<h2>Upstream account alert</h2>
<p>Upstream account <b>%s</b> (%d) is now <b>%s</b>.</p>
<p><b>Platform</b>: %s</p>
<p><b>Time</b>: %s</p>`,
		html.EscapeString(strings.TrimSpace(account.Name)), account.ID,
		html.EscapeString(status), html.EscapeString(strings.TrimSpace(account.Platform)),
		now.Format(time.RFC3339))
	go s.sendEmails(recipients, subject, body, "account_id", account.ID, "status", status)
}

// CheckDailySpending checks the current day's actual usage after a usage log
// is written and alerts once per user/day when the hard-coded $10 threshold is
// crossed. The repository implementation performs the SELECT aggregation.
func (s *BalanceNotifyService) CheckDailySpending(ctx context.Context, usageRepo UsageLogRepository, user *User, at time.Time) {
	const dailySpendingThreshold = 10.0
	if s == nil || usageRepo == nil || user == nil || user.ID <= 0 {
		return
	}
	if at.IsZero() {
		at = time.Now()
	}
	location := at.Location()
	dayStart := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, location)
	dayEnd := dayStart.Add(24 * time.Hour)
	stats, err := usageRepo.GetUserStatsAggregated(ctx, user.ID, dayStart, dayEnd)
	if err != nil || stats == nil || stats.TotalActualCost <= dailySpendingThreshold {
		return
	}

	emailCfg, err := s.getOpsEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil || !emailCfg.Alert.Enabled {
		return
	}
	recipients := normalizeAlertRecipients(emailCfg.Alert.Recipients)
	if len(recipients) == 0 || !shouldSendOpsAlertEmailByMinSeverity(emailCfg.Alert.MinSeverity, "critical") {
		return
	}

	now := time.Now().UTC()
	key := fmt.Sprintf("%d:%s", user.ID, dayStart.Format("2006-01-02"))
	if !s.claimAlertKey(s.dailySpendingAlertSent, key, now, 24*time.Hour) {
		return
	}

	siteName := sanitizeEmailHeader(s.getSiteName(ctx))
	subject := fmt.Sprintf("[%s][Critical] Abnormal daily spending", siteName)
	body := fmt.Sprintf(`<h2>Abnormal daily spending alert</h2>
<p>User <b>%d</b> (%s) has spent <b>$%.4f</b> today.</p>
<p><b>Threshold</b>: $%.2f</p>
<p><b>Triggered at</b>: %s</p>`,
		user.ID, html.EscapeString(strings.TrimSpace(user.Email)), stats.TotalActualCost,
		dailySpendingThreshold, now.Format(time.RFC3339))
	go s.sendEmails(recipients, subject, body, "user_id", user.ID, "daily_actual_cost", stats.TotalActualCost)
}

func (s *BalanceNotifyService) claimAlertKey(store map[string]time.Time, key string, now time.Time, ttl time.Duration) bool {
	s.alertMu.Lock()
	defer s.alertMu.Unlock()
	if store == nil {
		return false
	}
	if previous, ok := store[key]; ok && now.Sub(previous) < ttl {
		return false
	}
	store[key] = now
	return true
}

// NotifyBillingShortfall dispatches a critical operator alert without delaying
// or changing the completed billing transaction.
func (s *BalanceNotifyService) NotifyBillingShortfall(_ context.Context, alert BillingShortfallAlert) {
	if s == nil {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("gateway.billing_shortfall_alert_panicked", "recover", recovered)
			}
		}()
		s.sendBillingShortfallAlert(alert)
	}()
}

func (s *BalanceNotifyService) sendBillingShortfallAlert(alert BillingShortfallAlert) bool {
	if s == nil || s.emailService == nil || s.settingRepo == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
	defer cancel()

	emailCfg, err := s.getOpsEmailNotificationConfig(ctx)
	if err != nil {
		slog.Error("gateway.billing_shortfall_alert_not_delivered", "reason", "config_error", "error", err)
		return false
	}
	if emailCfg == nil || !emailCfg.Alert.Enabled {
		slog.Warn("gateway.billing_shortfall_alert_not_delivered", "reason", "alert_email_disabled")
		return false
	}
	recipients := normalizeAlertRecipients(emailCfg.Alert.Recipients)
	if len(recipients) == 0 {
		slog.Warn("gateway.billing_shortfall_alert_not_delivered", "reason", "no_recipients")
		return false
	}
	if !shouldSendOpsAlertEmailByMinSeverity(emailCfg.Alert.MinSeverity, "critical") {
		slog.Warn("gateway.billing_shortfall_alert_not_delivered", "reason", "severity_filtered")
		return false
	}
	if s.shortfallEmailLimiter == nil {
		s.shortfallEmailLimiter = newSlidingWindowLimiter(0, time.Hour)
	}
	s.shortfallEmailLimiter.SetLimit(emailCfg.Alert.RateLimitPerHour)

	siteName := sanitizeEmailHeader(s.getSiteName(ctx))
	subject := fmt.Sprintf("[%s][Critical] Billing shortfall detected", siteName)
	body := buildBillingShortfallEmailBody(alert, siteName)
	anySent := false
	for _, recipient := range recipients {
		if !s.shortfallEmailLimiter.Allow(time.Now().UTC()) {
			slog.Warn("gateway.billing_shortfall_alert_not_delivered", "reason", "rate_limited")
			break
		}
		if err := s.emailService.SendEmail(ctx, recipient, subject, body); err != nil {
			slog.Error("gateway.billing_shortfall_alert_delivery_failed", "to", recipient, "error", err)
			continue
		}
		anySent = true
		slog.Info("gateway.billing_shortfall_alert_delivered", "to", recipient, "request_id", alert.RequestID)
	}
	return anySent
}

func (s *BalanceNotifyService) getOpsEmailNotificationConfig(ctx context.Context) (*OpsEmailNotificationConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpsEmailNotificationConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaultOpsEmailNotificationConfig(), nil
		}
		return nil, err
	}
	cfg := &OpsEmailNotificationConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("decode ops email notification config: %w", err)
	}
	normalizeOpsEmailNotificationConfig(cfg)
	if err := validateOpsEmailNotificationConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeAlertRecipients(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		address := strings.TrimSpace(value)
		key := strings.ToLower(address)
		if address == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, address)
	}
	return result
}

func buildBillingShortfallEmailBody(alert BillingShortfallAlert, siteName string) string {
	return fmt.Sprintf(`<h2>Critical billing shortfall</h2>
<p>A completed request exhausted the user's available balance. Review the account and billing records immediately.</p>
<p><b>Site</b>: %s</p>
<p><b>Request ID</b>: %s</p>
<p><b>User ID</b>: %d</p>
<p><b>API Key ID</b>: %d</p>
<p><b>Model</b>: %s</p>
<p><b>Charged</b>: %.6f</p>
<p><b>Shortfall</b>: %.6f</p>`,
		html.EscapeString(siteName),
		html.EscapeString(strings.TrimSpace(alert.RequestID)),
		alert.UserID,
		alert.APIKeyID,
		html.EscapeString(strings.TrimSpace(alert.Model)),
		alert.Charged,
		alert.Shortfall,
	)
}

// resolveBalanceThreshold returns the effective balance threshold.
// For percentage type, it computes threshold = totalRecharged * percentage / 100.
func resolveBalanceThreshold(threshold float64, thresholdType string, totalRecharged float64) float64 {
	if thresholdType == thresholdTypePercentage && totalRecharged > 0 {
		return totalRecharged * threshold / 100
	}
	return threshold
}

// CheckBalanceAfterDeduction checks if balance crossed below threshold after deduction.
// Notification is sent only on first crossing: oldBalance >= threshold && newBalance < threshold.
func (s *BalanceNotifyService) CheckBalanceAfterDeduction(ctx context.Context, user *User, oldBalance, cost float64) {
	if !s.canNotifyBalance(user) {
		return
	}
	effectiveThreshold, rechargeURL, ok := s.resolveUserEffectiveThreshold(ctx, user)
	if !ok {
		return
	}
	newBalance := oldBalance - cost
	if !crossedDownward(oldBalance, newBalance, effectiveThreshold) {
		return
	}
	s.dispatchBalanceLowEmail(ctx, user, newBalance, effectiveThreshold, rechargeURL)
}

// canNotifyBalance checks nil guards and user-level toggle.
func (s *BalanceNotifyService) canNotifyBalance(user *User) bool {
	if user == nil || s.emailService == nil || s.settingRepo == nil {
		return false
	}
	return user.BalanceNotifyEnabled
}

// resolveUserEffectiveThreshold reads global + user config, returns the effective threshold.
// Returns ok=false when notifications should be skipped.
func (s *BalanceNotifyService) resolveUserEffectiveThreshold(ctx context.Context, user *User) (effectiveThreshold float64, rechargeURL string, ok bool) {
	globalEnabled, globalThreshold, rechargeURL := s.getBalanceNotifyConfig(ctx)
	if !globalEnabled {
		return 0, "", false
	}
	threshold := globalThreshold
	if user.BalanceNotifyThreshold != nil {
		threshold = *user.BalanceNotifyThreshold
	}
	if threshold <= 0 {
		return 0, "", false
	}
	effectiveThreshold = resolveBalanceThreshold(threshold, user.BalanceNotifyThresholdType, user.TotalRecharged)
	if effectiveThreshold <= 0 {
		return 0, "", false
	}
	return effectiveThreshold, rechargeURL, true
}

// crossedDownward returns true when oldV was at-or-above threshold but newV dropped below it.
func crossedDownward(oldV, newV, threshold float64) bool {
	return oldV >= threshold && newV < threshold
}

// dispatchBalanceLowEmail collects recipients and sends the alert in a goroutine.
func (s *BalanceNotifyService) dispatchBalanceLowEmail(ctx context.Context, user *User, newBalance, threshold float64, rechargeURL string) {
	siteName := s.getSiteName(ctx)
	recipients := s.collectBalanceNotifyRecipients(user)
	slog.Info("CheckBalanceAfterDeduction: sending notification",
		"user_id", user.ID, "recipients", recipients, "new_balance", newBalance, "threshold", threshold)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in balance notification", "recover", r)
			}
		}()
		s.sendBalanceLowEmails(recipients, user.Username, user.Email, newBalance, threshold, siteName, rechargeURL)
	}()
}

// quotaDim describes one quota dimension for notification checking.
type quotaDim struct {
	name          string
	enabled       bool
	threshold     float64
	thresholdType string // "fixed" (default) or "percentage"
	currentUsed   float64
	limit         float64
}

// resolvedThreshold converts the user-facing "remaining" threshold into a usage-based trigger point.
// The threshold represents how much quota REMAINS when the alert fires:
//   - Fixed ($): threshold=400, limit=1000 → fires when usage reaches 600 (remaining drops to 400)
//   - Percentage (%): threshold=30, limit=1000 → fires when usage reaches 700 (remaining drops to 30%)
func (d quotaDim) resolvedThreshold() float64 {
	if d.limit <= 0 {
		return 0
	}
	if d.thresholdType == thresholdTypePercentage {
		return d.limit * (1 - d.threshold/100)
	}
	return d.limit - d.threshold
}

// buildQuotaDims returns the three quota dimensions for notification checking.
func buildQuotaDims(account *Account) []quotaDim {
	return []quotaDim{
		{quotaDimDaily, account.GetQuotaNotifyDailyEnabled(), account.GetQuotaNotifyDailyThreshold(), account.GetQuotaNotifyDailyThresholdType(), account.GetQuotaDailyUsed(), account.GetQuotaDailyLimit()},
		{quotaDimWeekly, account.GetQuotaNotifyWeeklyEnabled(), account.GetQuotaNotifyWeeklyThreshold(), account.GetQuotaNotifyWeeklyThresholdType(), account.GetQuotaWeeklyUsed(), account.GetQuotaWeeklyLimit()},
		{quotaDimTotal, account.GetQuotaNotifyTotalEnabled(), account.GetQuotaNotifyTotalThreshold(), account.GetQuotaNotifyTotalThresholdType(), account.GetQuotaUsed(), account.GetQuotaLimit()},
	}
}

// buildQuotaDimsFromState builds quota dimensions using DB transaction state instead of account snapshot.
// Notification settings (enabled, threshold, thresholdType) come from the account; usage values from quotaState.
func buildQuotaDimsFromState(account *Account, state *AccountQuotaState) []quotaDim {
	return []quotaDim{
		{quotaDimDaily, account.GetQuotaNotifyDailyEnabled(), account.GetQuotaNotifyDailyThreshold(), account.GetQuotaNotifyDailyThresholdType(), state.DailyUsed, state.DailyLimit},
		{quotaDimWeekly, account.GetQuotaNotifyWeeklyEnabled(), account.GetQuotaNotifyWeeklyThreshold(), account.GetQuotaNotifyWeeklyThresholdType(), state.WeeklyUsed, state.WeeklyLimit},
		{quotaDimTotal, account.GetQuotaNotifyTotalEnabled(), account.GetQuotaNotifyTotalThreshold(), account.GetQuotaNotifyTotalThresholdType(), state.TotalUsed, state.TotalLimit},
	}
}

// CheckAccountQuotaAfterIncrement checks if any quota dimension crossed above its notify threshold.
// When quotaState is non-nil (from DB transaction RETURNING), it is used directly for threshold
// checking, avoiding a separate DB read. Otherwise it falls back to fetching fresh account data.
func (s *BalanceNotifyService) CheckAccountQuotaAfterIncrement(ctx context.Context, account *Account, cost float64, quotaState *AccountQuotaState) {
	if account == nil || s.emailService == nil || s.settingRepo == nil || cost <= 0 {
		return
	}
	if !s.isAccountQuotaNotifyEnabled(ctx) {
		return
	}
	adminEmails := s.getAccountQuotaNotifyEmails(ctx)
	if len(adminEmails) == 0 {
		return
	}

	siteName := s.getSiteName(ctx)
	var dims []quotaDim
	if quotaState != nil {
		dims = buildQuotaDimsFromState(account, quotaState)
	} else {
		freshAccount := s.fetchFreshAccount(ctx, account)
		dims = buildQuotaDims(freshAccount)
		account = freshAccount // use fresh data for alert metadata
	}
	s.checkQuotaDimCrossings(account, dims, cost, adminEmails, siteName)
}

// fetchFreshAccount loads the latest account from DB; falls back to the snapshot on error.
func (s *BalanceNotifyService) fetchFreshAccount(ctx context.Context, snapshot *Account) *Account {
	if s.accountRepo == nil {
		return snapshot
	}
	fresh, err := s.accountRepo.GetByID(ctx, snapshot.ID)
	if err != nil {
		slog.Warn("failed to fetch fresh account for quota notify, using snapshot",
			"account_id", snapshot.ID, "error", err)
		return snapshot
	}
	return fresh
}

// checkQuotaDimCrossings iterates pre-built quota dimensions and sends alerts for threshold crossings.
// Pre-increment value is reconstructed as currentUsed - cost to detect the crossing moment.
func (s *BalanceNotifyService) checkQuotaDimCrossings(account *Account, dims []quotaDim, cost float64, adminEmails []string, siteName string) {
	for _, dim := range dims {
		if !dim.enabled || dim.threshold <= 0 {
			continue
		}
		effectiveThreshold := dim.resolvedThreshold()
		if effectiveThreshold <= 0 {
			continue
		}
		newUsed := dim.currentUsed
		oldUsed := dim.currentUsed - cost
		if oldUsed < effectiveThreshold && newUsed >= effectiveThreshold {
			s.asyncSendQuotaAlert(adminEmails, account.ID, account.Name, account.Platform, dim, newUsed, effectiveThreshold, siteName)
		}
	}
}

// asyncSendQuotaAlert sends quota alert email in a goroutine with panic recovery.
func (s *BalanceNotifyService) asyncSendQuotaAlert(adminEmails []string, accountID int64, accountName, platform string, dim quotaDim, newUsed, effectiveThreshold float64, siteName string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in quota notification", "recover", r)
			}
		}()
		s.sendQuotaAlertEmails(adminEmails, accountID, accountName, platform, dim, newUsed, siteName)
	}()
}

// getBalanceNotifyConfig reads global balance notification settings.
func (s *BalanceNotifyService) getBalanceNotifyConfig(ctx context.Context) (enabled bool, threshold float64, rechargeURL string) {
	keys := []string{SettingKeyBalanceLowNotifyEnabled, SettingKeyBalanceLowNotifyThreshold, SettingKeyBalanceLowNotifyRechargeURL}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return false, 0, ""
	}
	enabled = true
	if raw, ok := settings[SettingKeyBalanceLowNotifyEnabled]; ok && strings.TrimSpace(raw) != "" {
		enabled = raw == "true"
	}
	threshold = 1.0
	if v := settings[SettingKeyBalanceLowNotifyThreshold]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			threshold = f
		}
	}
	rechargeURL = settings[SettingKeyBalanceLowNotifyRechargeURL]
	return
}

// isAccountQuotaNotifyEnabled checks the global account quota notification toggle.
func (s *BalanceNotifyService) isAccountQuotaNotifyEnabled(ctx context.Context) bool {
	val, err := s.settingRepo.GetValue(ctx, SettingKeyAccountQuotaNotifyEnabled)
	if err != nil {
		return false
	}
	return val == "true"
}

// getAccountQuotaNotifyEmails reads admin notification emails from settings,
// filtering out disabled and unverified entries.
func (s *BalanceNotifyService) getAccountQuotaNotifyEmails(ctx context.Context) []string {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAccountQuotaNotifyEmails)
	if err != nil || strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}

	entries := ParseNotifyEmails(raw)
	if len(entries) == 0 {
		return nil
	}

	return filterVerifiedEmails(entries)
}

// getSiteName reads site name from settings with fallback.
func (s *BalanceNotifyService) getSiteName(ctx context.Context) string {
	name, err := s.settingRepo.GetValue(ctx, SettingKeySiteName)
	if err != nil || name == "" {
		return defaultSiteName
	}
	return name
}

// filterVerifiedEmails returns deduplicated, non-disabled, verified emails.
func filterVerifiedEmails(entries []NotifyEmailEntry) []string {
	var recipients []string
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.Disabled || !entry.Verified {
			continue
		}
		email := strings.TrimSpace(entry.Email)
		if email == "" {
			continue
		}
		lower := strings.ToLower(email)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		recipients = append(recipients, email)
	}
	return recipients
}

// collectBalanceNotifyRecipients returns verified, non-disabled email recipients.
// Only emails with verified=true and disabled=false are included.
func (s *BalanceNotifyService) collectBalanceNotifyRecipients(user *User) []string {
	return filterVerifiedEmails(user.BalanceNotifyExtraEmails)
}

// sendEmails sends an email to all recipients with shared timeout and error logging.
func (s *BalanceNotifyService) sendEmails(recipients []string, subject, body string, logAttrs ...any) {
	if len(recipients) == 0 {
		slog.Warn("sendEmails: no recipients", "subject", subject)
		return
	}
	for _, to := range recipients {
		ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
		if err := s.emailService.SendEmail(ctx, to, subject, body); err != nil {
			attrs := append([]any{"to", to, "error", err}, logAttrs...)
			slog.Error("failed to send notification", attrs...)
		} else {
			slog.Info("notification email sent successfully", "to", to, "subject", subject)
		}
		cancel()
	}
}

// sendBalanceLowEmails sends balance low notification to all recipients.
func (s *BalanceNotifyService) sendBalanceLowEmails(recipients []string, userName, userEmail string, balance, threshold float64, siteName, rechargeURL string) {
	displayName := userName
	if displayName == "" {
		displayName = userEmail
	}
	subject := fmt.Sprintf("[%s] 余额不足提醒 / Balance Low Alert", sanitizeEmailHeader(siteName))
	body := s.buildBalanceLowEmailBody(html.EscapeString(displayName), balance, threshold, html.EscapeString(siteName), rechargeURL)
	s.sendEmails(recipients, subject, body, "user_email", userEmail, "balance", balance)
}

// sendQuotaAlertEmails sends quota alert notification to admin emails.
func (s *BalanceNotifyService) sendQuotaAlertEmails(adminEmails []string, accountID int64, accountName, platform string, dim quotaDim, used float64, siteName string) {
	dimLabel := quotaDimLabels[dim.name]
	if dimLabel == "" {
		dimLabel = dim.name
	}

	// Format the remaining-based threshold for display
	thresholdDisplay := fmt.Sprintf("$%.2f", dim.threshold)
	if dim.thresholdType == thresholdTypePercentage {
		thresholdDisplay = fmt.Sprintf("%.0f%%", dim.threshold)
	}
	remaining := dim.limit - used
	if remaining < 0 {
		remaining = 0
	}

	subject := fmt.Sprintf("[%s] 账号限额告警 / Account Quota Alert - %s", sanitizeEmailHeader(siteName), sanitizeEmailHeader(accountName))
	body := s.buildQuotaAlertEmailBody(accountID, html.EscapeString(accountName), html.EscapeString(platform), html.EscapeString(dimLabel), used, dim.limit, remaining, thresholdDisplay, html.EscapeString(siteName))
	s.sendEmails(adminEmails, subject, body, "account", accountName, "dimension", dim.name)
}

// sanitizeEmailHeader removes CR/LF characters to prevent SMTP header injection.
func sanitizeEmailHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// balanceLowEmailTemplate is the HTML template for balance low notifications.
// Format args: siteName, userName, userName, balance, threshold, threshold.
// The recharge button is appended dynamically when rechargeURL is set.
const balanceLowEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 40px 30px; text-align: center; }
        .balance { font-size: 36px; font-weight: bold; color: #dc2626; margin: 20px 0; }
        .info { color: #666; font-size: 14px; line-height: 1.6; margin-top: 20px; }
        .recharge-btn { display: inline-block; margin-top: 24px; padding: 12px 32px; background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%); color: #fff; text-decoration: none; border-radius: 6px; font-size: 16px; font-weight: bold; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header"><h1>%s</h1></div>
        <div class="content">
            <p style="font-size: 18px; color: #333;">%s，您的余额不足</p>
            <p style="color: #666;">Dear %s, your balance is running low</p>
            <div class="balance">$%.2f</div>
            <div class="info">
                <p>您的账户余额已低于提醒阈值 <strong>$%.2f</strong>。</p>
                <p>Your account balance has fallen below the alert threshold of <strong>$%.2f</strong>.</p>
                <p>请及时充值以免服务中断。</p>
                <p>Please top up to avoid service interruption.</p>
            </div>
            %s
        </div>
        <div class="footer"><p>此邮件由系统自动发送，请勿回复。</p></div>
    </div>
</body>
</html>`

// quotaAlertEmailTemplate is the HTML template for account quota alert notifications.
// Format args: siteName, accountID, accountName, platform, dimLabel, used, limitStr, remaining, thresholdDisplay.
const quotaAlertEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #ef4444 0%%, #dc2626 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 40px 30px; }
        .metric { display: flex; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid #eee; }
        .metric-label { color: #666; }
        .metric-value { font-weight: bold; color: #333; }
        .info { color: #666; font-size: 14px; line-height: 1.6; margin-top: 20px; text-align: center; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header"><h1>%s</h1></div>
        <div class="content">
            <p style="font-size: 18px; color: #333; text-align: center;">账号限额告警 / Account Quota Alert</p>
            <div class="metric"><span class="metric-label">账号 ID / Account ID</span><span class="metric-value">#%d</span></div>
            <div class="metric"><span class="metric-label">账号 / Account</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">平台 / Platform</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">维度 / Dimension</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">已使用 / Used</span><span class="metric-value">$%.2f</span></div>
            <div class="metric"><span class="metric-label">限额 / Limit</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">剩余额度 / Remaining</span><span class="metric-value">$%.2f</span></div>
            <div class="metric"><span class="metric-label">提醒阈值 / Alert Threshold</span><span class="metric-value">%s</span></div>
            <div class="info">
                <p>账号剩余额度已低于提醒阈值，请及时关注。</p>
                <p>Account remaining quota has fallen below the alert threshold.</p>
            </div>
        </div>
        <div class="footer"><p>此邮件由系统自动发送，请勿回复。</p></div>
    </div>
</body>
</html>`

// buildBalanceLowEmailBody builds HTML email for balance low notification.
func (s *BalanceNotifyService) buildBalanceLowEmailBody(userName string, balance, threshold float64, siteName, rechargeURL string) string {
	rechargeBlock := ""
	if rechargeURL != "" {
		rechargeBlock = fmt.Sprintf(`<a href="%s" class="recharge-btn">立即充值 / Top Up Now</a>`, html.EscapeString(rechargeURL))
	}
	return fmt.Sprintf(balanceLowEmailTemplate, siteName, userName, userName, balance, threshold, threshold, rechargeBlock)
}

// buildQuotaAlertEmailBody builds HTML email for account quota alert.
func (s *BalanceNotifyService) buildQuotaAlertEmailBody(accountID int64, accountName, platform, dimLabel string, used, limit, remaining float64, thresholdDisplay, siteName string) string {
	limitStr := fmt.Sprintf("$%.2f", limit)
	if limit <= 0 {
		limitStr = "无限制 / Unlimited"
	}
	return fmt.Sprintf(quotaAlertEmailTemplate, siteName, accountID, accountName, platform, dimLabel, used, limitStr, remaining, thresholdDisplay)
}
