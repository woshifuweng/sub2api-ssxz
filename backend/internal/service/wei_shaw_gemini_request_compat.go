package service

import (
	"encoding/json"
	"strings"
)

func (s *GeminiMessagesCompatService) extractImageInputSize(body []byte) string {
	var req struct {
		GenerationConfig *struct {
			ImageConfig *struct {
				ImageSize string `json:"imageSize"`
			} `json:"imageConfig"`
		} `json:"generationConfig"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	if req.GenerationConfig != nil && req.GenerationConfig.ImageConfig != nil {
		return strings.TrimSpace(req.GenerationConfig.ImageConfig.ImageSize)
	}
	return ""
}
