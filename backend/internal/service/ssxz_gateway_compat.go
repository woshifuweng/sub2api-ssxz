package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// gatewayFallbackGroupEntry carries the resolved group chain used by the SSXZ
// cross-group fallback policy.
type gatewayFallbackGroupEntry struct {
	Group   *Group
	GroupID int64
}

func (s *GatewayService) buildGatewayFallbackChain(ctx context.Context, group *Group, groupID *int64) ([]gatewayFallbackGroupEntry, error) {
	if group == nil || groupID == nil || *groupID <= 0 {
		return nil, nil
	}
	out := []gatewayFallbackGroupEntry{{Group: group, GroupID: *groupID}}
	visited := map[int64]struct{}{*groupID: {}}
	current := group
	for current != nil && current.FallbackGroupID != nil && *current.FallbackGroupID > 0 {
		nextGroup, nextID, err := s.resolveGatewayGroup(ctx, current.FallbackGroupID)
		if err != nil {
			return nil, err
		}
		if nextGroup == nil || nextID == nil || *nextID <= 0 {
			break
		}
		if _, seen := visited[*nextID]; seen {
			return nil, fmt.Errorf("fallback group cycle detected")
		}
		visited[*nextID] = struct{}{}
		out = append(out, gatewayFallbackGroupEntry{Group: nextGroup, GroupID: *nextID})
		current = nextGroup
	}
	return out, nil
}

func (s *GatewayService) listSoraSchedulableAccounts(ctx context.Context, groupID *int64) ([]Account, bool, error) {
	const useMixed = false
	var accounts []Account
	var err error
	if s.cfg != nil && s.cfg.RunMode == "simple" {
		accounts, err = s.accountRepo.ListByPlatform(ctx, PlatformSora)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListByGroup(ctx, *groupID)
	} else {
		accounts, err = s.accountRepo.ListByPlatform(ctx, PlatformSora)
	}
	if err != nil {
		return nil, useMixed, err
	}
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == PlatformSora && s.isSoraAccountSchedulable(&account) {
			filtered = append(filtered, account)
		}
	}
	slog.Debug("account_scheduling_list_sora", "group_id", derefGroupID(groupID), "raw_count", len(accounts), "filtered_count", len(filtered))
	return filtered, useMixed, nil
}

func (s *GatewayService) isSoraAccountSchedulable(account *Account) bool {
	return s.soraUnschedulableReason(account) == ""
}

func (s *GatewayService) soraUnschedulableReason(account *Account) string {
	if account == nil {
		return "account_nil"
	}
	if account.Status != StatusActive {
		return fmt.Sprintf("status=%s", account.Status)
	}
	if !account.Schedulable {
		return "schedulable=false"
	}
	if account.TempUnschedulableUntil != nil && time.Now().Before(*account.TempUnschedulableUntil) {
		return fmt.Sprintf("temp_unschedulable_until=%s", account.TempUnschedulableUntil.UTC().Format(time.RFC3339))
	}
	return ""
}

func (s *GatewayService) logSoraSelectionFailureDetails(ctx context.Context, groupID *int64, sessionHash, requestedModel string, accounts []Account, excludedIDs map[int64]struct{}, allowMixedScheduling bool) {
	const maxLines = 30
	logged := 0
	for i := range accounts {
		if logged >= maxLines {
			break
		}
		account := &accounts[i]
		diagnosis := s.diagnoseSelectionFailure(ctx, account, requestedModel, PlatformSora, excludedIDs, allowMixedScheduling)
		if diagnosis.Category == "eligible" {
			continue
		}
		detail := diagnosis.Detail
		if detail == "" {
			detail = "-"
		}
		logger.LegacyPrintf("service.gateway", "[SelectAccountDetailed:Sora] group_id=%v model=%s session=%s account_id=%d account_platform=%s category=%s detail=%s", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), account.ID, account.Platform, diagnosis.Category, detail)
		logged++
	}
	if len(accounts) > maxLines {
		logger.LegacyPrintf("service.gateway", "[SelectAccountDetailed:Sora] group_id=%v model=%s session=%s truncated=true total=%d logged=%d", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), len(accounts), logged)
	}
}

func (s *GatewayService) isSoraModelSupportedByAccount(account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(requestedModel) == "" {
		return true
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 || account.IsModelSupported(requestedModel) {
		return true
	}
	aliases := buildSoraModelAliases(requestedModel)
	if len(aliases) == 0 {
		return false
	}
	hasSoraSelector := false
	for pattern := range mapping {
		if !isSoraModelSelector(pattern) {
			continue
		}
		hasSoraSelector = true
		if matchPatternAnyAlias(pattern, aliases) {
			return true
		}
	}
	return !hasSoraSelector
}

func matchPatternAnyAlias(pattern string, aliases []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(pattern))
	for _, alias := range aliases {
		if normalized != "" && matchWildcard(normalized, alias) {
			return true
		}
	}
	return false
}

func isSoraModelSelector(pattern string) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	return strings.HasPrefix(p, "sora") || strings.HasPrefix(p, "gpt-image") || strings.HasPrefix(p, "prompt-enhance") || strings.HasPrefix(p, "sy_") || p == "video" || p == "image"
}

func buildSoraModelAliases(requestedModel string) []string {
	modelID := strings.ToLower(strings.TrimSpace(requestedModel))
	if modelID == "" {
		return nil
	}
	aliases := make([]string, 0, 8)
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return
		}
		for _, existing := range aliases {
			if existing == value {
				return
			}
		}
		aliases = append(aliases, value)
	}
	add(modelID)
	if cfg, ok := GetSoraModelConfig(modelID); ok {
		add(cfg.Model)
		switch cfg.Type {
		case "video":
			add("video")
			add("sora")
			add(soraVideoFamilyAlias(modelID))
		case "image":
			add("image")
			add("gpt-image")
		case "prompt_enhance":
			add("prompt-enhance")
		}
		return aliases
	}
	switch {
	case strings.HasPrefix(modelID, "sora"):
		add("video")
		add("sora")
		add(soraVideoFamilyAlias(modelID))
	case strings.HasPrefix(modelID, "gpt-image"):
		add("image")
		add("gpt-image")
	case strings.HasPrefix(modelID, "prompt-enhance"):
		add("prompt-enhance")
	default:
		return nil
	}
	return aliases
}

func soraVideoFamilyAlias(modelID string) string {
	switch {
	case strings.HasPrefix(modelID, "sora2pro-hd"):
		return "sora2pro-hd"
	case strings.HasPrefix(modelID, "sora2pro"):
		return "sora2pro"
	case strings.HasPrefix(modelID, "sora2"):
		return "sora2"
	default:
		return ""
	}
}

func usageBillingParamsRequireIdempotentRepository(p *postUsageBillingParams) bool {
	if p == nil || p.Cost == nil {
		return false
	}
	if p.IsSubscriptionBill && p.Cost.TotalCost > 0 {
		return true
	}
	if !p.IsSubscriptionBill && p.Cost.ActualCost > 0 {
		return true
	}
	if p.Cost.ActualCost > 0 && p.APIKey != nil && p.APIKeyService != nil && (p.APIKey.Quota > 0 || p.APIKey.HasRateLimits()) {
		return true
	}
	return p.Cost.TotalCost > 0 && p.Account != nil && p.Account.IsAPIKeyOrBedrock() && p.Account.HasAnyQuotaLimit()
}

func usageBillingCommandRequiresRepository(cmd *UsageBillingCommand) bool {
	if cmd == nil {
		return false
	}
	return cmd.SubscriptionCost > 0 || cmd.BalanceCost > 0 || cmd.APIKeyQuotaCost > 0 || cmd.APIKeyRateLimitCost > 0 || cmd.AccountQuotaCost > 0
}
