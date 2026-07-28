package service

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBalanceCreditAmount(t *testing.T) {
	require.Equal(t, 110.0, BalanceCreditAmount(100))
	require.Equal(t, 99.99, BalanceCreditAmount(99.99))
	require.Equal(t, 100.01, BalanceCreditAmount(100.01))
}

func TestGenerateRedeemCodeIsSixteenAlphanumericCharacters(t *testing.T) {
	code, err := GenerateRedeemCode()
	require.NoError(t, err)
	require.Len(t, code, 16)
	require.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9]+$`), code)
}

func TestIsValidRedeemCodeFormat(t *testing.T) {
	require.False(t, IsValidRedeemCodeFormat(" BAL-123 "))
	require.True(t, IsValidRedeemCodeFormat("BAL-123"))
	require.False(t, IsValidRedeemCodeFormat("BAL 123"))
}

func TestRedeemCodeExpiry(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name        string
		code        RedeemCode
		wantExpired bool
		wantCanUse  bool
	}{
		{
			name:        "unused without expiry can be used",
			code:        RedeemCode{Status: StatusUnused},
			wantExpired: false,
			wantCanUse:  true,
		},
		{
			name:        "unused before expiry can be used",
			code:        RedeemCode{Status: StatusUnused, ExpiresAt: &future},
			wantExpired: false,
			wantCanUse:  true,
		},
		{
			name:        "unused after expiry cannot be used",
			code:        RedeemCode{Status: StatusUnused, ExpiresAt: &past},
			wantExpired: true,
			wantCanUse:  false,
		},
		{
			name:        "explicit expired status is expired",
			code:        RedeemCode{Status: StatusExpired},
			wantExpired: true,
			wantCanUse:  false,
		},
		{
			name:        "used code remains used even after expiry time",
			code:        RedeemCode{Status: StatusUsed, ExpiresAt: &past},
			wantExpired: false,
			wantCanUse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantExpired, tt.code.IsExpiredAt(now))
			require.Equal(t, tt.wantCanUse, tt.code.CanUse())
		})
	}
}
