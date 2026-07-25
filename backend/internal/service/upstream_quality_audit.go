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

type UpstreamImageQualityComparisonReason string

const (
	UpstreamImageQualityReasonModelMappingDifference UpstreamImageQualityComparisonReason = "model_mapping_difference"
	UpstreamImageQualityReasonFallbackOrDowngrade    UpstreamImageQualityComparisonReason = "fallback_or_downgrade"
	UpstreamImageQualityReasonMissingSize            UpstreamImageQualityComparisonReason = "missing_size"
	UpstreamImageQualityReasonMissingQuality         UpstreamImageQualityComparisonReason = "missing_quality"
	UpstreamImageQualityReasonMissingStyle           UpstreamImageQualityComparisonReason = "missing_style"
	UpstreamImageQualityReasonMissingOutputFormat    UpstreamImageQualityComparisonReason = "missing_output_format"
	UpstreamImageQualityReasonPromptEnhancerMissing  UpstreamImageQualityComparisonReason = "prompt_enhancer_missing"
	UpstreamImageQualityReasonPoorPromptFollowing    UpstreamImageQualityComparisonReason = "poor_prompt_following"
	UpstreamImageQualityReasonPoorTextQuality        UpstreamImageQualityComparisonReason = "poor_text_quality"
	UpstreamImageQualityReasonDistortedDetails       UpstreamImageQualityComparisonReason = "distorted_details"
	UpstreamImageQualityReasonLowCommercialAppeal    UpstreamImageQualityComparisonReason = "low_commercial_appeal"
	UpstreamImageQualityReasonNeedsRetrySelection    UpstreamImageQualityComparisonReason = "needs_retry_selection"
	UpstreamImageQualityReasonProviderReviewNeeded   UpstreamImageQualityComparisonReason = "provider_or_params_review_needed"
)

type UpstreamImageQualityComparisonReference struct {
	Label   string `json:"label,omitempty"`
	ImageID string `json:"image_id,omitempty"`
}

type UpstreamImageQualityManualScores struct {
	CompositionScore       int    `json:"composition_score,omitempty"`
	CommercialScore        int    `json:"commercial_score,omitempty"`
	TextQualityScore       int    `json:"text_quality_score,omitempty"`
	DetailScore            int    `json:"detail_score,omitempty"`
	PromptFollowingScore   int    `json:"prompt_following_score,omitempty"`
	SubjectStabilityScore  int    `json:"subject_stability_score,omitempty"`
	BrandConsistencyScore  int    `json:"brand_consistency_score,omitempty"`
	CommercialReadyScore   int    `json:"commercial_ready_score,omitempty"`
	TotalScore             int    `json:"total_score,omitempty"`
	CommerciallyUsable     bool   `json:"commercially_usable,omitempty"`
	ReviewerNotes          string `json:"reviewer_notes,omitempty"`
	NeedsRetry             bool   `json:"needs_retry,omitempty"`
	NeedsProviderReview    bool   `json:"needs_provider_review,omitempty"`
	NeedsPromptImprovement bool   `json:"needs_prompt_improvement,omitempty"`
}

type UpstreamImageQualityComparisonResult struct {
	BenchmarkSampleID        string                                 `json:"benchmark_sample_id"`
	OfficialReferenceLabel   string                                 `json:"official_reference_label,omitempty"`
	OfficialReferenceImageID string                                 `json:"official_reference_image_id,omitempty"`
	SiteOutputLabel          string                                 `json:"site_output_label,omitempty"`
	SiteOutputImageID        string                                 `json:"site_output_image_id,omitempty"`
	RequestedModel           string                                 `json:"requested_model,omitempty"`
	MappedModel              string                                 `json:"mapped_model,omitempty"`
	UpstreamModel            string                                 `json:"upstream_model,omitempty"`
	ProviderName             string                                 `json:"provider_name,omitempty"`
	EndpointLabel            string                                 `json:"endpoint_label,omitempty"`
	ImageParams              UpstreamQualityImageParams             `json:"image_params,omitempty"`
	PromptHash               string                                 `json:"prompt_hash,omitempty"`
	PromptPreview            string                                 `json:"prompt_preview_redacted,omitempty"`
	PromptEnhanced           bool                                   `json:"prompt_enhancer_used,omitempty"`
	FallbackUsed             bool                                   `json:"fallback_used,omitempty"`
	FallbackReason           string                                 `json:"fallback_reason,omitempty"`
	Diagnostics              []UpstreamQualityDiagnosticFinding     `json:"diagnostics,omitempty"`
	ManualScores             UpstreamImageQualityManualScores       `json:"manual_scores,omitempty"`
	AttributionReasons       []UpstreamImageQualityComparisonReason `json:"attribution_reasons,omitempty"`
	HasAuditRecord           bool                                   `json:"has_audit_record"`
	HasOfficialReference     bool                                   `json:"has_official_reference"`
	HasSiteOutput            bool                                   `json:"has_site_output"`
}

type UpstreamImageQualityComparisonReport struct {
	TotalComparisons       int                                    `json:"total_comparisons"`
	MatchedAuditRecords    int                                    `json:"matched_audit_records"`
	OfficialReferences     int                                    `json:"official_references"`
	SiteOutputs            int                                    `json:"site_outputs"`
	CommerciallyUsable     int                                    `json:"commercially_usable"`
	NeedsRetry             int                                    `json:"needs_retry"`
	NeedsProviderReview    int                                    `json:"needs_provider_review"`
	NeedsPromptImprovement int                                    `json:"needs_prompt_improvement"`
	Results                []UpstreamImageQualityComparisonResult `json:"results"`
}

type UpstreamPromptEnhancerType string

const (
	UpstreamPromptEnhancerTypeText  UpstreamPromptEnhancerType = "text"
	UpstreamPromptEnhancerTypeImage UpstreamPromptEnhancerType = "image"
)

type UpstreamPromptEnhancementProfile string

const (
	UpstreamPromptEnhancementProfileCommercialImage UpstreamPromptEnhancementProfile = "commercial_image"
	UpstreamPromptEnhancementProfileChinesePoster   UpstreamPromptEnhancementProfile = "chinese_poster"
	UpstreamPromptEnhancementProfileStructuredText  UpstreamPromptEnhancementProfile = "structured_text"
	UpstreamPromptEnhancementProfileReasoningText   UpstreamPromptEnhancementProfile = "reasoning_text"
)

type UpstreamPromptEnhancerInput struct {
	InputPrompt        string                           `json:"input_prompt,omitempty"`
	PromptType         UpstreamPromptEnhancerType       `json:"prompt_type,omitempty"`
	Category           string                           `json:"category,omitempty"`
	Language           string                           `json:"language,omitempty"`
	TargetUseCase      string                           `json:"target_use_case,omitempty"`
	EnhancementProfile UpstreamPromptEnhancementProfile `json:"enhancement_profile,omitempty"`
	BenchmarkSampleID  string                           `json:"benchmark_sample_id,omitempty"`
}

type UpstreamPromptEnhancementResult struct {
	PromptHash           string                           `json:"prompt_hash,omitempty"`
	RedactedPreview      string                           `json:"redacted_preview,omitempty"`
	PromptType           UpstreamPromptEnhancerType       `json:"prompt_type,omitempty"`
	Category             string                           `json:"category,omitempty"`
	Language             string                           `json:"language,omitempty"`
	TargetUseCase        string                           `json:"target_use_case,omitempty"`
	EnhancementProfile   UpstreamPromptEnhancementProfile `json:"enhancement_profile,omitempty"`
	EnhancedPrompt       string                           `json:"enhanced_prompt,omitempty"`
	NegativeGuidance     []string                         `json:"negative_guidance,omitempty"`
	AvoidList            []string                         `json:"avoid_list,omitempty"`
	QualityDimensions    []string                         `json:"quality_dimensions,omitempty"`
	EnhancerVersion      string                           `json:"enhancer_version,omitempty"`
	EnhancementReasons   []string                         `json:"enhancement_reasons,omitempty"`
	RiskNotes            []string                         `json:"risk_notes,omitempty"`
	ApplicableModels     []string                         `json:"applicable_models,omitempty"`
	BenchmarkSampleID    string                           `json:"benchmark_sample_id,omitempty"`
	SuggestedForProvider bool                             `json:"suggested_for_provider"`
	Diagnostics          []string                         `json:"diagnostics,omitempty"`
}

type UpstreamPromptEnhancementReport struct {
	TotalPrompts int                               `json:"total_prompts"`
	TextPrompts  int                               `json:"text_prompts"`
	ImagePrompts int                               `json:"image_prompts"`
	Results      []UpstreamPromptEnhancementResult `json:"results"`
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

func BuildUpstreamImageQualityComparisonReport(
	samples []UpstreamQualityBenchmarkSample,
	records map[string]UpstreamQualityAuditRecord,
	officialReferences map[string]UpstreamImageQualityComparisonReference,
	siteOutputs map[string]UpstreamImageQualityComparisonReference,
	manualScores map[string]UpstreamImageQualityManualScores,
) UpstreamImageQualityComparisonReport {
	report := UpstreamImageQualityComparisonReport{}
	for _, sample := range samples {
		if sample.Type != "image" {
			continue
		}
		result := UpstreamImageQualityComparisonResult{
			BenchmarkSampleID: trimAuditValue(sample.ID, 120),
			PromptPreview:     auditPromptPreview(sample.Prompt, 120),
		}
		if officialReference, ok := officialReferences[sample.ID]; ok {
			report.OfficialReferences++
			result.HasOfficialReference = true
			result.OfficialReferenceLabel = sanitizeComparisonLabel(officialReference.Label)
			result.OfficialReferenceImageID = sanitizeComparisonImageID(officialReference.ImageID)
		}
		if siteOutput, ok := siteOutputs[sample.ID]; ok {
			report.SiteOutputs++
			result.HasSiteOutput = true
			result.SiteOutputLabel = sanitizeComparisonLabel(siteOutput.Label)
			result.SiteOutputImageID = sanitizeComparisonImageID(siteOutput.ImageID)
		}
		if record, ok := records[sample.ID]; ok {
			report.MatchedAuditRecords++
			result.HasAuditRecord = true
			result.RequestedModel = record.RequestedModel
			result.MappedModel = record.MappedModel
			result.UpstreamModel = record.UpstreamModel
			result.ProviderName = record.ProviderName
			result.EndpointLabel = record.EndpointLabel
			result.ImageParams = record.ImageParams
			result.PromptHash = record.PromptHash
			result.PromptEnhanced = record.PromptEnhanced
			result.FallbackUsed = record.FallbackUsed
			result.FallbackReason = record.FallbackReason
			result.Diagnostics = BuildUpstreamQualityDiagnosticReport([]UpstreamQualityAuditRecord{record}).Findings
		} else {
			result.Diagnostics = []UpstreamQualityDiagnosticFinding{{
				Code:           "missing_audit_record",
				Severity:       UpstreamQualityDiagnosticSeverityMedium,
				Operation:      sample.Operation,
				Message:        "Image comparison sample has no matching audit record.",
				Recommendation: "Attach a sanitized audit record before judging whether provider, parameters, fallback, or prompt pipeline caused quality gaps.",
			}}
		}
		if scores, ok := manualScores[sample.ID]; ok {
			result.ManualScores = normalizeImageQualityManualScores(scores)
			if result.ManualScores.CommerciallyUsable {
				report.CommerciallyUsable++
			}
			if result.ManualScores.NeedsRetry {
				report.NeedsRetry++
			}
			if result.ManualScores.NeedsProviderReview {
				report.NeedsProviderReview++
			}
			if result.ManualScores.NeedsPromptImprovement {
				report.NeedsPromptImprovement++
			}
		}
		result.AttributionReasons = imageQualityAttributionReasons(result)
		report.Results = append(report.Results, result)
	}
	report.TotalComparisons = len(report.Results)
	return report
}

func BuildPromptEnhancementReport(inputs []UpstreamPromptEnhancerInput) UpstreamPromptEnhancementReport {
	report := UpstreamPromptEnhancementReport{
		TotalPrompts: len(inputs),
	}
	for _, input := range inputs {
		result := BuildPromptEnhancement(input)
		if result.PromptType == UpstreamPromptEnhancerTypeText {
			report.TextPrompts++
		}
		if result.PromptType == UpstreamPromptEnhancerTypeImage {
			report.ImagePrompts++
		}
		report.Results = append(report.Results, result)
	}
	return report
}

func BuildPromptEnhancement(input UpstreamPromptEnhancerInput) UpstreamPromptEnhancementResult {
	promptType := normalizePromptEnhancerType(input.PromptType)
	category := normalizePromptEnhancerCategory(input.Category)
	profile := normalizePromptEnhancementProfile(input.EnhancementProfile, promptType, category)
	language := trimAuditValue(input.Language, 40)
	if language == "" {
		language = "zh-CN"
	}
	targetUseCase := trimAuditValue(input.TargetUseCase, 120)
	if targetUseCase == "" {
		targetUseCase = "quality benchmark"
	}
	result := UpstreamPromptEnhancementResult{
		PromptHash:           auditPromptHash(input.InputPrompt),
		RedactedPreview:      safePromptEnhancerPreview(input.InputPrompt),
		PromptType:           promptType,
		Category:             category,
		Language:             language,
		TargetUseCase:        targetUseCase,
		EnhancementProfile:   profile,
		EnhancerVersion:      "offline-prompt-enhancer-v1",
		BenchmarkSampleID:    trimAuditValue(input.BenchmarkSampleID, 120),
		SuggestedForProvider: false,
		ApplicableModels:     []string{"official product comparison", "OpenAI-compatible image/text models", "workspace provider adapter review"},
		RiskNotes: []string{
			"Offline design only; do not send to a real provider without manual review.",
			"Compare against upstream quality audit fields before changing provider routing.",
		},
	}
	if result.BenchmarkSampleID != "" {
		result.Diagnostics = append(result.Diagnostics, "benchmark_sample_linked")
	}
	if strings.TrimSpace(input.InputPrompt) == "" {
		result.Diagnostics = append(result.Diagnostics, "empty_input_prompt")
	}
	if promptEnhancerHasSensitiveContent(input.InputPrompt) {
		result.Diagnostics = append(result.Diagnostics, "sensitive_input_redacted")
	}
	if promptType == UpstreamPromptEnhancerTypeImage {
		applyImagePromptEnhancement(&result, input.InputPrompt)
		return result
	}
	applyTextPromptEnhancement(&result, input.InputPrompt)
	return result
}

func applyImagePromptEnhancement(result *UpstreamPromptEnhancementResult, inputPrompt string) {
	templates := imagePromptEnhancerTemplates()
	template, ok := templates[result.Category]
	if !ok {
		result.Category = "generic_commercial_ad"
		template = templates[result.Category]
		result.Diagnostics = append(result.Diagnostics, "unknown_category_template")
	}
	promptFragment := promptEnhancerInputFragment(inputPrompt)
	result.QualityDimensions = append([]string{}, template.QualityDimensions...)
	result.NegativeGuidance = append([]string{}, template.NegativeGuidance...)
	result.AvoidList = append([]string{}, template.AvoidList...)
	result.EnhancementReasons = append([]string{}, template.Reasons...)
	result.EnhancedPrompt = strings.Join([]string{
		"Create a commercially usable image for: " + promptFragment + ".",
		"Subject: " + template.Subject + ".",
		"Scene and background: " + template.Scene + ".",
		"Composition: " + template.Composition + ".",
		"Lighting: " + template.Lighting + ".",
		"Camera and perspective: " + template.Camera + ".",
		"Color and material direction: " + template.ColorMaterial + ".",
		"Commercial polish: " + template.CommercialStyle + ".",
		"Text requirements: " + template.TextRequirements + ".",
		"Brand style: " + template.BrandStyle + ".",
		"Avoid: " + strings.Join(template.AvoidList, ", ") + ".",
	}, " ")
	if result.EnhancementProfile == UpstreamPromptEnhancementProfileChinesePoster {
		result.EnhancedPrompt += " Keep Chinese typography areas simple, legible, and easy to replace manually if the image model cannot render exact text."
	}
}

func applyTextPromptEnhancement(result *UpstreamPromptEnhancementResult, inputPrompt string) {
	templates := textPromptEnhancerTemplates()
	template, ok := templates[result.Category]
	if !ok {
		result.Category = "chinese_commerce_copy"
		template = templates[result.Category]
		result.Diagnostics = append(result.Diagnostics, "unknown_category_template")
	}
	promptFragment := promptEnhancerInputFragment(inputPrompt)
	result.QualityDimensions = append([]string{}, template.QualityDimensions...)
	result.NegativeGuidance = append([]string{}, template.NegativeGuidance...)
	result.AvoidList = append([]string{}, template.AvoidList...)
	result.EnhancementReasons = append([]string{}, template.Reasons...)
	result.EnhancedPrompt = strings.Join([]string{
		"Task: " + promptFragment + ".",
		"Role and standard: " + template.Role + ".",
		"Context handling: " + template.ContextHandling + ".",
		"Output structure: " + template.OutputStructure + ".",
		"Quality bar: " + template.QualityBar + ".",
		"Constraints: " + strings.Join(template.AvoidList, ", ") + ".",
	}, " ")
}

type promptEnhancerImageTemplate struct {
	Subject           string
	Scene             string
	Composition       string
	Lighting          string
	Camera            string
	ColorMaterial     string
	CommercialStyle   string
	TextRequirements  string
	BrandStyle        string
	QualityDimensions []string
	NegativeGuidance  []string
	AvoidList         []string
	Reasons           []string
}

type promptEnhancerTextTemplate struct {
	Role              string
	ContextHandling   string
	OutputStructure   string
	QualityBar        string
	QualityDimensions []string
	NegativeGuidance  []string
	AvoidList         []string
	Reasons           []string
}

func imagePromptEnhancerTemplates() map[string]promptEnhancerImageTemplate {
	baseAvoid := []string{
		"garbled text",
		"wrong Chinese characters",
		"deformed hands or product shape",
		"low clarity",
		"cheap stock-photo look",
		"overly complex background",
		"watermark",
	}
	baseDimensions := []string{
		"subject clarity",
		"scene",
		"composition",
		"lighting",
		"camera perspective",
		"color and material",
		"commercial appeal",
		"text quality",
		"background",
		"brand style",
	}
	baseReasons := []string{
		"expand short user prompt into production art direction",
		"make parameters reviewable against official product output",
		"reduce low-commercial-quality image results",
	}
	return map[string]promptEnhancerImageTemplate{
		"ecommerce_product_hero": {
			Subject:           "single hero product with accurate shape, clean edges, and realistic scale",
			Scene:             "premium studio set with uncluttered surface and product-focused negative space",
			Composition:       "centered product hero, balanced whitespace, strong shadow anchor, e-commerce main image crop",
			Lighting:          "softbox studio lighting, realistic reflections, crisp details",
			Camera:            "front three-quarter product photography, medium focal length, no distortion",
			ColorMaterial:     "true-to-material texture, controlled contrast, premium neutral palette",
			CommercialStyle:   "high-end marketplace-ready commercial photography",
			TextRequirements:  "no embedded text unless explicitly requested; leave clean copy area",
			BrandStyle:        "minimal premium brand language with consistent product finish",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid fake labels", "avoid warped product geometry"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"milk_tea_commercial_poster": {
			Subject:           "appetizing milk tea cup with visible toppings, condensation, and fresh ingredients",
			Scene:             "summer beverage poster setting with fruit cues and clean promotional layout",
			Composition:       "large product focus, clear headline zone, secondary ingredient accents, poster hierarchy",
			Lighting:          "bright commercial food lighting, glossy highlights, fresh color temperature",
			Camera:            "slightly low front angle to make the drink look desirable and premium",
			ColorMaterial:     "mango yellow, creamy white, natural fruit textures, clear cup material",
			CommercialStyle:   "retail beverage campaign, polished and appetizing",
			TextRequirements:  "reserve simple Chinese headline area; avoid auto-rendering complex copy",
			BrandStyle:        "fresh, youthful, clean chain-store visual style",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid messy ingredients", "avoid plastic-looking drink"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"restaurant_ad": {
			Subject:           "hero food dish with realistic ingredients, steam, and appetizing texture",
			Scene:             "restaurant advertising setup with tableware and controlled background depth",
			Composition:       "dish in foreground, clear offer/copy zone, strong food focus, poster-ready crop",
			Lighting:          "warm directional food photography light with natural highlights",
			Camera:            "close three-quarter food photography angle, shallow depth of field",
			ColorMaterial:     "rich food color, realistic oil, steam, ceramic or wood textures",
			CommercialStyle:   "delivery-platform and storefront ad quality",
			TextRequirements:  "leave headline area; do not hallucinate unreadable menu text",
			BrandStyle:        "clean, appetizing, not over-saturated",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid unappetizing food texture", "avoid dirty table details"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"xiaohongshu_cover": {
			Subject:           "clear lifestyle or product subject with scroll-stopping social cover appeal",
			Scene:             "clean editorial social media layout with premium props and soft background",
			Composition:       "vertical cover composition, strong focal point, readable title zone",
			Lighting:          "soft natural light, premium beauty/lifestyle finish",
			Camera:            "editorial close-up with flattering perspective",
			ColorMaterial:     "soft but not washed-out palette, refined material detail",
			CommercialStyle:   "high-quality Xiaohongshu cover, click-worthy but credible",
			TextRequirements:  "simple title area only; avoid dense generated Chinese text",
			BrandStyle:        "premium social commerce, clean and aspirational",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid fake watermark", "avoid noisy collage"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"chinese_promo_poster": {
			Subject:           "promotional product or offer with clear product hierarchy",
			Scene:             "Chinese retail campaign poster with clean headline and offer zones",
			Composition:       "top headline zone, central product hero, bottom CTA area, simple layers",
			Lighting:          "bright campaign lighting with crisp separation",
			Camera:            "front-facing commercial ad perspective",
			ColorMaterial:     "bold but controlled promotional colors, high contrast, clean edges",
			CommercialStyle:   "e-commerce campaign poster suitable for manual typography polish",
			TextRequirements:  "reserve editable Chinese text blocks; avoid rendering long Chinese copy inside the image",
			BrandStyle:        "modern Chinese promotional design without clutter",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid unreadable Chinese typography", "avoid fake QR codes"},
			AvoidList:         baseAvoid,
			Reasons:           append(baseReasons, "make Chinese text risk explicit before commercial use"),
		},
		"portrait_commercial": {
			Subject:           "realistic person with stable face, natural hands, and commercially usable styling",
			Scene:             "clean portrait set with wardrobe and background matching the brand tone",
			Composition:       "half-body or three-quarter portrait, clear face, balanced negative space",
			Lighting:          "soft professional portrait lighting with natural skin texture",
			Camera:            "85mm editorial portrait look, no wide-angle distortion",
			ColorMaterial:     "natural skin tones, premium clothing fabric, controlled color palette",
			CommercialStyle:   "brand campaign portrait, polished but believable",
			TextRequirements:  "no generated text over face or body",
			BrandStyle:        "credible, refined, modern",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid distorted hands", "avoid waxy skin", "avoid uncanny face"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"interior_design": {
			Subject:           "room layout with realistic furniture, materials, and spatial proportions",
			Scene:             "interior design render with coherent style and practical layout",
			Composition:       "wide room view, clear focal wall, balanced furniture placement",
			Lighting:          "natural window light plus soft interior fill lighting",
			Camera:            "architectural interior lens, straight verticals, realistic perspective",
			ColorMaterial:     "accurate wood, fabric, stone, metal, and wall finishes",
			CommercialStyle:   "portfolio-ready interior visualization",
			TextRequirements:  "no text overlays",
			BrandStyle:        "cohesive interior style with clean material palette",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid impossible furniture scale", "avoid warped room geometry"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"app_banner": {
			Subject:           "software product concept with clear interface/product signal",
			Scene:             "professional SaaS or AI workspace banner environment",
			Composition:       "wide banner crop, clear headline area, product visual as first-viewport signal",
			Lighting:          "clean digital product lighting, high readability",
			Camera:            "straight-on or slight perspective UI/product showcase",
			ColorMaterial:     "modern UI colors, crisp panels, restrained contrast",
			CommercialStyle:   "conversion-ready app landing banner",
			TextRequirements:  "reserve text area; avoid tiny unreadable UI copy",
			BrandStyle:        "professional technology product, not generic sci-fi",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid random dashboard noise", "avoid fake tiny text"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"brand_style": {
			Subject:           "brand system objects such as packaging, cup, signage, and social visual",
			Scene:             "cohesive brand presentation with multiple touchpoints",
			Composition:       "orderly brand board, consistent spacing, clear hierarchy",
			Lighting:          "premium studio lighting with realistic shadows",
			Camera:            "product and brand collateral presentation perspective",
			ColorMaterial:     "consistent brand colors, print-ready material cues",
			CommercialStyle:   "brand identity concept suitable for client review",
			TextRequirements:  "use simple placeholder typography areas, not generated final text",
			BrandStyle:        "cohesive, premium, memorable",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid inconsistent logos", "avoid unreadable brand names"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"packaging_concept": {
			Subject:           "packaging box, label, and product display with stable geometry",
			Scene:             "studio packaging concept presentation",
			Composition:       "front package hero with side detail, clean copy zones, product context",
			Lighting:          "soft packaging photography light with crisp edges",
			Camera:            "front three-quarter packaging view, accurate perspective",
			ColorMaterial:     "paper, foil, plastic, or matte texture rendered realistically",
			CommercialStyle:   "client-ready packaging concept visualization",
			TextRequirements:  "avoid final small text; reserve editable label zones",
			BrandStyle:        "premium and coherent packaging language",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid warped boxes", "avoid fake nutritional text"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
		"generic_commercial_ad": {
			Subject:           "clear commercial subject with product or service benefit visible",
			Scene:             "clean ad setting matched to the use case",
			Composition:       "strong focal subject, copy-safe negative space, campaign-ready crop",
			Lighting:          "professional commercial lighting",
			Camera:            "stable advertising perspective with no distortion",
			ColorMaterial:     "controlled brand palette and realistic material cues",
			CommercialStyle:   "usable advertising visual with polished details",
			TextRequirements:  "reserve editable text areas and avoid generated dense copy",
			BrandStyle:        "consistent, premium, and not generic",
			QualityDimensions: baseDimensions,
			NegativeGuidance:  []string{"avoid visual clutter", "avoid low-quality stock look"},
			AvoidList:         baseAvoid,
			Reasons:           baseReasons,
		},
	}
}

func textPromptEnhancerTemplates() map[string]promptEnhancerTextTemplate {
	baseAvoid := []string{
		"unsupported claims",
		"fabricated facts",
		"vague generic wording",
		"hidden assumptions",
		"format drift",
	}
	return map[string]promptEnhancerTextTemplate{
		"chinese_commerce_copy": {
			Role:              "act as a senior Chinese e-commerce copywriter with realistic conversion taste",
			ContextHandling:   "extract product, audience, platform, tone, selling points, and compliance risks before writing",
			OutputStructure:   "return hero title, selling bullets, short-video voiceover, and risk notes",
			QualityBar:        "specific, credible, commercially sharp, and free of exaggerated medical or guaranteed claims",
			QualityDimensions: []string{"commercial tone", "Chinese fluency", "conversion clarity", "claim safety"},
			NegativeGuidance:  []string{"avoid exaggerated claims", "avoid generic luxury adjectives"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"turn short commercial request into a complete copy brief", "make safety and platform fit explicit"},
		},
		"xiaohongshu_copy": {
			Role:              "act as a Xiaohongshu content strategist who writes natural Chinese social commerce posts",
			ContextHandling:   "identify hook, user scenario, believable personal tone, benefits, and boundaries",
			OutputStructure:   "return title options, post body, topic tags, and claim safety notes",
			QualityBar:        "authentic, specific, not over-promising, and easy to scan on mobile",
			QualityDimensions: []string{"platform fit", "authentic tone", "hook quality", "safe claims"},
			NegativeGuidance:  []string{"avoid fake personal experience", "avoid medical guarantees"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"adapt generic copy into platform-native structure", "reduce fake or exaggerated social tone"},
		},
		"long_context_understanding": {
			Role:              "act as a precise analyst who preserves context and distinguishes facts from assumptions",
			ContextHandling:   "read all provided context, extract decisions, risks, owners, dependencies, and open questions",
			OutputStructure:   "return sections for summary, decisions, risks, owners, next actions, and missing information",
			QualityBar:        "faithful to source context, no invented owners or deadlines, actionable and concise",
			QualityDimensions: []string{"context retention", "decision extraction", "risk clarity", "actionability"},
			NegativeGuidance:  []string{"avoid inventing facts", "avoid collapsing unresolved questions into decisions"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"make context retention testable", "separate source facts from model inference"},
		},
		"strict_json_output": {
			Role:              "act as a strict JSON formatter and validator",
			ContextHandling:   "infer only requested fields from provided context and use null when unavailable",
			OutputStructure:   "return valid JSON only with the requested schema, no markdown and no prose",
			QualityBar:        "parseable JSON, stable key order, correct types, no trailing commentary",
			QualityDimensions: []string{"JSON validity", "schema compliance", "format stability", "instruction following"},
			NegativeGuidance:  []string{"avoid markdown fences", "avoid extra explanations"},
			AvoidList:         append(baseAvoid, "markdown fences", "extra prose"),
			Reasons:           []string{"convert natural-language request into strict output contract", "reduce format drift"},
		},
		"table_organization": {
			Role:              "act as a product operations analyst",
			ContextHandling:   "cluster messy requirements by goal, dependency, risk, owner, and acceptance criteria",
			OutputStructure:   "return a table with priority, impact, effort, owner, acceptance criteria, and notes",
			QualityBar:        "clear priorities, comparable rows, no duplicate items, practical acceptance criteria",
			QualityDimensions: []string{"structure", "prioritization", "completeness", "format consistency"},
			NegativeGuidance:  []string{"avoid vague rows", "avoid missing acceptance criteria"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"force messy requirements into comparable operating structure"},
		},
		"code_explanation": {
			Role:              "act as a senior engineer explaining code behavior and risks",
			ContextHandling:   "trace control flow, state changes, error handling, and edge cases from the provided code",
			OutputStructure:   "return execution order, likely bugs, risk level, and minimal fix suggestions",
			QualityBar:        "technically accurate, grounded in code, no invented APIs",
			QualityDimensions: []string{"technical accuracy", "bug spotting", "clarity", "minimal fix quality"},
			NegativeGuidance:  []string{"avoid broad refactors", "avoid guessing missing code"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"make code explanation grounded and actionable"},
		},
		"multiturn_followup": {
			Role:              "act as a planning assistant that preserves prior constraints across turns",
			ContextHandling:   "carry forward previous assumptions, constraints, and tradeoffs before answering the follow-up",
			OutputStructure:   "return updated answer, changed assumptions, unchanged constraints, and recommended next step",
			QualityBar:        "does not forget prior context and clearly explains what changed",
			QualityDimensions: []string{"context carryover", "adaptation", "tradeoff reasoning", "practicality"},
			NegativeGuidance:  []string{"avoid restarting from scratch", "avoid ignoring prior constraints"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"make multi-turn memory and adaptation measurable"},
		},
		"complex_reasoning": {
			Role:              "act as a rigorous strategist who reasons through tradeoffs before recommendations",
			ContextHandling:   "define assumptions, variables, constraints, failure modes, and measurable success criteria",
			OutputStructure:   "return reasoning summary, recommendation, alternatives, risks, metrics, and rollback criteria",
			QualityBar:        "specific, testable, logically coherent, and not hand-wavy",
			QualityDimensions: []string{"causal reasoning", "metrics design", "risk handling", "specificity"},
			NegativeGuidance:  []string{"avoid generic frameworks", "avoid unsupported confidence"},
			AvoidList:         baseAvoid,
			Reasons:           []string{"make complex reasoning output auditable against fixed benchmarks"},
		},
	}
}

func normalizePromptEnhancerType(value UpstreamPromptEnhancerType) UpstreamPromptEnhancerType {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case string(UpstreamPromptEnhancerTypeImage):
		return UpstreamPromptEnhancerTypeImage
	default:
		return UpstreamPromptEnhancerTypeText
	}
}

func normalizePromptEnhancerCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	category = strings.ReplaceAll(category, " ", "_")
	category = strings.ReplaceAll(category, "-", "_")
	if category == "" {
		return "generic_commercial_ad"
	}
	return trimAuditValue(category, 120)
}

func normalizePromptEnhancementProfile(
	profile UpstreamPromptEnhancementProfile,
	promptType UpstreamPromptEnhancerType,
	category string,
) UpstreamPromptEnhancementProfile {
	value := UpstreamPromptEnhancementProfile(strings.TrimSpace(string(profile)))
	if value != "" {
		return value
	}
	if promptType == UpstreamPromptEnhancerTypeImage {
		if category == "chinese_promo_poster" {
			return UpstreamPromptEnhancementProfileChinesePoster
		}
		return UpstreamPromptEnhancementProfileCommercialImage
	}
	if category == "complex_reasoning" || category == "long_context_understanding" || category == "multiturn_followup" {
		return UpstreamPromptEnhancementProfileReasoningText
	}
	return UpstreamPromptEnhancementProfileStructuredText
}

func normalizeImageQualityManualScores(scores UpstreamImageQualityManualScores) UpstreamImageQualityManualScores {
	scores.CompositionScore = clampQualityScore(scores.CompositionScore)
	scores.CommercialScore = clampQualityScore(scores.CommercialScore)
	scores.TextQualityScore = clampQualityScore(scores.TextQualityScore)
	scores.DetailScore = clampQualityScore(scores.DetailScore)
	scores.PromptFollowingScore = clampQualityScore(scores.PromptFollowingScore)
	scores.SubjectStabilityScore = clampQualityScore(scores.SubjectStabilityScore)
	scores.BrandConsistencyScore = clampQualityScore(scores.BrandConsistencyScore)
	scores.CommercialReadyScore = clampQualityScore(scores.CommercialReadyScore)
	if scores.TotalScore <= 0 {
		scores.TotalScore = scores.CompositionScore +
			scores.CommercialScore +
			scores.TextQualityScore +
			scores.DetailScore +
			scores.PromptFollowingScore +
			scores.SubjectStabilityScore +
			scores.BrandConsistencyScore +
			scores.CommercialReadyScore
	}
	scores.TotalScore = clampTotalQualityScore(scores.TotalScore)
	scores.ReviewerNotes = sanitizeReviewerNotes(scores.ReviewerNotes)
	return scores
}

func imageQualityAttributionReasons(result UpstreamImageQualityComparisonResult) []UpstreamImageQualityComparisonReason {
	reasons := []UpstreamImageQualityComparisonReason{}
	addReason := func(reason UpstreamImageQualityComparisonReason) {
		for _, existing := range reasons {
			if existing == reason {
				return
			}
		}
		reasons = append(reasons, reason)
	}
	if result.HasAuditRecord {
		if result.RequestedModel != "" && result.UpstreamModel != "" && result.RequestedModel != result.UpstreamModel {
			addReason(UpstreamImageQualityReasonModelMappingDifference)
		}
		if result.FallbackUsed {
			addReason(UpstreamImageQualityReasonFallbackOrDowngrade)
			addReason(UpstreamImageQualityReasonProviderReviewNeeded)
		}
		if strings.TrimSpace(result.ImageParams.Size) == "" {
			addReason(UpstreamImageQualityReasonMissingSize)
		}
		if strings.TrimSpace(result.ImageParams.Quality) == "" {
			addReason(UpstreamImageQualityReasonMissingQuality)
			addReason(UpstreamImageQualityReasonProviderReviewNeeded)
		}
		if strings.TrimSpace(result.ImageParams.Style) == "" {
			addReason(UpstreamImageQualityReasonMissingStyle)
		}
		if strings.TrimSpace(result.ImageParams.OutputFormat) == "" {
			addReason(UpstreamImageQualityReasonMissingOutputFormat)
		}
		if !result.PromptEnhanced {
			addReason(UpstreamImageQualityReasonPromptEnhancerMissing)
		}
	}
	scores := result.ManualScores
	if scores.PromptFollowingScore > 0 && scores.PromptFollowingScore <= 2 {
		addReason(UpstreamImageQualityReasonPoorPromptFollowing)
	}
	if scores.TextQualityScore > 0 && scores.TextQualityScore <= 2 {
		addReason(UpstreamImageQualityReasonPoorTextQuality)
	}
	if scores.DetailScore > 0 && scores.DetailScore <= 2 {
		addReason(UpstreamImageQualityReasonDistortedDetails)
	}
	if scores.CommercialScore > 0 && scores.CommercialScore <= 2 {
		addReason(UpstreamImageQualityReasonLowCommercialAppeal)
	}
	if scores.NeedsRetry || (scores.TotalScore > 0 && scores.TotalScore < 28) || (scores.CommercialReadyScore > 0 && scores.CommercialReadyScore <= 2) {
		addReason(UpstreamImageQualityReasonNeedsRetrySelection)
	}
	if scores.NeedsProviderReview {
		addReason(UpstreamImageQualityReasonProviderReviewNeeded)
	}
	if scores.NeedsPromptImprovement {
		addReason(UpstreamImageQualityReasonPromptEnhancerMissing)
	}
	return reasons
}

func clampQualityScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 5 {
		return 5
	}
	return score
}

func clampTotalQualityScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 40 {
		return 40
	}
	return score
}

func sanitizeComparisonLabel(label string) string {
	label = strings.TrimSpace(label)
	lower := strings.ToLower(label)
	if strings.Contains(lower, "://") || strings.Contains(lower, "token=") || strings.Contains(lower, "signature=") {
		return "label_sha256:" + shortAuditHash(label)
	}
	return auditPromptPreview(label, 120)
}

func sanitizeComparisonImageID(imageID string) string {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ""
	}
	lower := strings.ToLower(imageID)
	if strings.Contains(lower, "://") || strings.Contains(lower, "token=") || strings.Contains(lower, "signature=") {
		return "image_ref_sha256:" + shortAuditHash(imageID)
	}
	return trimAuditValue(imageID, 160)
}

func sanitizeReviewerNotes(notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return ""
	}
	for _, pattern := range auditSecretPatterns {
		if pattern.MatchString(notes) {
			return "reviewer_notes_sha256:" + shortAuditHash(notes)
		}
	}
	return auditPromptPreview(notes, 180)
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

func promptEnhancerHasSensitiveContent(prompt string) bool {
	for _, pattern := range auditSecretPatterns {
		if pattern.MatchString(prompt) {
			return true
		}
	}
	return false
}

func safePromptEnhancerPreview(prompt string) string {
	if promptEnhancerHasSensitiveContent(prompt) {
		return "prompt_sha256:" + shortAuditHash(prompt)
	}
	return auditPromptPreview(prompt, 120)
}

func promptEnhancerInputFragment(prompt string) string {
	if promptEnhancerHasSensitiveContent(prompt) {
		return "[REDACTED_INPUT_PROMPT]"
	}
	preview := auditPromptPreview(prompt, 220)
	if preview == "" {
		return "[EMPTY_INPUT_PROMPT]"
	}
	return preview
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
