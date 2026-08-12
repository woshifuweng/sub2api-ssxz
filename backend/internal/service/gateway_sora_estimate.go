package service

import (
	"context"
	"encoding/json"
	"strings"
)

// EstimateSoraRequestCost estimates the charge before a Sora client request is
// accepted so insufficient balance is rejected before upstream work starts.
func (s *GatewayService) EstimateSoraRequestCost(ctx context.Context, requestedModel string, body []byte, apiKey *APIKey, user *User, account *Account) (*CostBreakdown, error) {
	if s == nil || s.billingService == nil || len(body) == 0 {
		return nil, nil
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, err
	}

	modelID := strings.TrimSpace(requestedModel)
	if modelID == "" {
		modelID, _ = reqBody["model"].(string)
		modelID = strings.TrimSpace(modelID)
	}
	if modelID == "" {
		return nil, nil
	}
	if account != nil {
		modelID = strings.TrimSpace(account.GetMappedModel(modelID))
	}

	modelCfg, ok := GetSoraModelConfig(modelID)
	if !ok {
		return nil, nil
	}
	if modelCfg.Type == "prompt_enhance" {
		return &CostBreakdown{}, nil
	}

	multiplier := 0.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if apiKey != nil && apiKey.GroupID != nil && apiKey.Group != nil && user != nil {
		multiplier = s.getUserGroupRateMultiplier(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}

	var soraConfig *SoraPriceConfig
	if apiKey != nil {
		soraConfig = soraPriceConfigFromGroup(apiKey.Group)
	}

	switch modelCfg.Type {
	case "image":
		return s.billingService.CalculateSoraImageCost(soraImageSizeFromModel(modelID), 1, soraConfig, multiplier), nil
	case "video":
		videoCount := parseSoraVideoCount(reqBody)
		prompt, _, _, remixTargetID := extractSoraInput(reqBody)
		if strings.TrimSpace(remixTargetID) == "" && isSoraStoryboardPrompt(prompt) {
			videoCount = 1
		}
		return s.billingService.CalculateSoraVideoCostForCount(modelID, videoCount, soraConfig, multiplier), nil
	default:
		return nil, nil
	}
}

func soraPriceConfigFromGroup(group *Group) *SoraPriceConfig {
	if group == nil {
		return nil
	}
	return &SoraPriceConfig{
		ImagePrice360:          group.SoraImagePrice360,
		ImagePrice540:          group.SoraImagePrice540,
		VideoPricePerRequest:   group.SoraVideoPricePerRequest,
		VideoPricePerRequestHD: group.SoraVideoPricePerRequestHD,
	}
}
