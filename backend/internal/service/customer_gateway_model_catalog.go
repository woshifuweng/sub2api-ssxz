package service

import (
	"sort"
	"strings"
)

var customerGatewayModelsByPlatform = map[string][]string{
	PlatformOpenAI: {
		"gpt-5.6-sol",
		"gpt-5.6-terra",
		"gpt-5.6-luna",
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
	},
	PlatformAnthropic: {
		"claude-fable-5",
		"claude-opus-4-6",
		"claude-opus-4-7",
		"claude-opus-4-8",
		"claude-sonnet-4-6",
		"claude-sonnet-5",
	},
}

func IsCustomerGatewayPlatform(platform string) bool {
	_, ok := customerGatewayModelsByPlatform[strings.ToLower(strings.TrimSpace(platform))]
	return ok
}

func CustomerGatewayModelIDs(platform string) []string {
	models := customerGatewayModelsByPlatform[strings.ToLower(strings.TrimSpace(platform))]
	return cloneStringSlice(models)
}

func CanonicalCustomerGatewayModel(platform, model string) (string, bool) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimPrefix(model, "models/")
	if model == "" {
		return "", false
	}

	if platform == PlatformOpenAI {
		model = normalizeCodexModel(model)
	}
	for _, allowed := range customerGatewayModelsByPlatform[platform] {
		if model == allowed {
			return allowed, true
		}
	}
	return "", false
}

func IsBlockedCustomerGatewayModel(platform, model string) bool {
	platform = strings.ToLower(strings.TrimSpace(platform))
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimPrefix(model, "models/")
	switch platform {
	case PlatformOpenAI:
		return strings.HasPrefix(model, "gpt-5.3-codex-spark")
	case PlatformAnthropic:
		return model == "haiku" || strings.Contains(model, "haiku")
	default:
		return false
	}
}

func filterCustomerGatewayModels(platform string, models []string) []string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if !IsCustomerGatewayPlatform(platform) {
		return cloneStringSlice(models)
	}

	allowed := make(map[string]struct{}, len(customerGatewayModelsByPlatform[platform]))
	for _, model := range customerGatewayModelsByPlatform[platform] {
		allowed[model] = struct{}{}
	}

	filteredSet := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.ToLower(strings.TrimSpace(model))
		if _, ok := allowed[model]; !ok {
			continue
		}
		if getOfficialExactModelPricing(model) == nil {
			continue
		}
		filteredSet[model] = struct{}{}
	}

	filtered := make([]string, 0, len(filteredSet))
	for model := range filteredSet {
		filtered = append(filtered, model)
	}
	sort.Strings(filtered)
	return filtered
}
