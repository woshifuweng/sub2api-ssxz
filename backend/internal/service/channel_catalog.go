package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ListAvailableCatalog projects existing routing and billing configuration. It
// never discovers providers on page load or changes an account/key permission.
// One view per authorized group prevents another group's price/support from
// being advertised as this group's. Groups without a channel are included too.
func (s *ChannelService) ListAvailableCatalog(ctx context.Context, groups []Group, repo AccountRepository, billing *BillingService) ([]AvailableChannel, error) {
	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog channels: %w", err)
	}
	platformSet := make(map[string]bool)
	for _, g := range groups {
		for _, platform := range matchingPlatforms(g.Platform) {
			platformSet[platform] = true
		}
	}
	if platformSet[PlatformAnthropic] || platformSet[PlatformGemini] {
		platformSet[PlatformAntigravity] = true
	}
	platforms := make([]string, 0, len(platformSet))
	for platform := range platformSet {
		platforms = append(platforms, platform)
	}
	accounts, err := repo.ListModelAvailabilityCandidates(ctx, nil, platforms, true)
	if err != nil {
		return nil, fmt.Errorf("catalog accounts: %w", err)
	}
	return buildGroupCatalog(ctx, channels, groups, accounts, billing), nil
}

func buildGroupCatalog(ctx context.Context, channels []Channel, groups []Group, accounts []Account, billing *BillingService) []AvailableChannel {
	platforms := make(map[int64]string, len(groups))
	for _, g := range groups {
		platforms[g.ID] = g.Platform
	}
	// Reuse the runtime mapping/restriction and price resolver against one frozen
	// snapshot; don't independently reimplement their precedence rules.
	snapshot := populateChannelCache(channels, platforms)
	local := &ChannelService{}
	local.cache.Store(snapshot)
	resolver := NewModelPricingResolver(local, billing)
	out := make([]AvailableChannel, 0, len(groups))
	for i := range groups {
		g := &groups[i]
		if g.Status != StatusActive {
			continue
		}
		ch := snapshot.channelByGroupID[g.ID]
		if ch != nil && !ch.IsActive() {
			continue
		}
		candidates := map[string]SupportedModel{}
		add := func(model SupportedModel) {
			if model.Name != "" && !strings.ContainsAny(model.Name, "*?") && isPlatformPricingMatch(g.Platform, model.Platform) {
				candidates[model.Platform+":"+strings.ToLower(model.Name)] = model
			}
		}
		for _, m := range ch.SupportedModels() {
			add(m)
		}
		for _, pricing := range g.ModelPricing {
			platform := pricing.Platform
			if platform == "" {
				platform = g.Platform
			}
			for _, model := range pricing.Models {
				add(SupportedModel{Name: model, Platform: platform})
			}
		}
		pool := make([]*Account, 0)
		for j := range accounts {
			a := &accounts[j]
			bound := false
			for _, id := range a.GroupIDs {
				bound = bound || id == g.ID
			}
			if !bound || a.Status != StatusActive || !a.Schedulable {
				continue
			}
			pool = append(pool, a)
			for model := range a.GetModelMapping() {
				add(SupportedModel{Name: model, Platform: a.Platform})
			}
		}
		view := AvailableChannel{Name: g.Name, Description: g.Description, Status: StatusActive,
			Groups: []AvailableGroupRef{{ID: g.ID, Name: g.Name, Platform: g.Platform,
				ImageRateIndependent: g.ImageRateIndependent, ImageRateMultiplier: g.ImageRateMultiplier,
				SubscriptionType: g.SubscriptionType, RateMultiplier: g.RateMultiplier,
				PeakRateEnabled: g.PeakRateEnabled, PeakStart: g.PeakStart, PeakEnd: g.PeakEnd,
				PeakRateMultiplier: g.PeakRateMultiplier, IsExclusive: g.IsExclusive}}}
		for _, model := range candidates {
			modelCtx := WithResolvedTargetPlatform(ctx, model.Platform)
			mapping, _ := local.ResolveChannelMappingAndRestrict(modelCtx, &g.ID, model.Name)
			restrictionModel := billingModelForRestriction(mapping.BillingModelSource, model.Name, mapping.MappedModel)
			if restrictionModel != "" && local.IsModelRestricted(modelCtx, g.ID, restrictionModel) {
				continue
			}
			supported := false
			for _, a := range pool {
				platformMatch := a.Platform == model.Platform || (a.Platform == PlatformAntigravity &&
					a.IsMixedSchedulingEnabled() && (model.Platform == PlatformAnthropic || model.Platform == PlatformGemini))
				if platformMatch && a.IsModelSupported(mapping.MappedModel) {
					if mapping.BillingModelSource == BillingModelSourceUpstream && local.IsModelRestricted(modelCtx, g.ID, a.GetMappedModel(mapping.MappedModel)) {
						continue
					}
					supported = true
					break
				}
			}
			if !supported {
				continue
			}
			billingModel := mapping.MappedModel
			if mapping.BillingModelSource == BillingModelSourceRequested {
				billingModel = model.Name
			}
			resolved := resolver.Resolve(modelCtx, PricingInput{Model: billingModel, GroupID: &g.ID, Group: g})
			model.Pricing = catalogResolvedPricing(resolved)
			if sched, err := billing.ResolveContextPricingSchedule(modelCtx, resolver, ContextPricingScheduleInput{Model: billingModel, Group: g, Platform: model.Platform}); err == nil && sched != nil && len(sched.Tiers) > 0 {
				model.Pricing = plazaPricingFromSchedule(model.Pricing, sched)
			}
			model.Pricing = withDefaultMaxReasoningEffortMultiplier(model.Pricing, billingModel)
			if resolved.Mode == BillingModeToken && pricingNeedsFallback(resolved.channelPricing) {
				if _, err := billing.GetModelPricing(billingModel); err != nil {
					model.Pricing = nil // An empty price card is not an explicit free model.
				}
			}
			if (isOpenAIImageGenerationModel(billingModel) || isImageGenerationModel(billingModel)) &&
				(g.ImagePrice1K != nil || g.ImagePrice2K != nil || g.ImagePrice4K != nil) && resolved.Source != PricingSourceGroup {
				model.Pricing = plazaImageDisplayPricing(&ChannelModelPricing{BillingMode: BillingModeImage}, g)
			} else if resolved.Mode == BillingModeImage && resolved.Source != PricingSourceGroup {
				model.Pricing = plazaImageDisplayPricing(model.Pricing, g)
			}
			if mapping.BillingModelSource == BillingModelSourceUpstream || mapping.BillingModelSource == BillingModelSourceResponse {
				// The actual billed model is selected by the account/response; do
				// not quote the requested model's price as a guaranteed charge.
				model.Pricing = nil
			}
			view.SupportedModels = append(view.SupportedModels, model)
		}
		sort.Slice(view.SupportedModels, func(i, j int) bool { return view.SupportedModels[i].Name < view.SupportedModels[j].Name })
		if len(view.SupportedModels) > 0 {
			out = append(out, view)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func catalogResolvedPricing(resolved *ResolvedPricing) *ChannelModelPricing {
	p := &ChannelModelPricing{BillingMode: resolved.Mode}
	if resolved.channelPricing != nil {
		cp := resolved.channelPricing.Clone()
		p = &cp
	}
	if resolved.Mode != BillingModeToken {
		return p
	}
	b := resolved.BasePricing
	if b == nil {
		return nil // Unknown is not free. The UI must retain an unpriced row.
	}
	p.InputPrice, p.OutputPrice = &b.InputPricePerToken, &b.OutputPricePerToken
	p.CacheWritePrice, p.CacheReadPrice = &b.CacheCreationPricePerToken, &b.CacheReadPricePerToken
	p.CacheWrite1hPrice = nonZeroPtr(b.CacheCreation1hPrice)
	p.ImageInputPrice, p.ImageOutputPrice = nonZeroPtr(b.ImageInputPricePerToken), nonZeroPtr(b.ImageOutputPricePerToken)
	p.FastMultiplier, p.FlexMultiplier = b.FastMultiplier, b.FlexMultiplier
	p.MaxReasoningEffortMultiplier = b.MaxReasoningEffortMultiplier
	return p
}
