package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/cespare/xxhash/v2"
)

var errAPIKeyGroupBindingIncomplete = errors.New("api key group binding is incomplete")

type gatewayAPIKeySelection struct {
	APIKey    *service.APIKey
	Selection *service.AccountSelectionResult
}

type openAIAPIKeySelection struct {
	APIKey    *service.APIKey
	Selection *service.AccountSelectionResult
	Decision  service.OpenAIAccountScheduleDecision
}

type channelMappingResolver interface {
	ResolveChannelMappingAndRestrict(context.Context, *int64, string) (service.ChannelMappingResult, bool)
}

func usageChannelMappingForAPIKey(ctx context.Context, resolver channelMappingResolver, apiKey *service.APIKey, model string) service.ChannelMappingResult {
	if resolver == nil || apiKey == nil {
		return service.ChannelMappingResult{MappedModel: model}
	}
	mapping, _ := resolver.ResolveChannelMappingAndRestrict(ctx, apiKey.GroupIDForUsage(), model)
	return mapping
}

func orderedAPIKeyGroups(apiKey *service.APIKey, seed string) ([]*service.Group, error) {
	if apiKey == nil {
		return nil, errAPIKeyGroupBindingIncomplete
	}
	groupIDs := service.NormalizeAPIKeyGroupIDs(apiKey.GroupID, apiKey.GroupIDs)
	if len(groupIDs) == 0 {
		return nil, nil
	}
	byID := make(map[int64]*service.Group, len(apiKey.Groups)+1)
	for _, group := range apiKey.Groups {
		if group != nil && group.ID > 0 {
			byID[group.ID] = group
		}
	}
	if apiKey.Group != nil && apiKey.Group.ID > 0 {
		byID[apiKey.Group.ID] = apiKey.Group
	}
	groups := make([]*service.Group, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group := byID[groupID]
		if group == nil {
			// Single-group callers only need the persisted GroupID for account
			// selection. Some internal gateway paths and older auth-cache entries do
			// not hydrate the full Group object; keep that established path working.
			// Multi-group routing still fails closed because it needs every concrete
			// group to clone the selected billing context safely.
			if len(groupIDs) == 1 && apiKey.GroupID != nil && *apiKey.GroupID == groupID {
				return nil, nil
			}
			return nil, errAPIKeyGroupBindingIncomplete
		}
		groups = append(groups, group)
	}
	if len(groups) <= 1 {
		return groups, nil
	}
	if strings.TrimSpace(seed) == "" {
		seed = "apikey-multi-group"
	}
	start := int(xxhash.Sum64String(seed) % uint64(len(groups)))
	rotated := make([]*service.Group, 0, len(groups))
	rotated = append(rotated, groups[start:]...)
	rotated = append(rotated, groups[:start]...)
	return rotated, nil
}

func isNoAvailableSelectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrNoAvailableAccounts) || errors.Is(err, service.ErrNoAvailableCompactAccounts) {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no available")
}

func selectGatewayAPIKeyGroup(apiKey *service.APIKey, seed string, selectFn func(*int64) (*service.AccountSelectionResult, error)) (*gatewayAPIKeySelection, error) {
	groups, err := orderedAPIKeyGroups(apiKey, seed)
	if err != nil {
		return nil, err
	}
	if len(groups) <= 1 {
		selection, err := selectFn(apiKey.GroupID)
		if err != nil {
			return nil, err
		}
		return &gatewayAPIKeySelection{APIKey: apiKey, Selection: selection}, nil
	}
	var firstWait *gatewayAPIKeySelection
	var lastErr error
	for _, group := range groups {
		current := cloneAPIKeyWithGroup(apiKey, group)
		selection, err := selectFn(current.GroupID)
		if err == nil {
			candidate := &gatewayAPIKeySelection{APIKey: current, Selection: selection}
			if selection != nil && selection.WaitPlan != nil && !selection.Acquired {
				if firstWait == nil {
					firstWait = candidate
				}
				continue
			}
			return candidate, nil
		}
		if !isNoAvailableSelectionError(err) {
			return nil, err
		}
		lastErr = err
	}
	if firstWait != nil {
		return firstWait, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, service.ErrNoAvailableAccounts
}

func selectOpenAIAPIKeyGroup(apiKey *service.APIKey, seed string, selectFn func(*int64) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error)) (*openAIAPIKeySelection, error) {
	groups, err := orderedAPIKeyGroups(apiKey, seed)
	if err != nil {
		return nil, err
	}
	if len(groups) <= 1 {
		selection, decision, err := selectFn(apiKey.GroupID)
		if err != nil {
			return nil, err
		}
		return &openAIAPIKeySelection{APIKey: apiKey, Selection: selection, Decision: decision}, nil
	}
	var firstWait *openAIAPIKeySelection
	var lastErr error
	for _, group := range groups {
		current := cloneAPIKeyWithGroup(apiKey, group)
		selection, decision, err := selectFn(current.GroupID)
		if err == nil {
			candidate := &openAIAPIKeySelection{APIKey: current, Selection: selection, Decision: decision}
			if selection != nil && selection.WaitPlan != nil && !selection.Acquired {
				if firstWait == nil {
					firstWait = candidate
				}
				continue
			}
			return candidate, nil
		}
		if !isNoAvailableSelectionError(err) {
			return nil, err
		}
		lastErr = err
	}
	if firstWait != nil {
		return firstWait, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, service.ErrNoAvailableAccounts
}

func selectAccountForModelAcrossAPIKeyGroups(apiKey *service.APIKey, seed string, selectFn func(*int64) (*service.Account, error)) (*service.APIKey, *service.Account, error) {
	groups, err := orderedAPIKeyGroups(apiKey, seed)
	if err != nil {
		return nil, nil, err
	}
	if len(groups) <= 1 {
		account, err := selectFn(apiKey.GroupID)
		return apiKey, account, err
	}
	var lastErr error
	for _, group := range groups {
		current := cloneAPIKeyWithGroup(apiKey, group)
		account, err := selectFn(current.GroupID)
		if err == nil {
			return current, account, nil
		}
		if !isNoAvailableSelectionError(err) {
			return nil, nil, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, service.ErrNoAvailableAccounts
}
