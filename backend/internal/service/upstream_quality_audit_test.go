package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func TestBuildUpstreamQualityAuditRecordRedactsSensitivePrompt(t *testing.T) {
	t.Parallel()

	prompt := "请生成商品图 Authorization: Bearer sk-live-secret api_key=abc123 cookie=sessionid password=hunter2 Product name: premium bottle"
	record := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      "req_123",
		Route:          "/api/v1/image-studio/generate",
		Operation:      UpstreamQualityOperationImageGeneration,
		RequestedModel: "gpt-image-2",
		UpstreamModel:  "gpt-image-2-upstream",
		ProviderName:   "openai",
		Endpoint:       "https://api.openai.com/v1/images/generations?api_key=must-not-appear",
		ImageParams: UpstreamQualityImageParams{
			Size:         "1024x1024",
			Quality:      "high",
			Style:        "commercial studio",
			Background:   "white",
			OutputFormat: "png",
			Count:        2,
		},
		Prompt:         prompt,
		PromptEnhanced: true,
		CreatedAt:      time.Unix(1700000000, 0),
	})

	if record.PromptHash == "" {
		t.Fatal("expected prompt hash")
	}
	for _, forbidden := range []string{"sk-live-secret", "abc123", "sessionid", "hunter2", "api.openai.com", "must-not-appear"} {
		if strings.Contains(record.PromptPreview, forbidden) || strings.Contains(record.EndpointLabel, forbidden) {
			t.Fatalf("audit record leaked %q: preview=%q endpoint=%q", forbidden, record.PromptPreview, record.EndpointLabel)
		}
	}
	if record.EndpointHostHash == "" {
		t.Fatalf("expected endpoint host hash for full URL")
	}
	if record.EndpointLabel == "" || !strings.Contains(record.EndpointLabel, "/v1/images/generations") {
		t.Fatalf("expected endpoint label to keep normalized path only, got %q", record.EndpointLabel)
	}
	if !record.FallbackUsed {
		t.Fatalf("expected fallback/mapping flag when upstream model differs")
	}
	if record.ImageParams.Quality != "high" || record.ImageParams.OutputFormat != "png" || record.ImageParams.Count != 2 {
		t.Fatalf("image params not preserved: %+v", record.ImageParams)
	}
}

func TestBuildUpstreamQualityAuditRecordKeepsTextParams(t *testing.T) {
	t.Parallel()

	temp := 0.7
	maxTokens := 512
	record := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		Route:          "/api/v1/chat-studio/complete",
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: "gpt-5.5",
		MappedModel:    "gpt-5.5",
		UpstreamModel:  "gpt-5.5",
		ProviderName:   "openai",
		Endpoint:       "/v1/chat/completions",
		TextParams: UpstreamQualityTextParams{
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		},
		Prompt: "请根据上下文输出一个可靠方案",
	})

	if record.TextParams.Temperature == nil || *record.TextParams.Temperature != temp {
		t.Fatalf("expected temperature to be preserved, got %+v", record.TextParams)
	}
	if record.TextParams.MaxTokens == nil || *record.TextParams.MaxTokens != maxTokens {
		t.Fatalf("expected max_tokens to be preserved, got %+v", record.TextParams)
	}
	if record.FallbackUsed {
		t.Fatalf("did not expect fallback when requested and upstream model match")
	}
}

func TestMergeUpstreamQualityAuditRecordContextAddsUpstreamFields(t *testing.T) {
	t.Parallel()

	req := newAuditTestRequest()
	ctx := gatewayctx.NewNative(req, nil, nil, "127.0.0.1")
	SetUpstreamQualityAuditRecordContext(ctx, BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		Route:          "/api/v1/image-studio/generate",
		Operation:      UpstreamQualityOperationImageGeneration,
		RequestedModel: "gpt-image-2",
		Endpoint:       "/v1/images/generations",
		Prompt:         "商品主图",
	}))

	MergeUpstreamQualityAuditRecordContext(ctx, UpstreamQualityAuditInput{
		ProviderName:  "openai",
		MappedModel:   "gpt-image-2-prod",
		UpstreamModel: "gpt-image-2-prod",
		Status:        "succeeded",
		LatencyMs:     345,
		TokenUsage: UpstreamQualityUsage{
			ImageCount: 1,
		},
	})

	record, ok := GetUpstreamQualityAuditRecordContext(ctx)
	if !ok {
		t.Fatal("expected audit record in context")
	}
	if record.ProviderName != "openai" || record.UpstreamModel != "gpt-image-2-prod" {
		t.Fatalf("upstream fields were not merged: %+v", record)
	}
	if !record.FallbackUsed {
		t.Fatalf("expected fallback/mapping flag when upstream model differs")
	}
	if record.Status != "succeeded" || record.LatencyMs != 345 || record.TokenUsage.ImageCount != 1 {
		t.Fatalf("result fields were not merged: %+v", record)
	}
}

func TestUpstreamQualityPromptSamplesCoverTextAndCommercialImages(t *testing.T) {
	t.Parallel()

	samples := UpstreamQualityPromptSamples()
	if len(samples) < 5 {
		t.Fatalf("expected at least 5 quality samples, got %d", len(samples))
	}
	seenText := false
	imageCategories := map[string]bool{}
	for _, sample := range samples {
		if strings.TrimSpace(sample.ID) == "" || strings.TrimSpace(sample.Prompt) == "" {
			t.Fatalf("sample must include id and prompt: %+v", sample)
		}
		if sample.Operation == UpstreamQualityOperationTextCompletion {
			seenText = true
		}
		if sample.Operation == UpstreamQualityOperationImageGeneration {
			imageCategories[sample.ID] = true
		}
	}
	if !seenText {
		t.Fatalf("expected a text sample")
	}
	for _, id := range []string{"image-ecommerce-hero", "image-xiaohongshu-cover", "image-restaurant-ad", "image-app-banner"} {
		if !imageCategories[id] {
			t.Fatalf("missing image quality sample %s", id)
		}
	}
}

func TestBuildUpstreamQualityDiagnosticReportFlagsModelAndImageGaps(t *testing.T) {
	t.Parallel()

	report := BuildUpstreamQualityDiagnosticReport([]UpstreamQualityAuditRecord{
		BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/image-studio/generate",
			Operation:      UpstreamQualityOperationImageGeneration,
			RequestedModel: "gpt-image-official",
			UpstreamModel:  "cheap-image-fallback",
			ProviderName:   "image-provider",
			Endpoint:       "/v1/images/generations",
			ImageParams: UpstreamQualityImageParams{
				Size: "1024x1024",
			},
			Prompt:         "commercial product poster",
			PromptEnhanced: false,
			Status:         "succeeded",
		}),
	})

	if report.TotalRecords != 1 || report.ImageRecords != 1 || report.FallbackRecords != 1 {
		t.Fatalf("unexpected report counters: %+v", report)
	}
	for _, code := range []string{
		"model_fallback_or_mapping_mismatch",
		"missing_image_quality",
		"missing_image_output_format",
		"missing_image_count",
		"prompt_enhancer_not_recorded",
	} {
		if !diagnosticReportHasFinding(report, code) {
			t.Fatalf("expected finding %s in %+v", code, report.Findings)
		}
	}
}

func TestBuildUpstreamQualityDiagnosticReportDoesNotLeakSensitivePrompt(t *testing.T) {
	t.Parallel()

	record := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		Route:          "/api/v1/chat-studio/complete",
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: "gpt-5.5",
		UpstreamModel:  "gpt-5.5",
		ProviderName:   "openai",
		Endpoint:       "https://api.openai.com/v1/chat/completions?api_key=must-not-appear",
		Prompt:         "Authorization: Bearer sk-live-secret cookie=sessionid password=hunter2 user asks for campaign plan",
		Status:         "succeeded",
	})
	report := BuildUpstreamQualityDiagnosticReport([]UpstreamQualityAuditRecord{record})

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, forbidden := range []string{"sk-live-secret", "sessionid", "hunter2", "api.openai.com", "must-not-appear", "campaign plan"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("diagnostic report leaked %q: %s", forbidden, body)
		}
	}
	if !diagnosticReportHasFinding(report, "missing_text_params") {
		t.Fatalf("expected missing text params finding: %+v", report.Findings)
	}
}

func TestBuildUpstreamQualityDiagnosticReportFlagsFailedAndMissingFields(t *testing.T) {
	t.Parallel()

	report := BuildUpstreamQualityDiagnosticReport([]UpstreamQualityAuditRecord{
		BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:     "/api/v1/chat-studio/complete",
			Operation: UpstreamQualityOperationTextCompletion,
			Status:    "failed",
			ErrorCode: "provider_timeout",
		}),
	})

	if report.FailedRecords != 1 || report.MissingUpstreamModelRecords != 1 {
		t.Fatalf("unexpected counters: %+v", report)
	}
	for _, code := range []string{"missing_requested_model", "missing_upstream_model", "missing_provider", "missing_endpoint_label", "upstream_failed"} {
		if !diagnosticReportHasFinding(report, code) {
			t.Fatalf("expected finding %s in %+v", code, report.Findings)
		}
	}
}

func TestUpstreamQualityBenchmarkSamplesCoverRequiredCategories(t *testing.T) {
	t.Parallel()

	samples := UpstreamQualityBenchmarkSamples()
	textCategories := map[string]bool{}
	imageCategories := map[string]bool{}
	for _, sample := range samples {
		if strings.TrimSpace(sample.ID) == "" || strings.TrimSpace(sample.Prompt) == "" {
			t.Fatalf("sample must include id and prompt: %+v", sample)
		}
		if sample.Type == "text" {
			textCategories[sample.Category] = true
		}
		if sample.Type == "image" {
			imageCategories[sample.Category] = true
			for _, dimension := range []string{
				"composition",
				"commercial appeal",
				"text quality",
				"detail quality",
				"prompt adherence",
				"person/product stability",
				"commercial usability",
			} {
				if !testContainsString(sample.ExpectedQualityDimensions, dimension) {
					t.Fatalf("image sample %s missing dimension %q: %+v", sample.ID, dimension, sample.ExpectedQualityDimensions)
				}
			}
		}
	}

	for _, category := range []string{
		"long_context_understanding",
		"complex_reasoning",
		"chinese_commerce_copy",
		"xiaohongshu_copy",
		"code_explanation",
		"table_organization",
		"multiturn_followup",
		"strict_json_output",
	} {
		if !textCategories[category] {
			t.Fatalf("missing text benchmark category %s", category)
		}
	}
	for _, category := range []string{
		"ecommerce_product_hero",
		"milk_tea_commercial_poster",
		"xiaohongshu_cover",
		"chinese_promo_poster",
		"portrait_commercial",
		"interior_design",
		"restaurant_ad",
		"app_banner",
		"brand_style",
		"packaging_concept",
	} {
		if !imageCategories[category] {
			t.Fatalf("missing image benchmark category %s", category)
		}
	}
}

func TestBuildUpstreamQualityBenchmarkReportMergesSyntheticAuditRecords(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{
		{
			ID:                        "text-strict-json-output",
			Type:                      "text",
			Category:                  "strict_json_output",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Return only JSON.",
			ExpectedQualityDimensions: []string{"format compliance"},
			CreatedFor:                "regression",
		},
		{
			ID:                        "image-ecommerce-product-hero",
			Type:                      "image",
			Category:                  "ecommerce_product_hero",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Premium product hero image.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
	}
	temp := 0.2
	records := map[string]UpstreamQualityAuditRecord{
		"text-strict-json-output": BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/chat-studio/complete",
			Operation:      UpstreamQualityOperationTextCompletion,
			RequestedModel: "gpt-5.5",
			MappedModel:    "gpt-5.5",
			UpstreamModel:  "gpt-5.5",
			ProviderName:   "openai-compatible-main",
			Endpoint:       "/v1/chat/completions",
			TextParams: UpstreamQualityTextParams{
				Temperature: &temp,
			},
			TokenUsage: UpstreamQualityUsage{
				InputTokens:  120,
				OutputTokens: 80,
				TotalTokens:  200,
			},
			Prompt:         "Authorization: Bearer sk-live-secret return only JSON",
			PromptEnhanced: true,
			LatencyMs:      900,
			Status:         "succeeded",
		}),
		"image-ecommerce-product-hero": BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/image-studio/generate",
			Operation:      UpstreamQualityOperationImageGeneration,
			RequestedModel: "gpt-image-2",
			MappedModel:    "gpt-image-2",
			UpstreamModel:  "gpt-image-2-compatible",
			ProviderName:   "image-provider-main",
			Endpoint:       "/v1/images/generations",
			ImageParams: UpstreamQualityImageParams{
				Size:         "1024x1536",
				Quality:      "high",
				Style:        "commercial",
				Background:   "studio",
				OutputFormat: "png",
				Count:        1,
			},
			Prompt:         "premium product hero",
			PromptEnhanced: false,
			LatencyMs:      3200,
			Status:         "succeeded",
		}),
	}

	report := BuildUpstreamQualityBenchmarkReport(samples, records)
	if report.TotalSamples != 2 || report.TextSamples != 1 || report.ImageSamples != 1 || report.MatchedAuditRecords != 2 {
		t.Fatalf("unexpected benchmark counters: %+v", report)
	}
	if len(report.Results) != 2 {
		t.Fatalf("expected two sample results, got %d", len(report.Results))
	}
	textResult := benchmarkResultByID(report, "text-strict-json-output")
	if textResult.RequestedModel != "gpt-5.5" || textResult.UpstreamModel != "gpt-5.5" {
		t.Fatalf("text model fields not merged: %+v", textResult)
	}
	imageResult := benchmarkResultByID(report, "image-ecommerce-product-hero")
	if imageResult.ImageParams.Size != "1024x1536" || imageResult.ImageParams.Quality != "high" || !imageResult.FallbackUsed {
		t.Fatalf("image params/fallback not merged: %+v", imageResult)
	}
	if imageResult.ManualScoreFields.CommerciallyUsable == "" {
		t.Fatalf("expected manual image score fields: %+v", imageResult.ManualScoreFields)
	}
	if !benchmarkResultHasFinding(imageResult, "model_fallback_or_mapping_mismatch") ||
		!benchmarkResultHasFinding(imageResult, "prompt_enhancer_not_recorded") {
		t.Fatalf("expected image diagnostic findings: %+v", imageResult.Diagnostics)
	}
}

func TestBuildUpstreamQualityBenchmarkReportDoesNotLeakSensitiveData(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{{
		ID:                        "text-sensitive-redaction",
		Type:                      "text",
		Category:                  "strict_json_output",
		Operation:                 UpstreamQualityOperationTextCompletion,
		Prompt:                    "Synthetic non-sensitive sample.",
		ExpectedQualityDimensions: []string{"format compliance"},
		CreatedFor:                "benchmark",
	}}
	records := map[string]UpstreamQualityAuditRecord{
		"text-sensitive-redaction": BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/chat-studio/complete",
			Operation:      UpstreamQualityOperationTextCompletion,
			RequestedModel: "gpt-5.5",
			UpstreamModel:  "gpt-5.5",
			ProviderName:   "openai",
			Endpoint:       "https://api.openai.com/v1/chat/completions?api_key=must-not-appear",
			Prompt:         "Authorization: Bearer sk-live-secret cookie=sessionid password=hunter2 private campaign strategy",
			Status:         "succeeded",
		}),
	}
	report := BuildUpstreamQualityBenchmarkReport(samples, records)
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, forbidden := range []string{"sk-live-secret", "sessionid", "hunter2", "api.openai.com", "must-not-appear", "private campaign strategy"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("benchmark report leaked %q: %s", forbidden, body)
		}
	}
}

func TestBuildUpstreamQualityBenchmarkReportMarksMissingAuditRecord(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{{
		ID:                        "image-no-audit-record",
		Type:                      "image",
		Category:                  "ecommerce_product_hero",
		Operation:                 UpstreamQualityOperationImageGeneration,
		Prompt:                    "Premium product hero image.",
		ExpectedQualityDimensions: commercialImageQualityDimensions(),
		CreatedFor:                "manual_review",
	}}
	report := BuildUpstreamQualityBenchmarkReport(samples, nil)
	result := benchmarkResultByID(report, "image-no-audit-record")
	if result.HasAuditRecord {
		t.Fatalf("expected no audit record: %+v", result)
	}
	if !benchmarkResultHasFinding(result, "missing_audit_record") {
		t.Fatalf("expected missing audit finding: %+v", result.Diagnostics)
	}
}

func TestBuildUpstreamImageQualityComparisonReportMergesReferencesScoresAndDiagnostics(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{{
		ID:                        "image-ecommerce-product-hero",
		Type:                      "image",
		Category:                  "ecommerce_product_hero",
		Operation:                 UpstreamQualityOperationImageGeneration,
		Prompt:                    "Premium product hero image.",
		ExpectedQualityDimensions: commercialImageQualityDimensions(),
		CreatedFor:                "manual_review",
	}}
	records := map[string]UpstreamQualityAuditRecord{
		"image-ecommerce-product-hero": BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/image-studio/generate",
			Operation:      UpstreamQualityOperationImageGeneration,
			RequestedModel: "gpt-image-2",
			MappedModel:    "gpt-image-2",
			UpstreamModel:  "gpt-image-2-compatible",
			ProviderName:   "image-provider-main",
			Endpoint:       "/v1/images/generations",
			ImageParams: UpstreamQualityImageParams{
				Size:  "1024x1536",
				Count: 1,
			},
			Prompt:         "Authorization: Bearer sk-live-secret premium product hero",
			PromptEnhanced: false,
			FallbackReason: "mapped_provider_model",
			LatencyMs:      3000,
			Status:         "succeeded",
		}),
	}
	report := BuildUpstreamImageQualityComparisonReport(
		samples,
		records,
		map[string]UpstreamImageQualityComparisonReference{
			"image-ecommerce-product-hero": {
				Label:   "official product baseline",
				ImageID: "official_image_001",
			},
		},
		map[string]UpstreamImageQualityComparisonReference{
			"image-ecommerce-product-hero": {
				Label:   "relay output candidate",
				ImageID: "site_image_001",
			},
		},
		map[string]UpstreamImageQualityManualScores{
			"image-ecommerce-product-hero": {
				CompositionScore:       4,
				CommercialScore:        2,
				TextQualityScore:       2,
				DetailScore:            2,
				PromptFollowingScore:   3,
				SubjectStabilityScore:  4,
				BrandConsistencyScore:  3,
				CommercialReadyScore:   2,
				ReviewerNotes:          "Commercial feel is weak; Chinese text is unstable.",
				NeedsRetry:             true,
				NeedsProviderReview:    true,
				NeedsPromptImprovement: true,
			},
		},
	)

	if report.TotalComparisons != 1 || report.MatchedAuditRecords != 1 || report.OfficialReferences != 1 || report.SiteOutputs != 1 {
		t.Fatalf("unexpected comparison counters: %+v", report)
	}
	result := imageComparisonResultByID(report, "image-ecommerce-product-hero")
	if !result.HasAuditRecord || !result.HasOfficialReference || !result.HasSiteOutput {
		t.Fatalf("expected audit/reference/site output markers: %+v", result)
	}
	if result.ImageParams.Size != "1024x1536" || result.ImageParams.Quality != "" {
		t.Fatalf("expected image params to be preserved for attribution: %+v", result.ImageParams)
	}
	for _, reason := range []UpstreamImageQualityComparisonReason{
		UpstreamImageQualityReasonModelMappingDifference,
		UpstreamImageQualityReasonFallbackOrDowngrade,
		UpstreamImageQualityReasonMissingQuality,
		UpstreamImageQualityReasonMissingStyle,
		UpstreamImageQualityReasonMissingOutputFormat,
		UpstreamImageQualityReasonPromptEnhancerMissing,
		UpstreamImageQualityReasonPoorTextQuality,
		UpstreamImageQualityReasonDistortedDetails,
		UpstreamImageQualityReasonLowCommercialAppeal,
		UpstreamImageQualityReasonNeedsRetrySelection,
		UpstreamImageQualityReasonProviderReviewNeeded,
	} {
		if !imageComparisonHasReason(result, reason) {
			t.Fatalf("expected attribution reason %s in %+v", reason, result.AttributionReasons)
		}
	}
}

func TestBuildUpstreamImageQualityComparisonReportDoesNotLeakSensitiveData(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{{
		ID:                        "image-sensitive-comparison",
		Type:                      "image",
		Category:                  "ecommerce_product_hero",
		Operation:                 UpstreamQualityOperationImageGeneration,
		Prompt:                    "Synthetic non-sensitive image benchmark.",
		ExpectedQualityDimensions: commercialImageQualityDimensions(),
		CreatedFor:                "manual_review",
	}}
	records := map[string]UpstreamQualityAuditRecord{
		"image-sensitive-comparison": BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
			Route:          "/api/v1/image-studio/generate",
			Operation:      UpstreamQualityOperationImageGeneration,
			RequestedModel: "gpt-image-2",
			UpstreamModel:  "gpt-image-2",
			ProviderName:   "image-provider-main",
			Endpoint:       "https://api.openai.com/v1/images/generations?api_key=must-not-appear",
			ImageParams: UpstreamQualityImageParams{
				Size:         "1024x1024",
				Quality:      "high",
				Style:        "commercial",
				OutputFormat: "png",
				Count:        1,
			},
			Prompt:         "Authorization: Bearer sk-live-secret cookie=sessionid password=hunter2 private customer product image prompt",
			PromptEnhanced: true,
			Status:         "succeeded",
		}),
	}
	report := BuildUpstreamImageQualityComparisonReport(
		samples,
		records,
		map[string]UpstreamImageQualityComparisonReference{
			"image-sensitive-comparison": {
				Label:   "official https://private.example.com/reference.png?token=official-secret",
				ImageID: "https://private.example.com/reference.png?token=official-secret",
			},
		},
		map[string]UpstreamImageQualityComparisonReference{
			"image-sensitive-comparison": {
				Label:   "site output https://cdn.example.com/output.png?signature=site-secret",
				ImageID: "https://cdn.example.com/output.png?signature=site-secret",
			},
		},
		map[string]UpstreamImageQualityManualScores{
			"image-sensitive-comparison": {
				ReviewerNotes: "Authorization: Bearer sk-reviewer-secret private launch prompt",
			},
		},
	)
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, forbidden := range []string{
		"sk-live-secret",
		"sessionid",
		"hunter2",
		"api.openai.com",
		"must-not-appear",
		"private customer product image prompt",
		"private.example.com",
		"official-secret",
		"cdn.example.com",
		"site-secret",
		"sk-reviewer-secret",
		"private launch prompt",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("image quality comparison report leaked %q: %s", forbidden, body)
		}
	}
	result := imageComparisonResultByID(report, "image-sensitive-comparison")
	if !strings.HasPrefix(result.OfficialReferenceImageID, "image_ref_sha256:") ||
		!strings.HasPrefix(result.SiteOutputImageID, "image_ref_sha256:") {
		t.Fatalf("expected private image references to be hashed: %+v", result)
	}
}

func TestBuildUpstreamImageQualityComparisonReportUsesOnlyImageSamples(t *testing.T) {
	t.Parallel()

	samples := []UpstreamQualityBenchmarkSample{
		{
			ID:                        "text-strict-json-output",
			Type:                      "text",
			Category:                  "strict_json_output",
			Operation:                 UpstreamQualityOperationTextCompletion,
			Prompt:                    "Return JSON.",
			ExpectedQualityDimensions: []string{"format compliance"},
			CreatedFor:                "regression",
		},
		{
			ID:                        "image-no-audit-record",
			Type:                      "image",
			Category:                  "ecommerce_product_hero",
			Operation:                 UpstreamQualityOperationImageGeneration,
			Prompt:                    "Premium product hero image.",
			ExpectedQualityDimensions: commercialImageQualityDimensions(),
			CreatedFor:                "manual_review",
		},
	}
	report := BuildUpstreamImageQualityComparisonReport(samples, nil, nil, nil, nil)
	if report.TotalComparisons != 1 {
		t.Fatalf("expected only image samples to be included: %+v", report)
	}
	result := imageComparisonResultByID(report, "image-no-audit-record")
	if result.HasAuditRecord {
		t.Fatalf("did not expect audit record: %+v", result)
	}
	if !imageComparisonResultHasFinding(result, "missing_audit_record") {
		t.Fatalf("expected missing audit record finding: %+v", result.Diagnostics)
	}
}

func newAuditTestRequest() *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/image-studio/generate", nil)
	if err != nil {
		panic(err)
	}
	return req
}

func diagnosticReportHasFinding(report UpstreamQualityDiagnosticReport, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func testContainsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func benchmarkResultByID(report UpstreamQualityBenchmarkReport, sampleID string) UpstreamQualityBenchmarkSampleResult {
	for _, result := range report.Results {
		if result.SampleID == sampleID {
			return result
		}
	}
	return UpstreamQualityBenchmarkSampleResult{}
}

func benchmarkResultHasFinding(result UpstreamQualityBenchmarkSampleResult, code string) bool {
	for _, finding := range result.Diagnostics {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func imageComparisonResultByID(report UpstreamImageQualityComparisonReport, sampleID string) UpstreamImageQualityComparisonResult {
	for _, result := range report.Results {
		if result.BenchmarkSampleID == sampleID {
			return result
		}
	}
	return UpstreamImageQualityComparisonResult{}
}

func imageComparisonHasReason(result UpstreamImageQualityComparisonResult, reason UpstreamImageQualityComparisonReason) bool {
	for _, value := range result.AttributionReasons {
		if value == reason {
			return true
		}
	}
	return false
}

func imageComparisonResultHasFinding(result UpstreamImageQualityComparisonResult, code string) bool {
	for _, finding := range result.Diagnostics {
		if finding.Code == code {
			return true
		}
	}
	return false
}
