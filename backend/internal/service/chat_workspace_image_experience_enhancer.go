package service

import "strings"

const (
	WorkspaceImageExperienceEnhancerVersion = "workspace_image_experience_enhancer_v1"

	workspaceImageExperienceReasonNotImageGeneration = "planned_capability_not_image_generation"
	workspaceImageExperienceReasonCommercialProduct  = "commercial_product_image"
	workspaceImageExperienceReasonSocialCover        = "social_media_cover"
	workspaceImageExperienceReasonPoster             = "poster_or_advertisement"
	workspaceImageExperienceReasonBanner             = "banner_image"
	workspaceImageExperienceReasonAvatar             = "avatar_or_portrait"
	workspaceImageExperienceReasonGeneric            = "generic_image_generation"
)

type WorkspaceImageExperienceEnhancerInput struct {
	Text                    string
	PlannedCapability       WorkspacePlannedCapability
	SelectedModel           string
	ModelCapabilityMetadata WorkspaceModelCapabilityMetadata
	AttachmentCount         int
	ContextMessageCount     int
}

type WorkspaceImageExperiencePlan struct {
	Present                 bool
	OriginalPromptPresent   bool
	EnhancedPrompt          string
	EnhancedPromptPresent   bool
	NegativePrompt          string
	NegativePromptPresent   bool
	SubjectHint             string
	SceneHint               string
	StyleHint               string
	AspectRatio             string
	QualityPreset           string
	SafetyHints             []string
	EnhancerVersion         string
	Confidence              float64
	Reason                  string
	SelectedModel           string
	ModelCapabilityMatched  bool
	ModelCapabilityMismatch string
}

type WorkspaceImageExperienceEnhancer struct{}

func NewWorkspaceImageExperienceEnhancer() WorkspaceImageExperienceEnhancer {
	return WorkspaceImageExperienceEnhancer{}
}

func BuildWorkspaceImageExperiencePlan(input WorkspaceImageExperienceEnhancerInput) WorkspaceImageExperiencePlan {
	return NewWorkspaceImageExperienceEnhancer().Build(input)
}

func (WorkspaceImageExperienceEnhancer) Build(input WorkspaceImageExperienceEnhancerInput) WorkspaceImageExperiencePlan {
	selectedModel := strings.TrimSpace(input.SelectedModel)
	plan := WorkspaceImageExperiencePlan{
		Present:                false,
		EnhancerVersion:        WorkspaceImageExperienceEnhancerVersion,
		Reason:                 workspaceImageExperienceReasonNotImageGeneration,
		SelectedModel:          selectedModel,
		ModelCapabilityMatched: workspaceModelCapabilityListContains(input.ModelCapabilityMetadata.Capabilities, WorkspaceModelCapabilityImageGeneration),
	}
	if input.PlannedCapability != WorkspacePlannedCapabilityImageGeneration {
		return plan
	}

	text := strings.TrimSpace(input.Text)
	plan.Present = true
	plan.OriginalPromptPresent = text != ""
	plan.EnhancedPromptPresent = true
	plan.NegativePromptPresent = true
	plan.QualityPreset = "commercial"
	plan.AspectRatio = "1:1"
	plan.SubjectHint = "image subject"
	plan.SceneHint = "commercial image"
	plan.StyleHint = "premium commercial visual"
	plan.Confidence = 0.74
	plan.Reason = workspaceImageExperienceReasonGeneric
	plan.SafetyHints = []string{
		"provider_agnostic_plan",
		"no_image_execution",
		"no_secret_metadata",
	}
	if !plan.ModelCapabilityMatched {
		plan.ModelCapabilityMismatch = "selected_model_does_not_support_image_generation"
	}

	classifyWorkspaceImageExperience(text, &plan)
	plan.NegativePrompt = buildWorkspaceImageNegativePrompt()
	plan.EnhancedPrompt = buildWorkspaceImageEnhancedPrompt(plan)
	return plan
}

func mergeWorkspaceImageExperiencePlanMetadata(metadata map[string]any, plan WorkspaceImageExperiencePlan) map[string]any {
	if !plan.Present {
		return metadata
	}
	out := make(map[string]any, len(metadata)+12)
	for key, value := range metadata {
		out[key] = value
	}
	out["image_experience_plan_present"] = true
	out["image_experience_enhancer_version"] = plan.EnhancerVersion
	out["image_subject_hint"] = plan.SubjectHint
	out["image_scene_hint"] = plan.SceneHint
	out["image_style_hint"] = plan.StyleHint
	out["image_aspect_ratio"] = plan.AspectRatio
	out["image_quality_preset"] = plan.QualityPreset
	out["image_experience_confidence"] = plan.Confidence
	out["image_experience_reason"] = plan.Reason
	out["enhanced_prompt_present"] = plan.EnhancedPromptPresent
	out["negative_prompt_present"] = plan.NegativePromptPresent
	out["original_prompt_present"] = plan.OriginalPromptPresent
	return out
}

func classifyWorkspaceImageExperience(text string, plan *WorkspaceImageExperiencePlan) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch {
	case workspaceTextContainsAny(normalized, []string{"banner", "横版", "横幅"}):
		plan.SubjectHint = "banner image"
		plan.SceneHint = "horizontal banner"
		plan.StyleHint = "clean premium campaign banner"
		plan.AspectRatio = "16:9"
		plan.Confidence = 0.86
		plan.Reason = workspaceImageExperienceReasonBanner
	case workspaceTextContainsAny(normalized, []string{"小红书", "封面", "social media cover"}):
		plan.SubjectHint = "social media cover"
		plan.SceneHint = "vertical social media cover"
		plan.StyleHint = "clean premium social cover"
		plan.AspectRatio = "4:5"
		plan.Confidence = 0.88
		plan.Reason = workspaceImageExperienceReasonSocialCover
	case workspaceTextContainsAny(normalized, []string{"海报", "poster", "advertisement", " ad"}):
		plan.SubjectHint = inferWorkspaceImageSubjectHint(normalized)
		plan.SceneHint = "commercial advertisement"
		plan.StyleHint = "premium marketing poster"
		plan.AspectRatio = "4:5"
		plan.Confidence = 0.86
		plan.Reason = workspaceImageExperienceReasonPoster
	case workspaceTextContainsAny(normalized, []string{"产品", "商品", "电商", "主图", "product", "perfume"}):
		plan.SubjectHint = inferWorkspaceImageSubjectHint(normalized)
		plan.SceneHint = "commercial product image"
		plan.StyleHint = "commercial premium product photography"
		plan.AspectRatio = "1:1"
		plan.Confidence = 0.9
		plan.Reason = workspaceImageExperienceReasonCommercialProduct
	case workspaceTextContainsAny(normalized, []string{"头像", "avatar", "portrait"}):
		plan.SubjectHint = "portrait or avatar"
		plan.SceneHint = "clean portrait composition"
		plan.StyleHint = "polished avatar style"
		plan.AspectRatio = "1:1"
		plan.Confidence = 0.82
		plan.Reason = workspaceImageExperienceReasonAvatar
	}
}

func inferWorkspaceImageSubjectHint(text string) string {
	switch {
	case strings.Contains(text, "香水") || strings.Contains(text, "perfume"):
		return "perfume product"
	case strings.Contains(text, "咖啡") || strings.Contains(text, "coffee"):
		return "coffee shop promotion"
	case strings.Contains(text, "电商") || strings.Contains(text, "商品") || strings.Contains(text, "产品") || strings.Contains(text, "product"):
		return "product"
	default:
		return "image subject"
	}
}

func buildWorkspaceImageEnhancedPrompt(plan WorkspaceImageExperiencePlan) string {
	parts := []string{
		"Create a provider-agnostic image generation plan for " + plan.SubjectHint + ".",
		"Use " + plan.SceneHint + " with " + plan.StyleHint + ".",
		"Keep the subject clear, composition clean, background refined, lighting natural, and visual quality suitable for commercial use.",
		"Avoid cheap, cluttered, oversaturated, plastic-looking, low-quality AI artifacts.",
		"Preferred aspect ratio: " + plan.AspectRatio + ".",
	}
	return strings.Join(parts, " ")
}

func buildWorkspaceImageNegativePrompt() string {
	return "cheap, cluttered, oversaturated, low-quality AI look, distorted subject, messy background, plastic texture, incorrect text, watermark"
}

func workspaceTextContainsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
