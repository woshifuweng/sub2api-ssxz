package service

import (
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

func newAuditTestRequest() *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/image-studio/generate", nil)
	if err != nil {
		panic(err)
	}
	return req
}
