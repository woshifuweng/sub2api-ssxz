package dto

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogFromService_IncludesOpenAIWSMode(t *testing.T) {
	t.Parallel()

	wsLog := &service.UsageLog{
		RequestID:    "req_1",
		Model:        "gpt-5.3-codex",
		OpenAIWSMode: true,
	}
	httpLog := &service.UsageLog{
		RequestID:    "resp_1",
		Model:        "gpt-5.3-codex",
		OpenAIWSMode: false,
	}

	require.True(t, UsageLogFromService(wsLog).OpenAIWSMode)
	require.False(t, UsageLogFromService(httpLog).OpenAIWSMode)
	require.True(t, UsageLogFromServiceAdmin(wsLog).OpenAIWSMode)
	require.False(t, UsageLogFromServiceAdmin(httpLog).OpenAIWSMode)
}

func TestUsageLogFromService_PrefersRequestTypeForLegacyFields(t *testing.T) {
	t.Parallel()

	log := &service.UsageLog{
		RequestID:    "req_2",
		Model:        "gpt-5.3-codex",
		RequestType:  service.RequestTypeWSV2,
		Stream:       false,
		OpenAIWSMode: false,
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	require.Equal(t, "ws_v2", userDTO.RequestType)
	require.True(t, userDTO.Stream)
	require.True(t, userDTO.OpenAIWSMode)
	require.Equal(t, "ws_v2", adminDTO.RequestType)
	require.True(t, adminDTO.Stream)
	require.True(t, adminDTO.OpenAIWSMode)
}

func TestUsageCleanupTaskFromService_RequestTypeMapping(t *testing.T) {
	t.Parallel()

	requestType := int16(service.RequestTypeStream)
	task := &service.UsageCleanupTask{
		ID:     1,
		Status: service.UsageCleanupStatusPending,
		Filters: service.UsageCleanupFilters{
			RequestType: &requestType,
		},
	}

	dtoTask := UsageCleanupTaskFromService(task)
	require.NotNil(t, dtoTask)
	require.NotNil(t, dtoTask.Filters.RequestType)
	require.Equal(t, "stream", *dtoTask.Filters.RequestType)
}

func TestRequestTypeStringPtrNil(t *testing.T) {
	t.Parallel()
	require.Nil(t, requestTypeStringPtr(nil))
}

func TestUsageLogFromService_IncludesServiceTierForUserAndAdmin(t *testing.T) {
	t.Parallel()

	serviceTier := "priority"
	inboundEndpoint := "/v1/chat/completions"
	upstreamEndpoint := "/v1/responses"
	log := &service.UsageLog{
		RequestID:             "req_3",
		Model:                 "gpt-5.4",
		ServiceTier:           &serviceTier,
		InboundEndpoint:       &inboundEndpoint,
		UpstreamEndpoint:      &upstreamEndpoint,
		AccountRateMultiplier: f64Ptr(1.5),
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	require.NotNil(t, userDTO.ServiceTier)
	require.Equal(t, serviceTier, *userDTO.ServiceTier)
	require.NotNil(t, userDTO.InboundEndpoint)
	require.Equal(t, inboundEndpoint, *userDTO.InboundEndpoint)
	userJSON, err := json.Marshal(userDTO)
	require.NoError(t, err)
	require.NotContains(t, string(userJSON), "upstream_endpoint")
	require.NotNil(t, adminDTO.ServiceTier)
	require.Equal(t, serviceTier, *adminDTO.ServiceTier)
	require.NotNil(t, adminDTO.InboundEndpoint)
	require.Equal(t, inboundEndpoint, *adminDTO.InboundEndpoint)
	require.NotNil(t, adminDTO.UpstreamEndpoint)
	require.Equal(t, upstreamEndpoint, *adminDTO.UpstreamEndpoint)
	require.NotNil(t, adminDTO.AccountRateMultiplier)
	require.InDelta(t, 1.5, *adminDTO.AccountRateMultiplier, 1e-12)
}

func TestUsageLogFromService_UsesRequestedModelAndKeepsUpstreamAdminOnly(t *testing.T) {
	t.Parallel()

	upstreamModel := "claude-sonnet-4-20250514"
	upstreamResponseModel := "claude-sonnet-4-20250513"
	upstreamModelMismatch := true
	log := &service.UsageLog{
		RequestID:             "req_4",
		Model:                 upstreamModel,
		RequestedModel:        "claude-sonnet-4",
		UpstreamModel:         &upstreamModel,
		UpstreamResponseModel: &upstreamResponseModel,
		UpstreamModelMismatch: &upstreamModelMismatch,
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	require.Equal(t, "claude-sonnet-4", userDTO.Model)
	require.Equal(t, "claude-sonnet-4", adminDTO.Model)

	userJSON, err := json.Marshal(userDTO)
	require.NoError(t, err)
	require.NotContains(t, string(userJSON), "upstream_model")
	require.NotContains(t, string(userJSON), "upstream_response_model")
	require.NotContains(t, string(userJSON), "upstream_model_mismatch")

	adminJSON, err := json.Marshal(adminDTO)
	require.NoError(t, err)
	require.Contains(t, string(adminJSON), `"upstream_model":"claude-sonnet-4-20250514"`)
	require.Contains(t, string(adminJSON), `"upstream_response_model":"claude-sonnet-4-20250513"`)
	require.Contains(t, string(adminJSON), `"upstream_model_mismatch":true`)
}

func TestUsageLogFromService_KeepsUserBillingAndIPWithoutAdminCostFields(t *testing.T) {
	t.Parallel()

	ipAddress := "203.0.113.10"
	accountRateMultiplier := 1.5
	accountStatsCost := 0.21
	log := &service.UsageLog{
		RequestID:             "req_user_visible_billing",
		Model:                 "gpt-5.4",
		InputCost:             0.01,
		OutputCost:            0.02,
		CacheCreationCost:     0.03,
		CacheReadCost:         0.04,
		TotalCost:             0.10,
		ActualCost:            0.08,
		RateMultiplier:        0.8,
		IPAddress:             &ipAddress,
		AccountRateMultiplier: &accountRateMultiplier,
		AccountStatsCost:      &accountStatsCost,
	}

	userDTO := UsageLogFromService(log)
	require.Equal(t, 0.01, userDTO.InputCost)
	require.Equal(t, 0.02, userDTO.OutputCost)
	require.Equal(t, 0.03, userDTO.CacheCreationCost)
	require.Equal(t, 0.04, userDTO.CacheReadCost)
	require.Equal(t, 0.10, userDTO.TotalCost)
	require.Equal(t, 0.08, userDTO.ActualCost)
	require.Equal(t, 0.8, userDTO.RateMultiplier)
	require.NotNil(t, userDTO.IPAddress)
	require.Equal(t, ipAddress, *userDTO.IPAddress)

	userJSON, err := json.Marshal(userDTO)
	require.NoError(t, err)
	require.NotContains(t, string(userJSON), "account_rate_multiplier")
	require.NotContains(t, string(userJSON), "account_stats_cost")
	require.NotContains(t, string(userJSON), "account_cost")
}

func TestUsageLogFromService_FallsBackToLegacyModelWhenRequestedModelMissing(t *testing.T) {
	t.Parallel()

	log := &service.UsageLog{
		RequestID: "req_legacy",
		Model:     "claude-3",
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	require.Equal(t, "claude-3", userDTO.Model)
	require.Equal(t, "claude-3", adminDTO.Model)
}

func TestUsageLogFromService_IncludesImageBillingMetadataForUserAndAdmin(t *testing.T) {
	t.Parallel()

	imageSize := "4K"
	inputSize := "1024x1024"
	outputSize := "3840x2160"
	source := "output"
	log := &service.UsageLog{
		RequestID:          "req_image_metadata",
		Model:              "gpt-image-2",
		ImageCount:         2,
		ImageSize:          &imageSize,
		ImageInputSize:     &inputSize,
		ImageOutputSize:    &outputSize,
		ImageSizeSource:    &source,
		ImageSizeBreakdown: map[string]int{"4K": 2},
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	for _, got := range []*UsageLog{userDTO, &adminDTO.UsageLog} {
		require.Equal(t, 2, got.ImageCount)
		require.NotNil(t, got.ImageSize)
		require.Equal(t, imageSize, *got.ImageSize)
		require.NotNil(t, got.ImageInputSize)
		require.Equal(t, inputSize, *got.ImageInputSize)
		require.NotNil(t, got.ImageOutputSize)
		require.Equal(t, outputSize, *got.ImageOutputSize)
		require.NotNil(t, got.ImageSizeSource)
		require.Equal(t, source, *got.ImageSizeSource)
		require.Equal(t, map[string]int{"4K": 2}, got.ImageSizeBreakdown)
	}
}

func TestUsageLogFromService_PreservesHistoricalMissingImageSize(t *testing.T) {
	t.Parallel()

	log := &service.UsageLog{
		RequestID:  "req_legacy_image_missing_size",
		Model:      "gpt-image-2",
		ImageCount: 1,
		ImageSize:  nil,
	}

	dto := UsageLogFromService(log)
	require.Equal(t, 1, dto.ImageCount)
	require.Nil(t, dto.ImageSize)
	require.Nil(t, dto.ImageInputSize)
	require.Nil(t, dto.ImageOutputSize)
	require.Nil(t, dto.ImageSizeSource)
	require.Nil(t, dto.ImageSizeBreakdown)

	body, err := json.Marshal(dto)
	require.NoError(t, err)
	require.Contains(t, string(body), `"image_size":null`)
	require.NotContains(t, string(body), `"image_size":"2K"`)
}

func TestUsageLogFromService_UserResponseIsExplicitWhitelist(t *testing.T) {
	t.Parallel()

	groupID := int64(17)
	serviceTier := "priority"
	reasoningEffort := "high"
	inboundEndpoint := "/v1/chat/completions"
	durationMs := 1200
	firstTokenMs := 80
	imageSize := "1024x1024"
	imageInputSize := "512x512"
	imageOutputSize := "1024x1024"
	imageSizeSource := "output"
	mediaType := "image"
	userAgent := "safe-client/1.0"
	ipAddress := "203.0.113.10"
	sessionID := "session-safe"
	billingMode := "token"
	log := &service.UsageLog{
		ID:                        99,
		UserID:                    42,
		APIKeyID:                  7,
		AccountID:                 5,
		RequestID:                 "req-safe",
		Model:                     "served-private-model",
		RequestedModel:            "gpt-5",
		ServiceTier:               &serviceTier,
		ReasoningEffort:           &reasoningEffort,
		InboundEndpoint:           &inboundEndpoint,
		GroupID:                   &groupID,
		InputTokens:               10,
		OutputTokens:              20,
		CacheCreationTokens:       3,
		CacheReadTokens:           4,
		CacheCreation5mTokens:     1,
		CacheCreation1hTokens:     2,
		InputCost:                 0.01,
		OutputCost:                0.02,
		CacheCreationCost:         0.03,
		CacheReadCost:             0.04,
		TotalCost:                 0.10,
		ActualCost:                0.08,
		RateMultiplier:            0.8,
		LongContextBillingApplied: true,
		BillingType:               1,
		RequestType:               service.RequestTypeSync,
		DurationMs:                &durationMs,
		FirstTokenMs:              &firstTokenMs,
		ImageCount:                1,
		ImageSize:                 &imageSize,
		ImageInputSize:            &imageInputSize,
		ImageOutputSize:           &imageOutputSize,
		ImageInputTokens:          11,
		ImageInputCost:            0.11,
		ImageOutputTokens:         12,
		ImageOutputCost:           0.12,
		ImageSizeSource:           &imageSizeSource,
		ImageSizeBreakdown:        map[string]int{"1024x1024": 1},
		MediaType:                 &mediaType,
		UserAgent:                 &userAgent,
		IPAddress:                 &ipAddress,
		SessionID:                 &sessionID,
		CacheTTLOverridden:        true,
		BillingMode:               &billingMode,
		APIKey: &service.APIKey{
			ID:   7,
			Name: "safe-key-name",
			Key:  "sk-secret-must-not-leak",
		},
		Group: &service.Group{
			ID:              17,
			Name:            "safe-group-name",
			RateMultiplier:  1.7,
			FallbackGroupID: &groupID,
		},
	}

	body, err := json.Marshal(UsageLogFromService(log))
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	expectedKeys := []string{
		"actual_cost", "api_key", "billing_mode", "billing_type", "cache_creation_1h_tokens",
		"cache_creation_5m_tokens", "cache_creation_cost", "cache_creation_tokens", "cache_read_cost",
		"cache_read_tokens", "cache_ttl_overridden", "created_at", "duration_ms", "first_token_ms",
		"group", "group_id", "id", "image_count", "image_input_cost", "image_input_size",
		"image_input_tokens", "image_output_cost", "image_output_size", "image_output_tokens", "image_size",
		"image_size_breakdown", "image_size_source", "inbound_endpoint", "input_cost", "input_tokens",
		"ip_address", "long_context_billing_applied", "media_type", "model", "openai_ws_mode",
		"output_cost", "output_tokens", "rate_multiplier", "reasoning_effort", "request_id", "request_type",
		"service_tier", "session_id", "stream", "total_cost", "user_agent",
	}
	sort.Strings(expectedKeys)
	require.Equal(t, expectedKeys, keys)
	require.Equal(t, "gpt-5", payload["model"])
	require.Equal(t, map[string]any{"id": float64(7), "name": "safe-key-name"}, payload["api_key"])
	require.Equal(t, map[string]any{"id": float64(17), "name": "safe-group-name"}, payload["group"])
}

func TestUsageLogFromService_UserResponseForbidsOperationalFields(t *testing.T) {
	t.Parallel()

	upstreamModel := "upstream-private-model"
	upstreamEndpoint := "/internal/upstream"
	upstreamMismatch := true
	channelID := int64(88)
	mappingChain := "public-model→private-model"
	billingTier := "internal-tier"
	accountRateMultiplier := 1.5
	accountStatsCost := 0.07
	log := &service.UsageLog{
		UserID:                42,
		APIKeyID:              7,
		AccountID:             5,
		Model:                 "served-private-model",
		RequestedModel:        "public-model",
		UpstreamModel:         &upstreamModel,
		UpstreamResponseModel: &upstreamModel,
		UpstreamModelMismatch: &upstreamMismatch,
		UpstreamEndpoint:      &upstreamEndpoint,
		ChannelID:             &channelID,
		ModelMappingChain:     &mappingChain,
		BillingTier:           &billingTier,
		AccountRateMultiplier: &accountRateMultiplier,
		AccountStatsCost:      &accountStatsCost,
		APIKey:                &service.APIKey{ID: 7, Name: "key", Key: "sk-secret"},
		Group:                 &service.Group{ID: 17, Name: "group", FallbackGroupID: &channelID},
		Account:               &service.Account{ID: 5, Name: "internal-account"},
		User:                  &service.User{ID: 42},
		Subscription:          &service.UserSubscription{ID: 3},
	}

	body, err := json.Marshal(UsageLogFromService(log))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	for _, forbidden := range []string{
		"user_id", "api_key_id", "account_id", "account", "subscription_id", "subscription", "user",
		"served_model", "upstream_model", "upstream_response_model", "upstream_model_mismatch", "upstream_endpoint",
		"channel_id", "model_mapping_chain", "billing_tier", "account_rate_multiplier", "account_stats_cost",
	} {
		require.NotContains(t, payload, forbidden)
	}
	require.NotContains(t, string(body), "sk-secret")
	require.NotContains(t, string(body), "fallback_group_id")
}

func TestUsageLogFromService_PreservesDeletedGroupID(t *testing.T) {
	t.Parallel()

	groupID := int64(404)
	dto := UsageLogFromService(&service.UsageLog{GroupID: &groupID})
	require.Equal(t, &groupID, dto.GroupID)
	require.Nil(t, dto.Group)
}

func TestUsageLogFromServiceAdmin_RetainsOperationalFields(t *testing.T) {
	t.Parallel()

	groupID := int64(17)
	subscriptionID := int64(3)
	upstreamModel := "upstream-model"
	upstreamEndpoint := "/v1/responses"
	upstreamMismatch := true
	channelID := int64(88)
	mappingChain := "public→upstream"
	billingTier := "internal-tier"
	accountRateMultiplier := 1.5
	accountStatsCost := 0.07
	adminDTO := UsageLogFromServiceAdmin(&service.UsageLog{
		UserID:                42,
		APIKeyID:              7,
		AccountID:             5,
		GroupID:               &groupID,
		SubscriptionID:        &subscriptionID,
		UpstreamModel:         &upstreamModel,
		UpstreamResponseModel: &upstreamModel,
		UpstreamModelMismatch: &upstreamMismatch,
		UpstreamEndpoint:      &upstreamEndpoint,
		ChannelID:             &channelID,
		ModelMappingChain:     &mappingChain,
		BillingTier:           &billingTier,
		AccountRateMultiplier: &accountRateMultiplier,
		AccountStatsCost:      &accountStatsCost,
		User:                  &service.User{ID: 42},
		APIKey:                &service.APIKey{ID: 7, Name: "admin-key", Key: "sk-admin-visible"},
		Account:               &service.Account{ID: 5, Name: "admin-account"},
		Group:                 &service.Group{ID: 17, Name: "admin-group", FallbackGroupID: &groupID},
		Subscription:          &service.UserSubscription{ID: 3},
	})

	body, err := json.Marshal(adminDTO)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	for _, retained := range []string{
		"user_id", "api_key_id", "account_id", "subscription_id", "user", "api_key", "account", "group", "subscription",
		"upstream_model", "upstream_response_model", "upstream_model_mismatch", "upstream_endpoint", "channel_id",
		"model_mapping_chain", "billing_tier", "account_rate_multiplier", "account_stats_cost",
	} {
		require.Contains(t, payload, retained)
	}
	require.Contains(t, string(body), "sk-admin-visible")
	require.Contains(t, string(body), "fallback_group_id")
}

func f64Ptr(value float64) *float64 {
	return &value
}
