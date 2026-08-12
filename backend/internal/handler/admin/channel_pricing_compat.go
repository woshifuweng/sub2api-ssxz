package admin

import (
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var platformToLiteLLMProvider = map[string]string{
	service.PlatformAnthropic:   "anthropic",
	service.PlatformOpenAI:      "openai",
	service.PlatformGemini:      "google",
	service.PlatformAntigravity: "anthropic",
	service.PlatformGrok:        "xai",
}

// SyncPricingModels 返回 LiteLLM 定价目录中指定平台的最新模型列表
// GET /api/v1/admin/channels/pricing/sync-models?platform=anthropic
func (h *ChannelHandler) SyncPricingModels(c *gin.Context) {
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	if platform == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MISSING_PARAMETER", "platform parameter is required").
			WithMetadata(map[string]string{"param": "platform"}))
		return
	}

	provider, ok := platformToLiteLLMProvider[platform]
	if !ok {
		response.ErrorFrom(c, infraerrors.BadRequest("UNSUPPORTED_PLATFORM",
			fmt.Sprintf("unsupported platform: %s", platform)).
			WithMetadata(map[string]string{"param": "platform"}))
		return
	}

	models := h.pricingService.ListModelNamesByProvider(provider)
	response.Success(c, gin.H{"models": models})
}
