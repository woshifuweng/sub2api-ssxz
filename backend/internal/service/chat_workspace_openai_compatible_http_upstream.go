package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	workspaceOpenAICompatibleHTTPPath             = "/v1/chat/completions"
	workspaceOpenAICompatibleHTTPStatusSucceeded  = "succeeded"
	workspaceOpenAICompatibleHTTPErrorUnavailable = "workspace_openai_compatible_http_unavailable"
	workspaceOpenAICompatibleHTTPErrorUpstream    = "workspace_openai_compatible_http_upstream_error"
	workspaceOpenAICompatibleHTTPErrorTimeout     = "workspace_openai_compatible_http_timeout"
	workspaceOpenAICompatibleHTTPErrorDecode      = "workspace_openai_compatible_http_decode_error"
)

var (
	errWorkspaceOpenAICompatibleHTTPUnavailable = errors.New("workspace openai-compatible http upstream unavailable")
	errWorkspaceOpenAICompatibleHTTPFailed      = errors.New("workspace openai-compatible http upstream failed safely")
	errWorkspaceOpenAICompatibleHTTPTimeout     = errors.New("workspace openai-compatible http upstream timed out safely")
)

type WorkspaceOpenAICompatibleHTTPUpstream struct {
	BaseURL        string
	Model          string
	APIKey         string
	ProviderLabel  string
	EndpointLabel  string
	Timeout        time.Duration
	HTTPClient     *http.Client
	allowTestLocal bool
}

func NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceOpenAICompatibleTextUpstream {
	return newWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision, false)
}

func newWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg *config.Config, decision WorkspaceTextProviderGateDecision, allowTestLocal bool) WorkspaceOpenAICompatibleTextUpstream {
	if cfg == nil || !decision.Enabled {
		return nil
	}
	upstreamCfg := cfg.Workspace.TextProvider.OpenAICompatible
	baseURL, err := normalizeWorkspaceOpenAICompatibleBaseURL(upstreamCfg.BaseURL, allowTestLocal)
	if err != nil {
		return nil
	}
	model := strings.TrimSpace(upstreamCfg.Model)
	apiKey := strings.TrimSpace(upstreamCfg.APIKey)
	if model == "" || apiKey == "" {
		return nil
	}
	timeout := time.Duration(upstreamCfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		return nil
	}
	return WorkspaceOpenAICompatibleHTTPUpstream{
		BaseURL:        baseURL,
		Model:          model,
		APIKey:         apiKey,
		ProviderLabel:  decision.TestProviderLabel,
		EndpointLabel:  firstNonEmptyWorkspaceValue(decision.TestProviderLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel),
		Timeout:        timeout,
		allowTestLocal: allowTestLocal,
	}
}

func (u WorkspaceOpenAICompatibleHTTPUpstream) ExecuteWorkspaceOpenAICompatibleText(ctx context.Context, req WorkspaceOpenAICompatibleExecutionRequest) (WorkspaceOpenAICompatibleExecutionResponse, error) {
	baseURL, err := normalizeWorkspaceOpenAICompatibleBaseURL(u.BaseURL, u.allowTestLocal)
	if err != nil {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUnavailable, 0), errWorkspaceOpenAICompatibleHTTPUnavailable
	}
	model := firstNonEmptyWorkspaceValue(req.UpstreamModel, req.MappedModel, u.Model)
	if model == "" || strings.TrimSpace(u.APIKey) == "" {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUnavailable, 0), errWorkspaceOpenAICompatibleHTTPUnavailable
	}
	timeout := u.Timeout
	if timeout <= 0 {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUnavailable, 0), errWorkspaceOpenAICompatibleHTTPUnavailable
	}

	payload := workspaceOpenAICompatibleChatCompletionRequest{
		Model:    model,
		Messages: workspaceOpenAICompatibleHTTPMessages(req.Messages),
		Stream:   false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUnavailable, 0), errWorkspaceOpenAICompatibleHTTPUnavailable
	}

	startedAt := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(callCtx, http.MethodPost, baseURL+workspaceOpenAICompatibleHTTPPath, bytes.NewReader(body))
	if err != nil {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUnavailable, 0), errWorkspaceOpenAICompatibleHTTPUnavailable
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(u.APIKey))

	client := u.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	httpResp, err := client.Do(httpReq)
	latencyMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorTimeout, latencyMs), errWorkspaceOpenAICompatibleHTTPTimeout
		}
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUpstream, latencyMs), errWorkspaceOpenAICompatibleHTTPFailed
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResp.Body, 4096))
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorUpstream, latencyMs), errWorkspaceOpenAICompatibleHTTPFailed
	}

	var decoded workspaceOpenAICompatibleChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(httpResp.Body, 1<<20)).Decode(&decoded); err != nil {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorDecode, latencyMs), errWorkspaceOpenAICompatibleHTTPFailed
	}
	content := strings.TrimSpace(workspaceOpenAICompatibleHTTPContent(decoded))
	if content == "" {
		return workspaceOpenAICompatibleHTTPErrorResponse(req, u, workspaceOpenAICompatibleHTTPErrorDecode, latencyMs), errWorkspaceOpenAICompatibleHTTPFailed
	}

	return WorkspaceOpenAICompatibleExecutionResponse{
		Content:        content,
		Status:         workspaceOpenAICompatibleHTTPStatusSucceeded,
		RequestedModel: req.RequestedModel,
		MappedModel:    firstNonEmptyWorkspaceValue(req.MappedModel, model),
		UpstreamModel:  firstNonEmptyWorkspaceValue(decoded.Model, req.UpstreamModel, model),
		ProviderLabel:  firstNonEmptyWorkspaceValue(req.ProviderLabel, u.ProviderLabel),
		ProviderName:   firstNonEmptyWorkspaceValue(req.ProviderLabel, u.ProviderLabel, WorkspaceOpenAICompatibleTextExecutorProviderName),
		EndpointLabel:  firstNonEmptyWorkspaceValue(u.EndpointLabel, req.EndpointLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel),
		ServiceTier:    req.ServiceTier,
		FallbackUsed:   false,
		LatencyMs:      latencyMs,
		Usage:          workspaceOpenAICompatibleHTTPUsage(decoded.Usage),
	}, nil
}

type workspaceOpenAICompatibleChatCompletionRequest struct {
	Model    string                                           `json:"model"`
	Messages []workspaceOpenAICompatibleChatCompletionMessage `json:"messages"`
	Stream   bool                                             `json:"stream"`
}

type workspaceOpenAICompatibleChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type workspaceOpenAICompatibleChatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message workspaceOpenAICompatibleChatCompletionMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func workspaceOpenAICompatibleHTTPMessages(messages []WorkspaceOpenAICompatibleTextMessage) []workspaceOpenAICompatibleChatCompletionMessage {
	out := make([]workspaceOpenAICompatibleChatCompletionMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = WorkspaceRoleUser
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		out = append(out, workspaceOpenAICompatibleChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}
	return out
}

func workspaceOpenAICompatibleHTTPContent(resp workspaceOpenAICompatibleChatCompletionResponse) string {
	for _, choice := range resp.Choices {
		if strings.TrimSpace(choice.Message.Content) != "" {
			return strings.TrimSpace(choice.Message.Content)
		}
	}
	return ""
}

func workspaceOpenAICompatibleHTTPUsage(usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}) UpstreamQualityUsage {
	total := usage.TotalTokens
	if total <= 0 {
		total = usage.PromptTokens + usage.CompletionTokens
	}
	return UpstreamQualityUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  total,
	}
}

func workspaceOpenAICompatibleHTTPErrorResponse(req WorkspaceOpenAICompatibleExecutionRequest, upstream WorkspaceOpenAICompatibleHTTPUpstream, errorCode string, latencyMs int64) WorkspaceOpenAICompatibleExecutionResponse {
	return WorkspaceOpenAICompatibleExecutionResponse{
		Status:         "failed",
		RequestedModel: req.RequestedModel,
		MappedModel:    firstNonEmptyWorkspaceValue(req.MappedModel, upstream.Model),
		UpstreamModel:  firstNonEmptyWorkspaceValue(req.UpstreamModel, upstream.Model),
		ProviderLabel:  firstNonEmptyWorkspaceValue(req.ProviderLabel, upstream.ProviderLabel),
		ProviderName:   firstNonEmptyWorkspaceValue(req.ProviderLabel, upstream.ProviderLabel, WorkspaceOpenAICompatibleTextExecutorProviderName),
		EndpointLabel:  firstNonEmptyWorkspaceValue(upstream.EndpointLabel, req.EndpointLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel),
		ServiceTier:    req.ServiceTier,
		LatencyMs:      latencyMs,
		ErrorCode:      errorCode,
	}
}

func normalizeWorkspaceOpenAICompatibleBaseURL(raw string, allowTestLocal bool) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("missing base url")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid base url")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("unsafe base url")
	}
	if parsed.Scheme != "https" {
		if !(allowTestLocal && parsed.Scheme == "http") {
			return "", fmt.Errorf("unsupported base url scheme")
		}
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("missing base url host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("base url path is not supported")
	}
	host := parsed.Hostname()
	if isUnsafeWorkspaceOpenAICompatibleHost(host) && !allowTestLocal {
		return "", fmt.Errorf("unsafe base url host")
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isUnsafeWorkspaceOpenAICompatibleHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
