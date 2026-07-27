//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMaybeSendAlertEmail_GuardsReturnFalse verifies that maybeSendAlertEmail
// returns false (without panic) for all early-exit guard conditions.
// The per-recipient email-failure logging path (the two continue branches
// we added) requires a live/stub SMTP endpoint and is covered by integration tests.
func TestMaybeSendAlertEmail_GuardsReturnFalse(t *testing.T) {
	t.Run("nil service", func(t *testing.T) {
		var svc *OpsAlertEvaluatorService
		require.False(t, svc.maybeSendAlertEmail(context.Background(), nil, &OpsAlertRule{}, &OpsAlertEvent{}))
	})

	t.Run("nil emailService", func(t *testing.T) {
		svc := &OpsAlertEvaluatorService{emailService: nil}
		require.False(t, svc.maybeSendAlertEmail(context.Background(), nil, &OpsAlertRule{}, &OpsAlertEvent{}))
	})

	t.Run("nil event", func(t *testing.T) {
		svc := &OpsAlertEvaluatorService{emailService: &EmailService{}}
		require.False(t, svc.maybeSendAlertEmail(context.Background(), nil, &OpsAlertRule{}, nil))
	})

	t.Run("event already emailed", func(t *testing.T) {
		svc := &OpsAlertEvaluatorService{emailService: &EmailService{}}
		require.False(t, svc.maybeSendAlertEmail(context.Background(), nil,
			&OpsAlertRule{NotifyEmail: true},
			&OpsAlertEvent{EmailSent: true},
		))
	})

	t.Run("rule notify_email=false", func(t *testing.T) {
		svc := &OpsAlertEvaluatorService{emailService: &EmailService{}}
		require.False(t, svc.maybeSendAlertEmail(context.Background(), nil,
			&OpsAlertRule{NotifyEmail: false},
			&OpsAlertEvent{EmailSent: false},
		))
	})
}
