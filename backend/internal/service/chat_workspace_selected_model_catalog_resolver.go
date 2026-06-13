package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceSelectedModelPricingConfigured = "configured"
	WorkspaceSelectedModelPricingMissing    = "missing"

	WorkspaceSelectedModelBlockReasonEmptyModel            = "selected_model_empty"
	WorkspaceSelectedModelBlockReasonChannelCatalogMissing = "selected_model_channel_catalog_missing"
	WorkspaceSelectedModelBlockReasonNotInRealChannel      = "selected_model_not_in_real_channel_catalog"
	WorkspaceSelectedModelBlockReasonUserOrGroupNotAllowed = "selected_model_user_or_group_not_allowed"
	WorkspaceSelectedModelBlockReasonPricingMissing        = "selected_model_pricing_missing"
	WorkspaceSelectedModelBlockReasonCapabilityMismatch    = "selected_model_capability_mismatch"
	WorkspaceSelectedModelBlockReasonRealGateNotAllowed    = "selected_model_real_provider_gate_not_allowed"
	WorkspaceSelectedModelBlockReasonFakeOnly              = "selected_model_fake_gate_real_provider_blocked"
)

type WorkspaceSelectedModelCatalogChannelLister interface {
	ListAvailable(ctx context.Context) ([]AvailableChannel, error)
}

type WorkspaceSelectedModelCatalogResolver interface {
	ResolveSelectedModel(ctx context.Context, input WorkspaceSelectedModelChannelCatalogResolverInput) (WorkspaceSelectedModelChannelCatalogResolution, error)
}

type WorkspaceSelectedModelChannelCatalogResolver struct {
	cfg           *config.Config
	channelLister WorkspaceSelectedModelCatalogChannelLister
}

type WorkspaceSelectedModelChannelCatalogResolverInput struct {
	UserID            int64
	AllowedGroupIDs   []int64
	SelectedModel     string
	PlannedCapability WorkspacePlannedCapability
}

type WorkspaceSelectedModelChannelCatalogResolution struct {
	Model                     string
	ModelCatalogSource        string
	ChannelID                 int64
	ChannelName               string
	GroupID                   int64
	Provider                  string
	Platform                  string
	ProviderLabel             string
	SelectedModelCapabilities []WorkspaceModelCapability
	CapabilitySource          string
	CapabilityMatched         bool
	PricingStatus             string
	UsageSupport              []string
	UserAllowed               bool
	GroupAllowed              bool
	RealGateOnAllowed         bool
	Fake                      bool
	TestOnly                  bool
	BlockReason               string
	Confidence                float64
}

func NewWorkspaceSelectedModelChannelCatalogResolver(cfg *config.Config, channelLister WorkspaceSelectedModelCatalogChannelLister) *WorkspaceSelectedModelChannelCatalogResolver {
	return &WorkspaceSelectedModelChannelCatalogResolver{cfg: cfg, channelLister: channelLister}
}

func (r *WorkspaceSelectedModelChannelCatalogResolver) ResolveSelectedModel(ctx context.Context, input WorkspaceSelectedModelChannelCatalogResolverInput) (WorkspaceSelectedModelChannelCatalogResolution, error) {
	selectedModel := strings.TrimSpace(input.SelectedModel)
	if selectedModel == "" {
		return WorkspaceSelectedModelChannelCatalogResolution{
			Model:              selectedModel,
			ModelCatalogSource: WorkspaceModelCatalogSourceUnknown,
			BlockReason:        WorkspaceSelectedModelBlockReasonEmptyModel,
		}, nil
	}

	if r == nil {
		return staticFallbackSelectedModelResolution(selectedModel, input.PlannedCapability, WorkspaceSelectedModelBlockReasonChannelCatalogMissing), nil
	}

	if strings.EqualFold(selectedModel, WorkspaceImageProviderFakeModel) {
		return r.resolveFakeModel(input.UserID)
	}

	if r.channelLister == nil {
		return staticFallbackSelectedModelResolution(selectedModel, input.PlannedCapability, WorkspaceSelectedModelBlockReasonChannelCatalogMissing), nil
	}

	channels, err := r.channelLister.ListAvailable(ctx)
	if err != nil {
		return WorkspaceSelectedModelChannelCatalogResolution{}, err
	}

	resolution, found := r.resolveRealChannelModel(channels, selectedModel, input)
	if !found {
		return staticFallbackSelectedModelResolution(selectedModel, input.PlannedCapability, WorkspaceSelectedModelBlockReasonNotInRealChannel), nil
	}
	return resolution, nil
}

func (r *WorkspaceSelectedModelChannelCatalogResolver) resolveFakeModel(userID int64) (WorkspaceSelectedModelChannelCatalogResolution, error) {
	exposure := workspaceImageFakeModelExposureFromConfig(r.config(), userID)
	if !exposure.Enabled {
		return WorkspaceSelectedModelChannelCatalogResolution{
			Model:              WorkspaceImageProviderFakeModel,
			ModelCatalogSource: WorkspaceModelCatalogSourceUnknown,
			BlockReason:        WorkspaceSelectedModelBlockReasonNotInRealChannel,
		}, nil
	}
	return WorkspaceSelectedModelChannelCatalogResolution{
		Model:                     exposure.Model,
		ModelCatalogSource:        exposure.ModelCatalogSource,
		ProviderLabel:             exposure.ProviderLabel,
		Platform:                  exposure.Platform,
		SelectedModelCapabilities: cloneWorkspaceModelCapabilities(exposure.Capabilities),
		CapabilitySource:          exposure.CapabilitySource,
		CapabilityMatched:         true,
		PricingStatus:             WorkspaceSelectedModelPricingMissing,
		UserAllowed:               true,
		GroupAllowed:              false,
		RealGateOnAllowed:         false,
		Fake:                      exposure.Fake,
		TestOnly:                  exposure.TestOnly,
		BlockReason:               WorkspaceSelectedModelBlockReasonFakeOnly,
		Confidence:                0.95,
	}, nil
}

func (r *WorkspaceSelectedModelChannelCatalogResolver) resolveRealChannelModel(channels []AvailableChannel, selectedModel string, input WorkspaceSelectedModelChannelCatalogResolverInput) (WorkspaceSelectedModelChannelCatalogResolution, bool) {
	for _, channel := range channels {
		group, groupAllowed := workspaceSelectedModelAllowedGroup(channel.Groups, input.AllowedGroupIDs)
		if !groupAllowed {
			continue
		}
		for _, model := range channel.SupportedModels {
			if !strings.EqualFold(strings.TrimSpace(model.Name), selectedModel) {
				continue
			}
			return r.realChannelResolution(channel, group, model, input), true
		}
	}
	return WorkspaceSelectedModelChannelCatalogResolution{}, false
}

func (r *WorkspaceSelectedModelChannelCatalogResolver) realChannelResolution(channel AvailableChannel, group AvailableGroupRef, model SupportedModel, input WorkspaceSelectedModelChannelCatalogResolverInput) WorkspaceSelectedModelChannelCatalogResolution {
	modelMetadata := ResolveWorkspaceModelCapabilities(model.Name, WorkspaceModelCapabilityHints{
		Platform: strings.TrimSpace(model.Platform),
	})
	resolution := WorkspaceSelectedModelChannelCatalogResolution{
		Model:                     strings.TrimSpace(model.Name),
		ModelCatalogSource:        WorkspaceModelCatalogSourceRealChannel,
		ChannelID:                 channel.ID,
		ChannelName:               strings.TrimSpace(channel.Name),
		GroupID:                   group.ID,
		Platform:                  strings.TrimSpace(model.Platform),
		SelectedModelCapabilities: cloneWorkspaceModelCapabilities(modelMetadata.Capabilities),
		CapabilitySource:          modelMetadata.CapabilitySource,
		CapabilityMatched:         workspaceSelectedModelCapabilityMatches(input.PlannedCapability, modelMetadata),
		PricingStatus:             WorkspaceSelectedModelPricingMissing,
		UsageSupport:              workspaceSelectedModelUsageSupport(model.Pricing),
		UserAllowed:               input.UserID > 0,
		GroupAllowed:              group.ID > 0,
		Confidence:                modelMetadata.Confidence,
	}
	if model.Pricing != nil {
		resolution.PricingStatus = WorkspaceSelectedModelPricingConfigured
	}
	if !resolution.UserAllowed || !resolution.GroupAllowed {
		resolution.BlockReason = WorkspaceSelectedModelBlockReasonUserOrGroupNotAllowed
		return resolution
	}
	if model.Pricing == nil {
		resolution.BlockReason = WorkspaceSelectedModelBlockReasonPricingMissing
		return resolution
	}
	if !resolution.CapabilityMatched {
		resolution.BlockReason = WorkspaceSelectedModelBlockReasonCapabilityMismatch
		return resolution
	}

	if input.PlannedCapability == WorkspacePlannedCapabilityImageGeneration {
		exposure := workspaceImageRealChannelModelExposureFromConfig(r.config(), input.UserID, model)
		if exposure.Model == "" || exposure.ModelCatalogSource != WorkspaceModelCatalogSourceRealChannel {
			resolution.BlockReason = WorkspaceSelectedModelBlockReasonRealGateNotAllowed
			return resolution
		}
		resolution.Provider = exposure.Provider
		resolution.ProviderLabel = exposure.ProviderLabel
		resolution.Platform = firstNonEmptyWorkspaceValue(exposure.Platform, resolution.Platform)
		resolution.SelectedModelCapabilities = cloneWorkspaceModelCapabilities(exposure.Capabilities)
		resolution.CapabilitySource = exposure.CapabilitySource
		resolution.CapabilityMatched = true
		resolution.RealGateOnAllowed = true
		resolution.Confidence = 0.95
	}
	return resolution
}

func (r *WorkspaceSelectedModelChannelCatalogResolver) config() *config.Config {
	if r == nil {
		return nil
	}
	return r.cfg
}

func staticFallbackSelectedModelResolution(model string, planned WorkspacePlannedCapability, blockReason string) WorkspaceSelectedModelChannelCatalogResolution {
	metadata := ResolveWorkspaceModelCapabilities(model, WorkspaceModelCapabilityHints{})
	return WorkspaceSelectedModelChannelCatalogResolution{
		Model:                     strings.TrimSpace(model),
		ModelCatalogSource:        WorkspaceModelCatalogSourceStaticFallback,
		SelectedModelCapabilities: cloneWorkspaceModelCapabilities(metadata.Capabilities),
		CapabilitySource:          metadata.CapabilitySource,
		CapabilityMatched:         workspaceSelectedModelCapabilityMatches(planned, metadata),
		PricingStatus:             WorkspaceSelectedModelPricingMissing,
		UserAllowed:               false,
		GroupAllowed:              false,
		RealGateOnAllowed:         false,
		BlockReason:               blockReason,
		Confidence:                metadata.Confidence,
	}
}

func workspaceSelectedModelCapabilityMatches(planned WorkspacePlannedCapability, metadata WorkspaceModelCapabilityMetadata) bool {
	matched, _ := WorkspaceModelCapabilitiesMatch(planned, metadata)
	return matched
}

func workspaceSelectedModelAllowedGroup(groups []AvailableGroupRef, allowedGroupIDs []int64) (AvailableGroupRef, bool) {
	for _, group := range groups {
		if group.ID <= 0 {
			continue
		}
		if workspaceInt64ListContains(allowedGroupIDs, group.ID) {
			return group, true
		}
	}
	return AvailableGroupRef{}, false
}

func workspaceSelectedModelUsageSupport(pricing *ChannelModelPricing) []string {
	if pricing == nil {
		return nil
	}
	support := make([]string, 0, 2)
	if pricing.BillingMode == BillingModeImage || pricing.ImageOutputPrice != nil || pricing.PerRequestPrice != nil || len(pricing.Intervals) > 0 {
		support = append(support, "image_count")
	}
	if len(pricing.Intervals) > 0 {
		support = append(support, "image_size")
	}
	return support
}

func cloneWorkspaceModelCapabilities(values []WorkspaceModelCapability) []WorkspaceModelCapability {
	if len(values) == 0 {
		return nil
	}
	out := make([]WorkspaceModelCapability, len(values))
	copy(out, values)
	return out
}

func (r WorkspaceSelectedModelChannelCatalogResolution) ModelCapabilityMetadata() WorkspaceModelCapabilityMetadata {
	return WorkspaceModelCapabilityMetadata{
		ModelName:          r.Model,
		ProviderLabel:      r.ProviderLabel,
		Provider:           r.Provider,
		Platform:           r.Platform,
		Capabilities:       cloneWorkspaceModelCapabilities(r.SelectedModelCapabilities),
		CapabilitySource:   r.CapabilitySource,
		ModelCatalogSource: r.ModelCatalogSource,
		Confidence:         r.Confidence,
	}
}

func mergeWorkspaceSelectedModelCatalogMetadata(metadata map[string]any, resolution WorkspaceSelectedModelChannelCatalogResolution) map[string]any {
	out := make(map[string]any, len(metadata)+18)
	for key, value := range metadata {
		out[key] = value
	}
	if resolution.ModelCatalogSource != "" {
		out["model_catalog_source"] = resolution.ModelCatalogSource
	}
	if resolution.ChannelID > 0 {
		out["model_channel_id"] = resolution.ChannelID
	}
	if resolution.ChannelName != "" {
		out["model_channel_name"] = resolution.ChannelName
	}
	if resolution.GroupID > 0 {
		out["model_group_id"] = resolution.GroupID
	}
	if resolution.ProviderLabel != "" {
		out["model_provider_label"] = resolution.ProviderLabel
	}
	if resolution.Provider != "" {
		out["model_provider"] = resolution.Provider
	}
	if resolution.Platform != "" {
		out["model_platform"] = resolution.Platform
	}
	if len(resolution.SelectedModelCapabilities) > 0 {
		out["selected_model_capabilities"] = workspaceModelCapabilityStrings(resolution.SelectedModelCapabilities)
	}
	if resolution.CapabilitySource != "" {
		out["model_capability_source"] = resolution.CapabilitySource
	}
	if resolution.Confidence > 0 {
		out["model_capability_confidence"] = resolution.Confidence
	}
	if resolution.PricingStatus != "" {
		out["pricing_status"] = resolution.PricingStatus
	}
	if len(resolution.UsageSupport) > 0 {
		out["usage_support"] = append([]string(nil), resolution.UsageSupport...)
	}
	out["model_user_allowed"] = resolution.UserAllowed
	out["model_group_allowed"] = resolution.GroupAllowed
	out["real_gate_on_allowed"] = resolution.RealGateOnAllowed
	out["model_fake"] = resolution.Fake
	out["model_test_only"] = resolution.TestOnly
	if resolution.BlockReason != "" {
		out["model_catalog_block_reason"] = resolution.BlockReason
	}
	return out
}
