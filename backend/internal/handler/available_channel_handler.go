package handler

import (
	"net/http"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AvailableChannelHandler 处理用户侧「可用渠道」查询。
//
// 用户侧接口委托 ChannelService.ListAvailable，并在返回前做三层过滤：
//  1. 行过滤：只保留状态为 Active 且与当前用户可访问分组有交集的渠道；
//  2. 分组过滤：渠道的 Groups 只保留用户可访问的那些；
//  3. 平台过滤：渠道的 SupportedModels 只保留平台在用户可见 Groups 中出现过的模型，
//     防止"渠道同时挂在 antigravity / anthropic 两个平台的分组上，用户只访问
//     antigravity，却看到 anthropic 模型"这类跨平台信息泄漏；
//  4. 字段白名单：仅返回用户需要的字段（省略 BillingModelSource / RestrictModels
//     / 内部 ID / Status 等管理字段）。
type AvailableChannelHandler struct {
	channelService *service.ChannelService
	apiKeyService  *service.APIKeyService
	settingService *service.SettingService
}

// NewAvailableChannelHandler 创建用户侧可用渠道 handler。
func NewAvailableChannelHandler(
	channelService *service.ChannelService,
	apiKeyService *service.APIKeyService,
	settingService *service.SettingService,
) *AvailableChannelHandler {
	return &AvailableChannelHandler{
		channelService: channelService,
		apiKeyService:  apiKeyService,
		settingService: settingService,
	}
}

// featureEnabled 返回 available-channels 开关是否启用。默认关闭（opt-in）。
func (h *AvailableChannelHandler) featureEnabled(c *gin.Context) bool {
	return h.featureEnabledGateway(gatewayctx.FromGin(c))
}

func (h *AvailableChannelHandler) featureEnabledGateway(c gatewayctx.GatewayContext) bool {
	if h.settingService == nil {
		return false
	}
	return h.settingService.GetAvailableChannelsRuntime(c.Request().Context()).Enabled
}

// userAvailableGroup 用户可见的分组概要（白名单字段）。
//
// 前端据此区分专属 vs 公开分组（IsExclusive）、订阅 vs 标准分组（SubscriptionType，
// 订阅视觉加深），并用 RateMultiplier 作为默认倍率；用户专属倍率前端走
// /groups/rates，和 API 密钥页面保持一致。
type userAvailableGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	SubscriptionType string  `json:"subscription_type"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	IsExclusive      bool    `json:"is_exclusive"`
}

// userSupportedModelPricing 用户可见的定价字段白名单。
type userSupportedModelPricing struct {
	BillingMode      string                   `json:"billing_mode"`
	InputPrice       *float64                 `json:"input_price"`
	OutputPrice      *float64                 `json:"output_price"`
	CacheWritePrice  *float64                 `json:"cache_write_price"`
	CacheReadPrice   *float64                 `json:"cache_read_price"`
	ImageOutputPrice *float64                 `json:"image_output_price"`
	PerRequestPrice  *float64                 `json:"per_request_price"`
	Intervals        []userPricingIntervalDTO `json:"intervals"`
}

// userPricingIntervalDTO 定价区间白名单（去掉内部 ID、SortOrder 等前端不渲染的字段）。
type userPricingIntervalDTO struct {
	MinTokens       int      `json:"min_tokens"`
	MaxTokens       *int     `json:"max_tokens"`
	TierLabel       string   `json:"tier_label,omitempty"`
	InputPrice      *float64 `json:"input_price"`
	OutputPrice     *float64 `json:"output_price"`
	CacheWritePrice *float64 `json:"cache_write_price"`
	CacheReadPrice  *float64 `json:"cache_read_price"`
	PerRequestPrice *float64 `json:"per_request_price"`
}

// userSupportedModel 用户可见的支持模型条目。
type userSupportedModel struct {
	Name               string                     `json:"name"`
	Platform           string                     `json:"platform"`
	Pricing            *userSupportedModelPricing `json:"pricing"`
	PricingStatus      string                     `json:"pricing_status,omitempty"`
	UsageSupport       []string                   `json:"usage_support,omitempty"`
	Capabilities       []string                   `json:"capabilities,omitempty"`
	ProviderLabel      string                     `json:"provider_label,omitempty"`
	Provider           string                     `json:"provider,omitempty"`
	CapabilitySource   string                     `json:"capability_source,omitempty"`
	ModelCatalogSource string                     `json:"model_catalog_source,omitempty"`
	Fake               bool                       `json:"fake,omitempty"`
	TestOnly           bool                       `json:"test_only,omitempty"`
	StagingOnly        bool                       `json:"staging_only,omitempty"`
}

// userChannelPlatformSection 单渠道内某个平台的子视图：用户可见的分组 + 该平台
// 支持的模型。按 platform 聚合后让前端可以把渠道名作为 row-group 一次渲染，
// 后面的平台行按 sections 顺序铺开。
type userChannelPlatformSection struct {
	Platform        string               `json:"platform"`
	Groups          []userAvailableGroup `json:"groups"`
	SupportedModels []userSupportedModel `json:"supported_models"`
}

// userAvailableChannel 用户可见的渠道条目（白名单字段）。
//
// 每个渠道聚合为一条记录，内嵌 platforms 子数组：每个 section 对应一个平台，
// 包含该平台的 groups 和 supported_models。
type userAvailableChannel struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Platforms   []userChannelPlatformSection `json:"platforms"`
}

// List 列出当前用户可见的「可用渠道」。
// GET /api/v1/channels/available
func (h *AvailableChannelHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *AvailableChannelHandler) ListGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Feature 未启用时返回空数组（不暴露渠道信息）。检查放在认证之后，
	// 保持与未开关前的 401 行为一致：未登录先 401，登录后再按开关决定。
	if !h.featureEnabledGateway(c) {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, []userAvailableChannel{})
		return
	}

	userGroups, err := h.apiKeyService.GetAvailableGroups(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	allowedGroupIDs := make(map[int64]struct{}, len(userGroups))
	for i := range userGroups {
		allowedGroupIDs[userGroups[i].ID] = struct{}{}
	}

	channels, err := h.channelService.ListAvailable(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]userAvailableChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.Status != service.StatusActive {
			continue
		}
		visibleGroups := filterUserVisibleGroups(ch.Groups, allowedGroupIDs)
		if len(visibleGroups) == 0 {
			continue
		}
		sections := buildPlatformSections(ch, visibleGroups, h.settingService, subject.UserID)
		if len(sections) == 0 {
			continue
		}
		out = append(out, userAvailableChannel{
			Name:        ch.Name,
			Description: ch.Description,
			Platforms:   sections,
		})
	}
	if fakeModel := h.settingService.GetWorkspaceImageFakeModelExposure(subject.UserID); fakeModel.Enabled {
		out = appendWorkspaceImageFakeModelChannel(out, fakeModel)
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, out)
}

func appendWorkspaceImageFakeModelChannel(
	channels []userAvailableChannel,
	fakeModel service.WorkspaceImageFakeModelExposure,
) []userAvailableChannel {
	if !fakeModel.Enabled {
		return channels
	}
	platform := fakeModel.Platform
	if platform == "" {
		platform = fakeModel.ProviderLabel
	}
	if platform == "" {
		platform = service.WorkspaceImageProviderFakeLabel
	}
	return append(channels, userAvailableChannel{
		Name:        "Workspace Image Fake",
		Description: "Staging-only fake image generation model for workspace validation.",
		Platforms: []userChannelPlatformSection{{
			Platform: platform,
			Groups: []userAvailableGroup{{
				ID:               0,
				Name:             "Workspace Image Fake",
				Platform:         platform,
				SubscriptionType: "test",
				RateMultiplier:   1,
				IsExclusive:      true,
			}},
			SupportedModels: []userSupportedModel{{
				Name:               fakeModel.Model,
				Platform:           platform,
				Pricing:            nil,
				Capabilities:       workspaceCapabilityStringsForUserDTO(fakeModel.Capabilities),
				ProviderLabel:      fakeModel.ProviderLabel,
				Provider:           fakeModel.ProviderLabel,
				CapabilitySource:   fakeModel.CapabilitySource,
				ModelCatalogSource: fakeModel.ModelCatalogSource,
				Fake:               fakeModel.Fake,
				TestOnly:           fakeModel.TestOnly,
			}},
		}},
	})
}

func workspaceCapabilityStringsForUserDTO(capabilities []service.WorkspaceModelCapability) []string {
	if len(capabilities) == 0 {
		return nil
	}
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability == "" {
			continue
		}
		out = append(out, string(capability))
	}
	return out
}

// buildPlatformSections 把一个渠道按 visibleGroups 的平台集合拆成有序的 section 列表：
// 每个 section 对应一个平台，只包含该平台的 groups 和 supported_models。
// 输出按 platform 字母序稳定排序，便于前端等效比较与回归测试。
func buildPlatformSections(
	ch service.AvailableChannel,
	visibleGroups []userAvailableGroup,
	settingService *service.SettingService,
	userID int64,
) []userChannelPlatformSection {
	groupsByPlatform := make(map[string][]userAvailableGroup, 4)
	for _, g := range visibleGroups {
		if g.Platform == "" {
			continue
		}
		groupsByPlatform[g.Platform] = append(groupsByPlatform[g.Platform], g)
	}
	if len(groupsByPlatform) == 0 {
		return nil
	}

	platforms := make([]string, 0, len(groupsByPlatform))
	for p := range groupsByPlatform {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)

	sections := make([]userChannelPlatformSection, 0, len(platforms))
	for _, platform := range platforms {
		platformSet := map[string]struct{}{platform: {}}
		sections = append(sections, userChannelPlatformSection{
			Platform:        platform,
			Groups:          groupsByPlatform[platform],
			SupportedModels: toUserSupportedModels(ch.SupportedModels, platformSet, settingService, userID),
		})
	}
	return sections
}

// filterUserVisibleGroups 仅保留用户可访问的分组。
func filterUserVisibleGroups(
	groups []service.AvailableGroupRef,
	allowed map[int64]struct{},
) []userAvailableGroup {
	visible := make([]userAvailableGroup, 0, len(groups))
	for _, g := range groups {
		if _, ok := allowed[g.ID]; !ok {
			continue
		}
		visible = append(visible, userAvailableGroup{
			ID:               g.ID,
			Name:             g.Name,
			Platform:         g.Platform,
			SubscriptionType: g.SubscriptionType,
			RateMultiplier:   g.RateMultiplier,
			IsExclusive:      g.IsExclusive,
		})
	}
	return visible
}

// toUserSupportedModels 将 service 层支持模型转换为用户 DTO（字段白名单）。
// 仅保留平台在 allowedPlatforms 中的条目，防止跨平台模型信息泄漏。
// allowedPlatforms 为 nil 时不做平台过滤（保留全部，供测试或明确无过滤场景使用）。
func toUserSupportedModels(
	src []service.SupportedModel,
	allowedPlatforms map[string]struct{},
	settingService *service.SettingService,
	userID int64,
) []userSupportedModel {
	out := make([]userSupportedModel, 0, len(src))
	for i := range src {
		m := src[i]
		if allowedPlatforms != nil {
			if _, ok := allowedPlatforms[m.Platform]; !ok {
				continue
			}
		}
		model := userSupportedModel{
			Name:          m.Name,
			Platform:      m.Platform,
			Pricing:       toUserPricing(m.Pricing),
			PricingStatus: userModelPricingStatus(m.Pricing),
			UsageSupport:  userModelUsageSupport(m.Pricing),
		}
		metadata := service.ResolveWorkspaceModelCapabilities(m.Name, service.WorkspaceModelCapabilityHints{
			Platform: m.Platform,
		})
		if !workspaceUserDTOCapabilitiesContain(metadata.Capabilities, service.WorkspaceModelCapabilityImageGeneration) {
			model.Capabilities = workspaceCapabilityStringsForUserDTO(metadata.Capabilities)
			model.Provider = m.Platform
			model.CapabilitySource = metadata.CapabilitySource
			model.ModelCatalogSource = service.WorkspaceModelCatalogSourceRealChannel
		}
		if settingService != nil {
			realMetadata := settingService.GetWorkspaceImageRealChannelModelExposure(userID, m)
			if realMetadata.Model != "" {
				model.Capabilities = workspaceCapabilityStringsForUserDTO(realMetadata.Capabilities)
				model.ProviderLabel = realMetadata.ProviderLabel
				model.Provider = realMetadata.Provider
				model.CapabilitySource = realMetadata.CapabilitySource
				model.ModelCatalogSource = realMetadata.ModelCatalogSource
				model.StagingOnly = realMetadata.StagingOnly
			}
		}
		out = append(out, model)
	}
	return out
}

func workspaceUserDTOCapabilitiesContain(capabilities []service.WorkspaceModelCapability, target service.WorkspaceModelCapability) bool {
	for _, capability := range capabilities {
		if capability == target {
			return true
		}
	}
	return false
}

func userModelPricingStatus(p *service.ChannelModelPricing) string {
	if p == nil {
		return service.WorkspaceSelectedModelPricingMissing
	}
	return service.WorkspaceSelectedModelPricingConfigured
}

func userModelUsageSupport(p *service.ChannelModelPricing) []string {
	if p == nil {
		return nil
	}
	switch p.BillingMode {
	case service.BillingModeImage:
		out := []string{"image_count"}
		if len(p.Intervals) > 0 {
			out = append(out, "image_size")
		}
		return out
	case service.BillingModePerRequest:
		return []string{"request"}
	default:
		return []string{"token"}
	}
}

// toUserPricing 将 service 层定价转换为用户 DTO；入参为 nil 时返回 nil。
func toUserPricing(p *service.ChannelModelPricing) *userSupportedModelPricing {
	if p == nil {
		return nil
	}
	intervals := make([]userPricingIntervalDTO, 0, len(p.Intervals))
	for _, iv := range p.Intervals {
		intervals = append(intervals, userPricingIntervalDTO{
			MinTokens:       iv.MinTokens,
			MaxTokens:       iv.MaxTokens,
			TierLabel:       iv.TierLabel,
			InputPrice:      iv.InputPrice,
			OutputPrice:     iv.OutputPrice,
			CacheWritePrice: iv.CacheWritePrice,
			CacheReadPrice:  iv.CacheReadPrice,
			PerRequestPrice: iv.PerRequestPrice,
		})
	}
	billingMode := string(p.BillingMode)
	if billingMode == "" {
		billingMode = string(service.BillingModeToken)
	}
	return &userSupportedModelPricing{
		BillingMode:      billingMode,
		InputPrice:       p.InputPrice,
		OutputPrice:      p.OutputPrice,
		CacheWritePrice:  p.CacheWritePrice,
		CacheReadPrice:   p.CacheReadPrice,
		ImageOutputPrice: p.ImageOutputPrice,
		PerRequestPrice:  p.PerRequestPrice,
		Intervals:        intervals,
	}
}
