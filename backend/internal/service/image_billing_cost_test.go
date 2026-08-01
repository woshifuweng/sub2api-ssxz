package service

import (
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/stretchr/testify/require"
)

type imageBillingLogSink struct {
	mu     sync.Mutex
	events []*logger.LogEvent
}

func (s *imageBillingLogSink) WriteLogEvent(event *logger.LogEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *imageBillingLogSink) contains(message string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, event := range s.events {
		if event.Message == message {
			return true
		}
	}
	return false
}

func TestCalculateImageCostForAPIKeyExplicitPriceBypassesTextMultiplier(t *testing.T) {
	svc := &BillingService{}
	price4K := 0.12
	apiKey := &APIKey{Group: &Group{ID: 7, ImagePrice4K: &price4K}}

	cost := calculateImageCostForAPIKey(svc, "gpt-image-2", ImageBillingSize4K, 1, apiKey, 0.25, "")

	require.InDelta(t, 0.12, cost.TotalCost, 1e-10)
	require.InDelta(t, 0.12, cost.ActualCost, 1e-10)
}

func TestCalculateImageCostForAPIKeyFallbackPreservesTextMultiplier(t *testing.T) {
	svc := &BillingService{}
	price1K := 0.04
	apiKey := &APIKey{Group: &Group{ID: 7, ImagePrice1K: &price1K}}

	cost := calculateImageCostForAPIKey(svc, "gpt-image-2", ImageBillingSize4K, 1, apiKey, 0.25, "")

	require.Greater(t, cost.TotalCost, 0.0)
	require.InDelta(t, cost.TotalCost*0.25, cost.ActualCost, 1e-10)
}

func TestCalculateImageCostForAPIKeyLowExplicitPriceDoesNotChangeCost(t *testing.T) {
	require.NoError(t, logger.Init(logger.InitOptions{
		Level:       "debug",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output: logger.OutputOptions{
			ToStdout: false,
			ToFile:   false,
		},
		Sampling: logger.SamplingOptions{Enabled: false},
	}))
	sink := &imageBillingLogSink{}
	logger.SetSink(sink)
	t.Cleanup(func() { logger.SetSink(nil) })

	svc := &BillingService{}
	price4K := 0.01
	apiKey := &APIKey{Group: &Group{ID: 7, ImagePrice4K: &price4K}}

	cost := calculateImageCostForAPIKey(svc, "gpt-image-2", ImageBillingSize4K, 2, apiKey, 0.25, "")

	require.InDelta(t, 0.02, cost.TotalCost, 1e-10)
	require.InDelta(t, 0.02, cost.ActualCost, 1e-10)
	require.True(t, sink.contains("image_billing.explicit_price_below_upstream_floor"))
}

func TestCalculateImageCostForAPIKeyGPTImage2AutoUses4KPrice(t *testing.T) {
	svc := &BillingService{}
	price2K := 0.06
	price4K := 0.12
	apiKey := &APIKey{Group: &Group{
		ID:           7,
		ImagePrice2K: &price2K,
		ImagePrice4K: &price4K,
	}}

	cost := calculateImageCostForAPIKey(svc, "gpt-image-2", "auto", 1, apiKey, 0.25, "")

	require.InDelta(t, 0.12, cost.TotalCost, 1e-10)
	require.InDelta(t, 0.12, cost.ActualCost, 1e-10)
}

func TestCalculateImageCostForAPIKeyOtherModelAutoKeepsGeneric2KDefault(t *testing.T) {
	svc := &BillingService{}
	price2K := 0.06
	price4K := 0.12
	apiKey := &APIKey{Group: &Group{
		ID:           7,
		ImagePrice2K: &price2K,
		ImagePrice4K: &price4K,
	}}

	cost := calculateImageCostForAPIKey(svc, "gemini-image", "auto", 1, apiKey, 0.25, "")

	require.InDelta(t, 0.06, cost.TotalCost, 1e-10)
	require.InDelta(t, 0.06, cost.ActualCost, 1e-10)
}

func TestImageQualityMultiplier(t *testing.T) {
	tests := []struct {
		quality string
		want    float64
	}{
		{quality: "low", want: 1.0},
		{quality: "auto", want: 1.3},
		{quality: "medium", want: 1.3},
		{quality: "high", want: 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.quality, func(t *testing.T) {
			require.InDelta(t, tt.want, imageQualityMultiplier(tt.quality), 1e-10)
		})
	}
}
