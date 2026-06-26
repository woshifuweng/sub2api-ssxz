package handler

import (
	"encoding/json"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestBuildPublicOrderResultOmitsProviderSnapshot(t *testing.T) {
	order := &dbent.PaymentOrder{
		ID:          42,
		OutTradeNo:  "sub2_public_result",
		Amount:      10,
		PayAmount:   10,
		FeeRate:     0,
		PaymentType: payment.TypeAlipay,
		OrderType:   payment.OrderTypeBalance,
		Status:      payment.OrderStatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
		ProviderSnapshot: map[string]any{
			"provider_key":         payment.TypeAlipay,
			"provider_instance_id": "7",
			"merchant_id":          "merchant-secret",
			"merchant_app_id":      "app-secret",
		},
	}

	payload, err := json.Marshal(buildPublicOrderResult(order))
	require.NoError(t, err)

	body := string(payload)
	require.Contains(t, body, "sub2_public_result")
	require.NotContains(t, body, "ProviderSnapshot")
	require.NotContains(t, body, "provider_snapshot")
	require.NotContains(t, body, "merchant-secret")
	require.NotContains(t, body, "app-secret")
}
