package service

import (
	"context"
	"fmt"
	"strings"
)

const (
	workspaceWebSearchRequestedKey          = "web_search_requested"
	workspaceWebSearchUsedKey               = "web_search_used"
	workspaceWebSearchStatusKey             = "web_search_status"
	workspaceWebSearchProviderKey           = "web_search_provider"
	workspaceWebSearchErrorCodeKey          = "web_search_error_code"
	workspaceWebSearchSummaryKey            = "web_search_summary"
	workspaceWebSearchCitationsKey          = "web_search_citations"
	workspaceWebSearchResultCountKey        = "web_search_result_count"
	workspaceWebSearchToolLogKey            = "web_search_tool_usage"
	workspaceWebSearchHTTPStatusKey         = "web_search_http_status"
	workspaceWebSearchResponseBodyLengthKey = "web_search_response_body_length"
	workspaceWebSearchUnavailableContent    = "联网搜索暂不可用，请稍后重试。"
	workspaceWebSearchUnavailableReason     = "web_search_unavailable"
	workspaceWebSearchSnippetMaxChars       = 240
	workspaceWebSearchSystemSummaryMaxRunes = 600
)

func (r WorkspaceSub2APITextBridgeResponder) prepareWebSearch(ctx context.Context, input WorkspaceAssistantResponseInput) ([]string, map[string]any, *WorkspaceAssistantResponse, error) {
	if !workspaceWebSearchRequested(input) {
		return nil, nil, nil, nil
	}
	if r.WebSearch == nil {
		metadata := workspaceWebSearchFailureMetadata(WorkspaceToolResult{
			ErrorCode: WorkspaceToolErrorProviderUnavailable,
			Message:   "web search is unavailable",
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:     WorkspaceToolWebSearch,
				Provider: "",
			},
		})
		return nil, metadata, workspaceWebSearchUnavailableResponse(input, metadata), nil
	}

	searchResult, err := r.WebSearch.SearchWeb(ctx, WorkspaceToolRequest{
		UserID:      input.UserID,
		Tool:        WorkspaceToolWebSearch,
		RequestedAt: input.UserMessage.CreatedAt,
		WebSearch: WebSearchRequest{
			Query: strings.TrimSpace(input.Content),
		},
	})
	if err != nil || searchResult.WebSearch == nil || len(searchResult.Citations) == 0 {
		metadata := workspaceWebSearchFailureMetadata(searchResult)
		if metadata == nil {
			metadata = workspaceWebSearchFailureMetadata(WorkspaceToolResult{
				ErrorCode: WorkspaceToolErrorProviderUnavailable,
				Message:   "web search is unavailable",
			})
		}
		return nil, metadata, workspaceWebSearchUnavailableResponse(input, metadata), nil
	}

	metadata := workspaceWebSearchSuccessMetadata(searchResult)
	systemMessage := buildWorkspaceWebSearchSystemMessage(searchResult.WebSearch.Summary, searchResult.Citations)
	if strings.TrimSpace(systemMessage) == "" {
		return nil, metadata, workspaceWebSearchUnavailableResponse(input, workspaceWebSearchFailureMetadata(WorkspaceToolResult{
			ErrorCode: WorkspaceToolErrorProviderUnavailable,
			Message:   "web search is unavailable",
		})), nil
	}
	return []string{systemMessage}, metadata, nil, nil
}

func workspaceWebSearchRequested(input WorkspaceAssistantResponseInput) bool {
	if workspaceMetadataBool(input.UserMessage.Metadata, workspaceWebSearchRequestedKey) {
		return true
	}
	return workspaceMetadataBool(input.Metadata, workspaceWebSearchRequestedKey)
}

func workspaceWebSearchUnavailableResponse(input WorkspaceAssistantResponseInput, metadata map[string]any) *WorkspaceAssistantResponse {
	response := workspaceSub2APITextBridgeBlockedResponseWithContent(input, workspaceWebSearchUnavailableReason, workspaceWebSearchUnavailableContent)
	response.Metadata = workspaceCloneMetadata(response.Metadata)
	mergeWorkspaceMetadata(response.Metadata, metadata)
	response.Metadata["provider_called"] = false
	response.Metadata["billing_touched"] = false
	response.Metadata["usage_recorded"] = false
	response.Metadata["bridge_block_reason"] = firstNonEmptyWorkspaceValue(
		workspaceMetadataString(metadata, workspaceWebSearchErrorCodeKey),
		workspaceWebSearchUnavailableReason,
	)
	return &response
}

func workspaceWebSearchSuccessMetadata(result WorkspaceToolResult) map[string]any {
	citations := make([]map[string]any, 0, len(result.Citations))
	for _, citation := range result.Citations {
		citations = append(citations, workspaceCitationMetadata(citation))
	}
	return map[string]any{
		workspaceWebSearchRequestedKey:   true,
		workspaceWebSearchUsedKey:        true,
		workspaceWebSearchStatusKey:      string(result.Status),
		workspaceWebSearchProviderKey:    strings.TrimSpace(result.UsageLog.Provider),
		workspaceWebSearchSummaryKey:     workspaceSearchSummaryText(result.WebSearch),
		workspaceWebSearchCitationsKey:   citations,
		workspaceWebSearchResultCountKey: len(citations),
		workspaceWebSearchToolLogKey: map[string]any{
			"tool":           string(result.UsageLog.Tool),
			"provider":       strings.TrimSpace(result.UsageLog.Provider),
			"status":         string(result.UsageLog.Status),
			"result_count":   result.UsageLog.ResultCount,
			"read_url_count": result.UsageLog.ReadURLCount,
			"latency_ms":     result.UsageLog.LatencyMS,
			"error_code":     strings.TrimSpace(result.UsageLog.ErrorCode),
		},
	}
}

func workspaceWebSearchFailureMetadata(result WorkspaceToolResult) map[string]any {
	provider := strings.TrimSpace(result.UsageLog.Provider)
	metadata := map[string]any{
		workspaceWebSearchRequestedKey:   true,
		workspaceWebSearchUsedKey:        false,
		workspaceWebSearchStatusKey:      firstNonEmptyWorkspaceValue(string(result.Status), string(WorkspaceToolStatusUnavailable)),
		workspaceWebSearchProviderKey:    provider,
		workspaceWebSearchErrorCodeKey:   firstNonEmptyWorkspaceValue(strings.TrimSpace(result.ErrorCode), WorkspaceToolErrorProviderUnavailable),
		workspaceWebSearchCitationsKey:   []map[string]any{},
		workspaceWebSearchResultCountKey: 0,
	}
	if result.Metadata != nil {
		if status, ok := result.Metadata["http_status"]; ok {
			metadata[workspaceWebSearchHTTPStatusKey] = status
		}
		if length, ok := result.Metadata["response_body_length"]; ok {
			metadata[workspaceWebSearchResponseBodyLengthKey] = length
		}
	}
	return metadata
}

func buildWorkspaceWebSearchSystemMessage(summary string, citations []Citation) string {
	if len(citations) == 0 {
		return ""
	}
	var builder strings.Builder
	_, _ = builder.WriteString("You have access to external web search results gathered server-side for this user request.\n")
	_, _ = builder.WriteString("Use only these results as external citations. If you rely on them, cite them inline as [1], [2], etc. Do not fabricate sources.\n")
	if trimmed := workspaceWebSearchTrim(summary, workspaceWebSearchSystemSummaryMaxRunes); trimmed != "" {
		_, _ = builder.WriteString("\nSearch summary:\n")
		_, _ = builder.WriteString(trimmed)
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("\nSources:\n")
	for _, citation := range citations {
		if citation.Index <= 0 {
			continue
		}
		_, _ = builder.WriteString(fmt.Sprintf("[%d] %s\n", citation.Index, workspaceWebSearchTrim(citation.Title, 160)))
		if domain := strings.TrimSpace(citation.Domain); domain != "" {
			_, _ = builder.WriteString("Domain: " + domain + "\n")
		}
		if rawURL := strings.TrimSpace(citation.URL); rawURL != "" {
			_, _ = builder.WriteString("URL: " + rawURL + "\n")
		}
		if snippet := workspaceWebSearchTrim(citation.Snippet, workspaceWebSearchSnippetMaxChars); snippet != "" {
			_, _ = builder.WriteString("Snippet: " + snippet + "\n")
		}
		_, _ = builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func workspaceCitationMetadata(citation Citation) map[string]any {
	return map[string]any{
		"index":   citation.Index,
		"title":   workspaceWebSearchTrim(citation.Title, 160),
		"domain":  strings.TrimSpace(citation.Domain),
		"url":     strings.TrimSpace(citation.URL),
		"snippet": workspaceWebSearchTrim(citation.Snippet, workspaceWebSearchSnippetMaxChars),
	}
}

func workspaceSearchSummaryText(result *WebSearchResult) string {
	if result == nil {
		return ""
	}
	return workspaceWebSearchTrim(result.Summary, workspaceWebSearchSystemSummaryMaxRunes)
}

func workspaceWebSearchTrim(value string, maxRunes int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxRunes]))
}

func workspaceCloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
