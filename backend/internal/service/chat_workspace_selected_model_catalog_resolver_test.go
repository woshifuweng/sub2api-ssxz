package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type testWorkspaceSelectedModelChannelLister struct {
	channels []AvailableChannel
	err      error
}

func (l testWorkspaceSelectedModelChannelLister) ListAvailable(_ context.Context) ([]AvailableChannel, error) {
	return l.channels, l.err
}

func TestWorkspaceSelectedModelChannelCatalogResolverRealChannelAllowsImageModel(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, "gpt-image-1", resolution.Model)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, resolution.ModelCatalogSource)
	require.Equal(t, "workspace-openai-compatible-image-staging", resolution.ProviderLabel)
	require.Equal(t, WorkspaceImageRealModelProvider, resolution.Provider)
	require.Contains(t, resolution.SelectedModelCapabilities, WorkspaceModelCapabilityImageGeneration)
	require.Equal(t, WorkspaceSelectedModelPricingConfigured, resolution.PricingStatus)
	require.Contains(t, resolution.UsageSupport, "image_count")
	require.True(t, resolution.UserAllowed)
	require.True(t, resolution.GroupAllowed)
	require.True(t, resolution.CapabilityMatched)
	require.True(t, resolution.RealGateOnAllowed)
	require.Empty(t, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverEnvAllowedModelWithoutChannelBlocksRealProvider(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceStaticFallback, resolution.ModelCatalogSource)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonNotInRealChannel, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverBlocksStaticFallbackForRealProvider(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), nil)

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceStaticFallback, resolution.ModelCatalogSource)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonChannelCatalogMissing, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverUnknownModelBlocks(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "unknown-model",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceStaticFallback, resolution.ModelCatalogSource)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonNotInRealChannel, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverBlocksUserOrGroupNotAllowed(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{20},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceStaticFallback, resolution.ModelCatalogSource)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonNotInRealChannel, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverBlocksRealGateUserNotAllowed(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            2,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, resolution.ModelCatalogSource)
	require.True(t, resolution.GroupAllowed)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonRealGateNotAllowed, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverBlocksPricingMissing(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, false, WorkspaceModelCapabilityImageGeneration)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "gpt-image-1",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, resolution.ModelCatalogSource)
	require.Equal(t, WorkspaceSelectedModelPricingMissing, resolution.PricingStatus)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonPricingMissing, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverBlocksCapabilityMismatch(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("deepseek-v4-flash", 10, true, WorkspaceModelCapabilityTextChat)},
	})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     "deepseek-v4-flash",
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, resolution.ModelCatalogSource)
	require.False(t, resolution.CapabilityMatched)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonCapabilityMismatch, resolution.BlockReason)
}

func TestWorkspaceSelectedModelChannelCatalogResolverFakeModelOnlyAllowsFakeGate(t *testing.T) {
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogFakeConfig(), testWorkspaceSelectedModelChannelLister{})

	resolution, err := resolver.ResolveSelectedModel(context.Background(), WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            1,
		AllowedGroupIDs:   []int64{10},
		SelectedModel:     WorkspaceImageProviderFakeModel,
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceFakeGate, resolution.ModelCatalogSource)
	require.True(t, resolution.Fake)
	require.True(t, resolution.TestOnly)
	require.False(t, resolution.RealGateOnAllowed)
	require.Equal(t, WorkspaceSelectedModelBlockReasonFakeOnly, resolution.BlockReason)
	require.Contains(t, resolution.SelectedModelCapabilities, WorkspaceModelCapabilityImageGeneration)
}

func TestChatWorkspaceServiceUsesSelectedModelCatalogResolverMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})
	svc := NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, nil, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "帮我生成一张高级感产品图",
		Model:           "gpt-image-1",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, msg.Metadata["model_catalog_source"])
	require.Equal(t, "workspace-openai-compatible-image-staging", msg.Metadata["model_provider_label"])
	require.Equal(t, true, msg.Metadata["real_gate_on_allowed"])
	require.NotContains(t, marshaledWorkspaceSelectedModelCatalogMetadata(t, msg.Metadata), "Authorization")
	require.NotContains(t, marshaledWorkspaceSelectedModelCatalogMetadata(t, msg.Metadata), "cookie")
	require.NotContains(t, marshaledWorkspaceSelectedModelCatalogMetadata(t, msg.Metadata), "secret")
}

func TestChatWorkspaceServiceAcceptsRealChannelModelOutsideStaticNames(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("claude-3-5-sonnet", 10, true, WorkspaceModelCapabilityTextChat)},
	})
	svc := NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, nil, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "claude-3-5-sonnet",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, "claude-3-5-sonnet", msg.Model)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, msg.Metadata["model_catalog_source"])
	require.Equal(t, WorkspaceSelectedModelPricingConfigured, msg.Metadata["pricing_status"])
}

func TestChatWorkspaceServiceRejectsModelMissingFromRuntimeCatalog(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("deepseek-v4-flash", 10, true, WorkspaceModelCapabilityTextChat)},
	})
	svc := NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, nil, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, err = svc.AppendMessage(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "env-only-model",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)
}

func testWorkspaceSelectedModelCatalogRealImageConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "staging"},
		Workspace: config.WorkspaceConfig{
			ImageRealProvider: config.WorkspaceImageRealProviderConfig{
				Enabled:               true,
				KillSwitch:            true,
				StagingOnly:           true,
				Environment:           "staging",
				ProviderLabel:         "workspace-openai-compatible-image-staging",
				AllowedUserIDs:        []int64{1},
				AllowedModels:         []string{"gpt-image-1"},
				AllowedProviderLabels: []string{"workspace-openai-compatible-image-staging"},
				MaxRequestsPerTestRun: 1,
			},
		},
	}
}

func testWorkspaceSelectedModelCatalogFakeConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "staging"},
		Workspace: config.WorkspaceConfig{
			ImageExecution: config.WorkspaceImageExecutionConfig{
				Enabled:               true,
				KillSwitch:            true,
				FakeProviderEnabled:   true,
				AllowedUserIDs:        []int64{1},
				AllowedModels:         []string{WorkspaceImageProviderFakeModel},
				AllowedProviderLabels: []string{WorkspaceImageProviderFakeLabel},
				MaxRequestsPerTestRun: 3,
			},
		},
	}
}

func testWorkspaceSelectedModelAvailableChannel(modelName string, groupID int64, withPricing bool, capability WorkspaceModelCapability) AvailableChannel {
	model := SupportedModel{Name: modelName, Platform: "openai"}
	if withPricing {
		model.Pricing = testWorkspaceSelectedModelPricing(modelName, capability)
	}
	return AvailableChannel{
		ID:     100,
		Name:   "OpenAI image channel",
		Status: StatusActive,
		Groups: []AvailableGroupRef{
			{ID: groupID, Name: "beta", Platform: "openai"},
		},
		SupportedModels: []SupportedModel{model},
	}
}

func testWorkspaceSelectedModelPricing(modelName string, capability WorkspaceModelCapability) *ChannelModelPricing {
	price := 0.01
	mode := BillingModeToken
	if capability == WorkspaceModelCapabilityImageGeneration {
		mode = BillingModeImage
	}
	return &ChannelModelPricing{
		Platform:         "openai",
		Models:           []string{modelName},
		BillingMode:      mode,
		ImageOutputPrice: &price,
	}
}

func marshaledWorkspaceSelectedModelCatalogMetadata(t *testing.T, metadata map[string]any) string {
	t.Helper()
	data, err := json.Marshal(metadata)
	require.NoError(t, err)
	return string(data)
}
