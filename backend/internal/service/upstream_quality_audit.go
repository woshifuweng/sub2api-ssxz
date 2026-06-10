package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

const UpstreamQualityAuditContextKey = "upstream_quality_audit"

type UpstreamQualityOperation string

const (
	UpstreamQualityOperationTextCompletion  UpstreamQualityOperation = "text_completion"
	UpstreamQualityOperationImageGeneration UpstreamQualityOperation = "image_generation"
	UpstreamQualityOperationImageEdit       UpstreamQualityOperation = "image_edit"
)

type UpstreamQualityImageParams struct {
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Style        string `json:"style,omitempty"`
	Background   string `json:"background,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Count        int    `json:"count,omitempty"`
}

type UpstreamQualityTextParams struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"top_p,omitempty"`
	MaxTokens       *int     `json:"max_tokens,omitempty"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
	ResponseFormat  string   `json:"response_format,omitempty"`
}

type UpstreamQualityUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
	ImageCount   int `json:"image_count,omitempty"`
}

type UpstreamQualityAuditRecord struct {
	RequestID        string                     `json:"request_id,omitempty"`
	Route            string                     `json:"route,omitempty"`
	Operation        UpstreamQualityOperation   `json:"operation,omitempty"`
	RequestedModel   string                     `json:"requested_model,omitempty"`
	MappedModel      string                     `json:"mapped_model,omitempty"`
	UpstreamModel    string                     `json:"upstream_model,omitempty"`
	ProviderName     string                     `json:"provider_name,omitempty"`
	EndpointLabel    string                     `json:"endpoint_label,omitempty"`
	EndpointHostHash string                     `json:"endpoint_host_hash,omitempty"`
	ServiceTier      string                     `json:"service_tier,omitempty"`
	FallbackUsed     bool                       `json:"fallback_used,omitempty"`
	FallbackReason   string                     `json:"fallback_reason,omitempty"`
	LatencyMs        int64                      `json:"latency_ms,omitempty"`
	Status           string                     `json:"status,omitempty"`
	ErrorCode        string                     `json:"error_code,omitempty"`
	TokenUsage       UpstreamQualityUsage       `json:"token_usage,omitempty"`
	TextParams       UpstreamQualityTextParams  `json:"text_params,omitempty"`
	ImageParams      UpstreamQualityImageParams `json:"image_params,omitempty"`
	PromptHash       string                     `json:"prompt_hash,omitempty"`
	PromptPreview    string                     `json:"prompt_preview_redacted,omitempty"`
	PromptEnhanced   bool                       `json:"prompt_enhancer_used,omitempty"`
	CreatedAt        time.Time                  `json:"created_at,omitempty"`
}

type UpstreamQualityDiagnosticSeverity string

const (
	UpstreamQualityDiagnosticSeverityHigh   UpstreamQualityDiagnosticSeverity = "high"
	UpstreamQualityDiagnosticSeverityMedium UpstreamQualityDiagnosticSeverity = "medium"
	UpstreamQualityDiagnosticSeverityLow    UpstreamQualityDiagnosticSeverity = "low"
)

type UpstreamQualityDiagnosticFinding struct {
	Code           string                            `json:"code"`
	Severity       UpstreamQualityDiagnosticSeverity `json:"severity"`
	Operation      UpstreamQualityOperation          `json:"operation,omitempty"`
	RequestedModel string                            `json:"requested_model,omitempty"`
	UpstreamModel  string                            `json:"upstream_model,omitempty"`
	ProviderName   string                            `json:"provider_name,omitempty"`
	Message        string                            `json:"message"`
	Recommendation string                            `json:"recommendation"`
}

type UpstreamQualityDiagnosticReport struct {
	TotalRecords                int                                `json:"total_records"`
	TextRecords                 int                                `json:"text_records"`
	ImageRecords                int                                `json:"image_records"`
	FallbackRecords             int                                `json:"fallback_records"`
	FailedRecords               int                                `json:"failed_records"`
	MissingUpstreamModelRecords int                                `json:"missing_upstream_model_records"`
	Findings                    []UpstreamQualityDiagnosticFinding `json:"findings"`
}

type UpstreamQualityAuditInput struct {
	RequestID      string
	Route          string
	Operation      UpstreamQualityOperation
	RequestedModel string
	MappedModel    string
	UpstreamModel  string
	ProviderName   string
	Endpoint       string
	ServiceTier    string
	FallbackUsed   bool
	FallbackReason string
	LatencyMs      int64
	Status         string
	ErrorCode      string
	TokenUsage     UpstreamQualityUsage
	TextParams     UpstreamQualityTextParams
	ImageParams    UpstreamQualityImageParams
	Prompt         string
	PromptEnhanced bool
	CreatedAt      time.Time
}

func BuildUpstreamQualityDiagnosticReport(records []UpstreamQualityAuditRecord) UpstreamQualityDiagnosticReport {
	report := UpstreamQualityDiagnosticReport{
		TotalRecords: len(records),
	}
	for _, record := range records {
		if record.Operation == UpstreamQualityOperationTextCompletion {
			report.TextRecords++
		}
		if record.Operation == UpstreamQualityOperationImageGeneration || record.Operation == UpstreamQualityOperationImageEdit {
			report.ImageRecords++
		}
		if record.FallbackUsed {
			report.FallbackRecords++
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"model_fallback_or_mapping_mismatch",
				UpstreamQualityDiagnosticSeverityHigh,
				record,
				"Requested model differs from the upstream model or fallback was explicitly recorded.",
				"Compare requested_model, mapped_model, and upstream_model before judging output quality against official products.",
			))
		}
		if strings.TrimSpace(record.RequestedModel) == "" {
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"missing_requested_model",
				UpstreamQualityDiagnosticSeverityHigh,
				record,
				"Requested model is missing from the audit record.",
				"Record the user-requested model before provider routing so same-model quality comparisons are meaningful.",
			))
		}
		if strings.TrimSpace(record.UpstreamModel) == "" {
			report.MissingUpstreamModelRecords++
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"missing_upstream_model",
				UpstreamQualityDiagnosticSeverityMedium,
				record,
				"Actual upstream model is missing from the audit record.",
				"Capture the mapped or provider-returned model to detect aliasing, downgrade, or provider fallback.",
			))
		}
		if strings.TrimSpace(record.ProviderName) == "" {
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"missing_provider",
				UpstreamQualityDiagnosticSeverityMedium,
				record,
				"Provider label is missing from the audit record.",
				"Record a non-secret provider label so quality differences can be grouped by upstream.",
			))
		}
		if strings.TrimSpace(record.EndpointLabel) == "" && strings.TrimSpace(record.EndpointHostHash) == "" {
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"missing_endpoint_label",
				UpstreamQualityDiagnosticSeverityMedium,
				record,
				"Endpoint label or host hash is missing from the audit record.",
				"Store a redacted endpoint label or host hash without logging the full provider URL.",
			))
		}
		status := strings.ToLower(strings.TrimSpace(record.Status))
		if status == "failed" || status == "error" || strings.HasPrefix(status, "failed_") {
			report.FailedRecords++
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"upstream_failed",
				UpstreamQualityDiagnosticSeverityHigh,
				record,
				"Audit record indicates an upstream failure.",
				"Group failures by provider, upstream model, endpoint label, and sanitized error_code before changing routing.",
			))
		}
		if record.LatencyMs <= 0 && status == "succeeded" {
			report.Findings = append(report.Findings, newUpstreamQualityFinding(
				"missing_latency",
				UpstreamQualityDiagnosticSeverityLow,
				record,
				"Succeeded record is missing latency.",
				"Capture latency_ms to compare perceived quality against speed and timeout behavior.",
			))
		}
		if record.Operation == UpstreamQualityOperationTextCompletion {
			addTextQualityFindings(&report, record)
		}
		if record.Operation == UpstreamQualityOperationImageGeneration || record.Operation == UpstreamQualityOperationImageEdit {
			addImageQualityFindings(&report, record)
		}
	}
	return report
}

func addTextQualityFindings(report *UpstreamQualityDiagnosticReport, record UpstreamQualityAuditRecord) {
	if record.TokenUsage == (UpstreamQualityUsage{}) && strings.ToLower(strings.TrimSpace(record.Status)) == "succeeded" {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_text_usage",
			UpstreamQualityDiagnosticSeverityLow,
			record,
			"Succeeded text record is missing token usage.",
			"Capture input/output/total tokens so quality and cost can be reviewed together without frontend billing decisions.",
		))
	}
	if record.TextParams.Temperature == nil && record.TextParams.TopP == nil && record.TextParams.MaxTokens == nil &&
		record.TextParams.ReasoningEffort == "" && record.TextParams.ResponseFormat == "" {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_text_params",
			UpstreamQualityDiagnosticSeverityMedium,
			record,
			"Text generation parameters are missing from the audit record.",
			"Record temperature, top_p, max_tokens, reasoning effort, and response format where available before comparing model quality.",
		))
	}
}

func addImageQualityFindings(report *UpstreamQualityDiagnosticReport, record UpstreamQualityAuditRecord) {
	if strings.TrimSpace(record.ImageParams.Size) == "" {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_image_size",
			UpstreamQualityDiagnosticSeverityMedium,
			record,
			"Image size is missing from the audit record.",
			"Record requested size because official-product quality comparisons depend heavily on size and aspect ratio.",
		))
	}
	if strings.TrimSpace(record.ImageParams.Quality) == "" {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_image_quality",
			UpstreamQualityDiagnosticSeverityMedium,
			record,
			"Image quality parameter is missing from the audit record.",
			"Record quality so low-detail outputs can be separated from provider/model quality problems.",
		))
	}
	if strings.TrimSpace(record.ImageParams.OutputFormat) == "" {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_image_output_format",
			UpstreamQualityDiagnosticSeverityLow,
			record,
			"Image output format is missing from the audit record.",
			"Record output_format to detect compression, conversion, or delivery differences.",
		))
	}
	if record.ImageParams.Count <= 0 {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"missing_image_count",
			UpstreamQualityDiagnosticSeverityLow,
			record,
			"Image count is missing from the audit record.",
			"Record n/count so retry, multi-image, and manual selection quality workflows can be evaluated later.",
		))
	}
	if !record.PromptEnhanced {
		report.Findings = append(report.Findings, newUpstreamQualityFinding(
			"prompt_enhancer_not_recorded",
			UpstreamQualityDiagnosticSeverityMedium,
			record,
			"Image record does not indicate prompt enhancement.",
			"Record whether a prompt builder/enhancer was used before judging commercial image quality.",
		))
	}
}

func newUpstreamQualityFinding(
	code string,
	severity UpstreamQualityDiagnosticSeverity,
	record UpstreamQualityAuditRecord,
	message string,
	recommendation string,
) UpstreamQualityDiagnosticFinding {
	return UpstreamQualityDiagnosticFinding{
		Code:           code,
		Severity:       severity,
		Operation:      record.Operation,
		RequestedModel: record.RequestedModel,
		UpstreamModel:  record.UpstreamModel,
		ProviderName:   record.ProviderName,
		Message:        message,
		Recommendation: recommendation,
	}
}

func BuildUpstreamQualityAuditRecord(input UpstreamQualityAuditInput) UpstreamQualityAuditRecord {
	endpointLabel, endpointHostHash := auditEndpointLabels(input.Endpoint)
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	requestedModel := strings.TrimSpace(input.RequestedModel)
	mappedModel := strings.TrimSpace(input.MappedModel)
	upstreamModel := strings.TrimSpace(input.UpstreamModel)
	fallbackUsed := input.FallbackUsed
	if !fallbackUsed && requestedModel != "" && upstreamModel != "" && upstreamModel != requestedModel {
		fallbackUsed = true
	}

	return UpstreamQualityAuditRecord{
		RequestID:        trimAuditValue(input.RequestID, 128),
		Route:            trimAuditValue(input.Route, 160),
		Operation:        input.Operation,
		RequestedModel:   trimAuditValue(requestedModel, 160),
		MappedModel:      trimAuditValue(mappedModel, 160),
		UpstreamModel:    trimAuditValue(upstreamModel, 160),
		ProviderName:     trimAuditValue(input.ProviderName, 80),
		EndpointLabel:    endpointLabel,
		EndpointHostHash: endpointHostHash,
		ServiceTier:      trimAuditValue(input.ServiceTier, 80),
		FallbackUsed:     fallbackUsed,
		FallbackReason:   trimAuditValue(input.FallbackReason, 240),
		LatencyMs:        input.LatencyMs,
		Status:           trimAuditValue(input.Status, 80),
		ErrorCode:        trimAuditValue(input.ErrorCode, 120),
		TokenUsage:       input.TokenUsage,
		TextParams:       sanitizeAuditTextParams(input.TextParams),
		ImageParams:      sanitizeAuditImageParams(input.ImageParams),
		PromptHash:       auditPromptHash(input.Prompt),
		PromptPreview:    auditPromptPreview(input.Prompt, 120),
		PromptEnhanced:   input.PromptEnhanced,
		CreatedAt:        createdAt,
	}
}

func SetUpstreamQualityAuditRecordContext(ctx gatewayctx.GatewayContext, record UpstreamQualityAuditRecord) {
	if ctx == nil {
		return
	}
	ctx.SetValue(UpstreamQualityAuditContextKey, record)
}

func GetUpstreamQualityAuditRecordContext(ctx gatewayctx.GatewayContext) (UpstreamQualityAuditRecord, bool) {
	if ctx == nil {
		return UpstreamQualityAuditRecord{}, false
	}
	value, ok := ctx.Value(UpstreamQualityAuditContextKey)
	if !ok {
		return UpstreamQualityAuditRecord{}, false
	}
	record, ok := value.(UpstreamQualityAuditRecord)
	return record, ok
}

func MergeUpstreamQualityAuditRecordContext(ctx gatewayctx.GatewayContext, patch UpstreamQualityAuditInput) {
	if ctx == nil {
		return
	}
	record, ok := GetUpstreamQualityAuditRecordContext(ctx)
	if !ok {
		SetUpstreamQualityAuditRecordContext(ctx, BuildUpstreamQualityAuditRecord(patch))
		return
	}
	if patch.RequestID != "" {
		record.RequestID = trimAuditValue(patch.RequestID, 128)
	}
	if patch.Route != "" {
		record.Route = trimAuditValue(patch.Route, 160)
	}
	if patch.Operation != "" {
		record.Operation = patch.Operation
	}
	if patch.RequestedModel != "" {
		record.RequestedModel = trimAuditValue(patch.RequestedModel, 160)
	}
	if patch.MappedModel != "" {
		record.MappedModel = trimAuditValue(patch.MappedModel, 160)
	}
	if patch.UpstreamModel != "" {
		record.UpstreamModel = trimAuditValue(patch.UpstreamModel, 160)
	}
	if patch.ProviderName != "" {
		record.ProviderName = trimAuditValue(patch.ProviderName, 80)
	}
	if patch.Endpoint != "" {
		record.EndpointLabel, record.EndpointHostHash = auditEndpointLabels(patch.Endpoint)
	}
	if patch.ServiceTier != "" {
		record.ServiceTier = trimAuditValue(patch.ServiceTier, 80)
	}
	if patch.FallbackUsed {
		record.FallbackUsed = true
	}
	if patch.FallbackReason != "" {
		record.FallbackReason = trimAuditValue(patch.FallbackReason, 240)
	}
	if patch.LatencyMs > 0 {
		record.LatencyMs = patch.LatencyMs
	}
	if patch.Status != "" {
		record.Status = trimAuditValue(patch.Status, 80)
	}
	if patch.ErrorCode != "" {
		record.ErrorCode = trimAuditValue(patch.ErrorCode, 120)
	}
	if patch.TokenUsage != (UpstreamQualityUsage{}) {
		record.TokenUsage = patch.TokenUsage
	}
	if patch.TextParams != (UpstreamQualityTextParams{}) {
		record.TextParams = sanitizeAuditTextParams(patch.TextParams)
	}
	if patch.ImageParams != (UpstreamQualityImageParams{}) {
		record.ImageParams = sanitizeAuditImageParams(patch.ImageParams)
	}
	if patch.Prompt != "" {
		record.PromptHash = auditPromptHash(patch.Prompt)
		record.PromptPreview = auditPromptPreview(patch.Prompt, 120)
	}
	if patch.PromptEnhanced {
		record.PromptEnhanced = true
	}
	if !patch.CreatedAt.IsZero() {
		record.CreatedAt = patch.CreatedAt
	}
	if record.RequestedModel != "" && record.UpstreamModel != "" && record.RequestedModel != record.UpstreamModel {
		record.FallbackUsed = true
	}
	ctx.SetValue(UpstreamQualityAuditContextKey, record)
}

type UpstreamQualityPromptSample struct {
	ID        string                   `json:"id"`
	Category  string                   `json:"category"`
	Operation UpstreamQualityOperation `json:"operation"`
	Prompt    string                   `json:"prompt"`
	Criteria  []string                 `json:"criteria"`
}

type UpstreamQualityBenchmarkSample struct {
	ID                        string                   `json:"id"`
	Type                      string                   `json:"type"`
	Category                  string                   `json:"category"`
	Operation                 UpstreamQualityOperation `json:"operation"`
	Prompt                    string                   `json:"prompt"`
	ExpectedQualityDimensions []string                 `json:"expected_quality_dimensions"`
	Notes                     string                   `json:"notes,omitempty"`
	CreatedFor                string                   `json:"created_for"`
}

type UpstreamQualityManualScoreFields struct {
	Composition        string `json:"composition,omitempty"`
	CommercialAppeal   string `json:"commercial_appeal,omitempty"`
	TextQuality        string `json:"text_quality,omitempty"`
	DetailQuality      string `json:"detail_quality,omitempty"`
	PromptAdherence    string `json:"prompt_adherence,omitempty"`
	SubjectStability   string `json:"subject_stability,omitempty"`
	CommerciallyUsable string `json:"commercially_usable,omitempty"`
	ReviewerNotes      string `json:"reviewer_notes,omitempty"`
}

type UpstreamQualityBenchmarkSampleResult struct {
	SampleID              string                             `json:"sample_id"`
	Type                  string                             `json:"type"`
	Category              string                             `json:"category"`
	RequestedModel        string                             `json:"requested_model,omitempty"`
	MappedModel           string                             `json:"mapped_model,omitempty"`
	UpstreamModel         string                             `json:"upstream_model,omitempty"`
	ProviderName          string                             `json:"provider_name,omitempty"`
	EndpointLabel         string                             `json:"endpoint_label,omitempty"`
	FallbackUsed          bool                               `json:"fallback_used,omitempty"`
	FallbackReason        string                             `json:"fallback_reason,omitempty"`
	LatencyMs             int64                              `json:"latency_ms,omitempty"`
	TokenUsage            UpstreamQualityUsage               `json:"token_usage,omitempty"`
	ImageParams           UpstreamQualityImageParams         `json:"image_params,omitempty"`
	PromptHash            string                             `json:"prompt_hash,omitempty"`
	PromptPreview         string                             `json:"prompt_preview_redacted,omitempty"`
	PromptEnhanced        bool                               `json:"prompt_enhancer_used,omitempty"`
	Diagnostics           []UpstreamQualityDiagnosticFinding `json:"diagnostics,omitempty"`
	ManualScoreFields     UpstreamQualityManualScoreFields   `json:"manual_score_fields,omitempty"`
	NeedsOfficialBaseline bool                               `json:"needs_official_baseline"`
	HasAuditRecord        bool                               `json:"has_audit_record"`
}

type UpstreamQualityBenchmarkReport struct {
	TotalSamples        int                                    `json:"total_samples"`
	TextSamples         int                                    `json:"text_samples"`
	ImageSamples        int                                    `json:"image_samples"`
	MatchedAuditRecords int                                    `json:"matched_audit_records"`
	Results             []UpstreamQualityBenchmarkSampleResult `json:"results"`
	Diagnostics         UpstreamQualityDiagnosticReport        `json:"diagnostics"`
}

func UpstreamQualityPromptSamples() []UpstreamQualityPromptSample {
	return []UpstreamQualityPromptSample{
		{
			ID:        "text-context-reasoning",
			Category:  "text",
			Operation: UpstreamQualityOperationTextCompletion,
			Prompt:    "用户先描述一个电商活动目标，再要求 AI 输出可执行的活动方案、文案和风险提醒。",
			Criteria:  []string{"keeps context", "actionable plan", "no fabricated data"},
		},
		{
			ID:        "image-ecommerce-hero",
			Category:  "image",
			Operation: UpstreamQualityOperationImageGeneration,
			Prompt:    "为一款高端保温杯生成电商主图，要求商业摄影质感、真实阴影、干净构图、无乱码文字。",
			Criteria:  []string{"commercial lighting", "product fidelity", "no garbled text"},
		},
		{
			ID:        "image-xiaohongshu-cover",
			Category:  "image",
			Operation: UpstreamQualityOperationImageGeneration,
			Prompt:    "生成小红书风格护肤品封面图，要求高级感、清晰主体、适合社媒投放、不要水印。",
			Criteria:  []string{"social cover composition", "premium visual", "no watermark"},
		},
		{
			ID:        "image-restaurant-ad",
			Category:  "image",
			Operation: UpstreamQualityOperationImageGeneration,
			Prompt:    "生成餐饮新品广告图，突出热气、食欲、真实食材和可商用海报构图。",
			Criteria:  []string{"appetizing realism", "poster composition", "usable detail"},
		},
		{
			ID:        "image-app-banner",
			Category:  "image",
			Operation: UpstreamQualityOperationImageGeneration,
			Prompt:    "生成一张 AI 工作台 App banner，科技感但不花哨，强调统一输入和多能力工作流。",
			Criteria:  []string{"clear product signal", "balanced composition", "professional polish"},
		},
	}
}

func UpstreamQualityBenchmarkSamples() []UpstreamQualityBenchmarkSample {
	return []UpstreamQualityBenchmarkSample{
		{
			ID:                        "text-long-context-understanding",
			Type:                      "text",
			Category:                  "long_context_understanding",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Read a long product meeting summary and extract decisions, risks, owners, and next actions in Chinese.",
			ExpectedQualityDimensions: []string{"context retention", "decision extraction", "risk clarity", "actionability"},
			Notes:                     "Use with the same long source text across official product and relay output.",
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-complex-reasoning",
			Type:                      "text",
			Category:                  "complex_reasoning",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Design a three-stage membership growth experiment with hypotheses, metrics, sample design, and rollback criteria.",
			ExpectedQualityDimensions: []string{"causal reasoning", "metrics design", "risk handling", "specificity"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-chinese-commerce-copy",
			Type:                      "text",
			Category:                  "chinese_commerce_copy",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Write product-detail-page selling points, hero title, and short video voiceover for a premium thermos bottle.",
			ExpectedQualityDimensions: []string{"commercial tone", "Chinese fluency", "non-exaggeration", "conversion clarity"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-xiaohongshu-copy",
			Type:                      "text",
			Category:                  "xiaohongshu_copy",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Write three Xiaohongshu titles and posts for a skincare essence, realistic tone, no exaggerated medical claims.",
			ExpectedQualityDimensions: []string{"platform fit", "authentic tone", "safe claims", "hook quality"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-code-explanation",
			Type:                      "text",
			Category:                  "code_explanation",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Explain the execution order and possible bugs in a Go HTTP middleware chain.",
			ExpectedQualityDimensions: []string{"technical accuracy", "bug spotting", "Go knowledge", "clarity"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-table-organization",
			Type:                      "text",
			Category:                  "table_organization",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Turn messy product requirements into a table with priority, impact, effort, owner, and acceptance criteria.",
			ExpectedQualityDimensions: []string{"structure", "prioritization", "completeness", "format consistency"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-multiturn-followup",
			Type:                      "text",
			Category:                  "multiturn_followup",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "First propose a launch plan, then answer a follow-up: what changes if the budget is cut in half?",
			ExpectedQualityDimensions: []string{"context carryover", "adaptation", "tradeoff reasoning", "practicality"},
			CreatedFor:                "benchmark",
		},
		{
			ID:                        "text-strict-json-output",
			Type:                      "text",
			Category:                  "strict_json_output",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Return only JSON with keys summary, risks, next_steps, and confidence. No markdown, no extra text.",
			ExpectedQualityDimensions: []string{"format compliance", "JSON validity", "instruction following", "conciseness"},
			CreatedFor:                "regression",
		},
		{
			ID:                        "image-ecommerce-product-hero",
			Type:                      "image",
			Category:                  "ecommerce_product_hero",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Premium thermos bottle e-commerce hero image, clean studio lighting, realistic shadow, no garbled text.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-milk-tea-poster",
			Type:                      "image",
			Category:                  "milk_tea_commercial_poster",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Summer mango pomelo milk tea commercial poster, appetizing product focus, clear Chinese headline area.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-xiaohongshu-cover",
			Type:                      "image",
			Category:                  "xiaohongshu_cover",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Skincare essence Xiaohongshu cover, premium clean layout, social media click appeal, no watermark.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-chinese-promo-poster",
			Type:                      "image",
			Category:                  "chinese_promo_poster",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Chinese 618 promotion poster for headphones, clear hierarchy, avoid garbled Chinese characters.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-portrait-commercial",
			Type:                      "image",
			Category:                  "portrait_commercial",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Young professional woman half-body portrait, natural light, commercial photography polish, realistic hands.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-interior-design",
			Type:                      "image",
			Category:                  "interior_design",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Modern cream-style living room interior design, realistic materials, correct spatial proportions.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-restaurant-ad",
			Type:                      "image",
			Category:                  "restaurant_ad",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Hot beef noodle restaurant ad image, appetizing steam, realistic ingredients, poster-ready composition.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-app-banner",
			Type:                      "image",
			Category:                  "app_banner",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "AI workspace app banner, unified input, multi-capability workflow, professional technology style.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-brand-style",
			Type:                      "image",
			Category:                  "brand_style",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Minimal premium coffee brand visual system, packaging, cup, and storefront with unified style.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
		{
			ID:                        "image-packaging-concept",
			Type:                      "image",
			Category:                  "packaging_concept",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Bluetooth earbuds packaging concept, 3D product box display, strong brand feel, clean typography area.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
	}
}

func BuildUpstreamQualityBenchmarkReport(
	samples []UpstreamQualityBenchmarkSample,
	records map[string]UpstreamQualityAuditRecord,
) UpstreamQualityBenchmarkReport {
	report := UpstreamQualityBenchmarkReport{
		TotalSamples: len(samples),
	}
	diagnosticRecords := make([]UpstreamQualityAuditRecord, 0, len(records))
	for _, sample := range samples {
		if sample.Type == "text" {
			report.TextSamples++
		}
		if sample.Type == "image" {
			report.ImageSamples++
		}
		result := UpstreamQualityBenchmarkSampleResult{
			SampleID:              trimAuditValue(sample.ID, 120),
			Type:                  trimAuditValue(sample.Type, 40),
			Category:              trimAuditValue(sample.Category, 120),
			ManualScoreFields:     manualScoreFieldsForSample(sample),
			NeedsOfficialBaseline: true,
		}
		if record, ok := records[sample.ID]; ok {
			report.MatchedAuditRecords++
			result.HasAuditRecord = true
			result.RequestedModel = record.RequestedModel
			result.MappedModel = record.MappedModel
			result.UpstreamModel = record.UpstreamModel
			result.ProviderName = record.ProviderName
			result.EndpointLabel = record.EndpointLabel
			result.FallbackUsed = record.FallbackUsed
			result.FallbackReason = record.FallbackReason
			result.LatencyMs = record.LatencyMs
			result.TokenUsage = record.TokenUsage
			result.ImageParams = record.ImageParams
			result.PromptHash = record.PromptHash
			result.PromptPreview = auditPromptPreview(sample.Prompt, 120)
			result.PromptEnhanced = record.PromptEnhanced
			diagnostics := BuildUpstreamQualityDiagnosticReport([]UpstreamQualityAuditRecord{record})
			result.Diagnostics = diagnostics.Findings
			diagnosticRecords = append(diagnosticRecords, record)
		} else {
			result.Diagnostics = []UpstreamQualityDiagnosticFinding{{
				Code:           "missing_audit_record",
				Severity:       UpstreamQualityDiagnosticSeverityMedium,
				Operation:      sample.Operation,
				Message:        "Benchmark sample has no matching audit record.",
				Recommendation: "Run this sample through the old text/image path or attach a synthetic audit record before comparing quality.",
			}}
		}
		report.Results = append(report.Results, result)
	}
	report.Diagnostics = BuildUpstreamQualityDiagnosticReport(diagnosticRecords)
	return report
}

func commercialImageQualityDimensions() []string {
	return []string{
		"composition",
		"commercial appeal",
		"text quality",
		"detail quality",
		"prompt adherence",
		"person/product stability",
		"commercial usability",
	}
}

func manualScoreFieldsForSample(sample UpstreamQualityBenchmarkSample) UpstreamQualityManualScoreFields {
	if sample.Type != "image" {
		return UpstreamQualityManualScoreFields{
			ReviewerNotes: "Review factuality, context retention, format compliance, and usefulness.",
		}
	}
	return UpstreamQualityManualScoreFields{
		Composition:        "1-5",
		CommercialAppeal:   "1-5",
		TextQuality:        "1-5",
		DetailQuality:      "1-5",
		PromptAdherence:    "1-5",
		SubjectStability:   "1-5",
		CommerciallyUsable: "yes/no",
		ReviewerNotes:      "Compare official product output and relay output side by side.",
	}
}

func auditPromptHash(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

var auditSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|session[_-]?token|client[_-]?secret|secret|password|cookie)\s*[:=]\s*[^,\s]+`),
}

func auditPromptPreview(prompt string, maxRunes int) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || maxRunes <= 0 {
		return ""
	}
	prompt = strings.ReplaceAll(prompt, "\r", " ")
	prompt = strings.ReplaceAll(prompt, "\n", " ")
	for _, pattern := range auditSecretPatterns {
		prompt = pattern.ReplaceAllStringFunc(prompt, func(match string) string {
			if strings.HasPrefix(strings.ToLower(match), "bearer ") {
				return "Bearer [REDACTED]"
			}
			if idx := strings.IndexAny(match, ":="); idx >= 0 {
				return strings.TrimSpace(match[:idx+1]) + "[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return truncateAuditRunes(strings.Join(strings.Fields(prompt), " "), maxRunes)
}

func auditEndpointLabels(endpoint string) (label string, hostHash string) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", ""
	}
	if strings.HasPrefix(endpoint, "/") {
		return truncateAuditRunes(endpoint, 180), ""
	}
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Host != "" {
		hostHash = shortAuditHash(strings.ToLower(parsed.Host))
		path := parsed.EscapedPath()
		if path == "" {
			path = "/"
		}
		return truncateAuditRunes("host_sha256:"+hostHash+" path:"+path, 220), hostHash
	}
	hostHash = shortAuditHash(endpoint)
	return "endpoint_sha256:" + hostHash, hostHash
}

func shortAuditHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func sanitizeAuditImageParams(params UpstreamQualityImageParams) UpstreamQualityImageParams {
	params.Size = trimAuditValue(params.Size, 40)
	params.Quality = trimAuditValue(params.Quality, 40)
	params.Style = trimAuditValue(params.Style, 80)
	params.Background = trimAuditValue(params.Background, 80)
	params.OutputFormat = trimAuditValue(params.OutputFormat, 40)
	if params.Count < 0 {
		params.Count = 0
	}
	return params
}

func sanitizeAuditTextParams(params UpstreamQualityTextParams) UpstreamQualityTextParams {
	params.ReasoningEffort = trimAuditValue(params.ReasoningEffort, 40)
	params.ResponseFormat = trimAuditValue(params.ResponseFormat, 80)
	return params
}

func trimAuditValue(value string, maxRunes int) string {
	return truncateAuditRunes(strings.TrimSpace(value), maxRunes)
}

func truncateAuditRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
