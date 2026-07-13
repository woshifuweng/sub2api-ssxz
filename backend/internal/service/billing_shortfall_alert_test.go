//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type billingShortfallSettingRepoStub struct {
	SettingRepository
	value string
	err   error
}

func (s *billingShortfallSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != SettingKeyOpsEmailNotificationConfig {
		return "", ErrSettingNotFound
	}
	return s.value, s.err
}

type billingShortfallEmailSenderStub struct {
	calls []billingShortfallEmailCall
	err   error
}

type billingShortfallEmailCall struct {
	to      string
	subject string
	body    string
}

func (s *billingShortfallEmailSenderStub) SendEmail(_ context.Context, to, subject, body string) error {
	s.calls = append(s.calls, billingShortfallEmailCall{to: to, subject: subject, body: body})
	return s.err
}

func TestBalanceNotifyServiceSendBillingShortfallAlertDeliversToConfiguredRecipients(t *testing.T) {
	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = true
	cfg.Alert.Recipients = []string{"ops@example.com", " OPS@example.com ", "backup@example.com"}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	emailSender := &billingShortfallEmailSenderStub{}
	svc := &BalanceNotifyService{
		emailService:          emailSender,
		settingRepo:           &billingShortfallSettingRepoStub{value: string(raw)},
		shortfallEmailLimiter: newSlidingWindowLimiter(0, time.Hour),
	}

	sent := svc.sendBillingShortfallAlert(BillingShortfallAlert{
		RequestID: "req-shortfall-1",
		UserID:    42,
		APIKeyID:  7,
		Model:     "configured-model",
		Charged:   0.75,
		Shortfall: 0.25,
	})

	require.True(t, sent)
	require.Len(t, emailSender.calls, 2)
	require.Equal(t, "ops@example.com", emailSender.calls[0].to)
	require.Contains(t, emailSender.calls[0].subject, "Critical")
	require.Contains(t, emailSender.calls[0].body, "req-shortfall-1")
	require.Contains(t, emailSender.calls[0].body, "Shortfall")
}

func TestBalanceNotifyServiceSendBillingShortfallAlertSkipsWhenRecipientsMissing(t *testing.T) {
	cfg := defaultOpsEmailNotificationConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	emailSender := &billingShortfallEmailSenderStub{}
	svc := &BalanceNotifyService{
		emailService:          emailSender,
		settingRepo:           &billingShortfallSettingRepoStub{value: string(raw)},
		shortfallEmailLimiter: newSlidingWindowLimiter(0, time.Hour),
	}

	require.False(t, svc.sendBillingShortfallAlert(BillingShortfallAlert{Shortfall: 1}))
	require.Empty(t, emailSender.calls)
}

func TestBalanceNotifyServiceSendBillingShortfallAlertReportsDeliveryFailure(t *testing.T) {
	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Recipients = []string{"ops@example.com"}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	emailSender := &billingShortfallEmailSenderStub{err: errors.New("smtp unavailable")}
	svc := &BalanceNotifyService{
		emailService:          emailSender,
		settingRepo:           &billingShortfallSettingRepoStub{value: string(raw)},
		shortfallEmailLimiter: newSlidingWindowLimiter(0, time.Hour),
	}

	require.False(t, svc.sendBillingShortfallAlert(BillingShortfallAlert{Shortfall: 1}))
	require.Len(t, emailSender.calls, 1)
}
