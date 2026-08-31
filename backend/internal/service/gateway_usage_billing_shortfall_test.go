//go:build unit

package service

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogUsageBillingShortfall_EmitsReconciliationFields(t *testing.T) {
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	logUsageBillingShortfall(
		&UsageBillingCommand{RequestID: "req-shortfall-1", UserID: 42, APIKeyID: 7, BalanceCost: 10},
		&UsageBillingApplyResult{Applied: true, BalanceOverdrafted: true, BalanceShortfall: 5},
	)

	logLine := output.String()
	for _, want := range []string{
		"usage billing balance shortfall",
		"user_id=42",
		"api_key_id=7",
		"request_id=req-shortfall-1",
		"balance_cost=10",
		"balance_charged=5",
		"balance_shortfall=5",
	} {
		require.True(t, strings.Contains(logLine, want), "missing %q in %q", want, logLine)
	}
}
