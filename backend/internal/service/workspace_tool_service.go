package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type WorkspaceToolName string
type WorkspaceToolStatus string

const (
	WorkspaceToolWebSearch WorkspaceToolName = "web_search"

	WorkspaceToolStatusCompleted   WorkspaceToolStatus = "completed"
	WorkspaceToolStatusUnavailable WorkspaceToolStatus = "unavailable"
	WorkspaceToolStatusBlocked     WorkspaceToolStatus = "blocked"
	WorkspaceToolStatusFailed      WorkspaceToolStatus = "failed"

	WorkspaceToolErrorDisabled            = "web_search_disabled"
	WorkspaceToolErrorKillSwitch          = "web_search_kill_switch"
	WorkspaceToolErrorUserNotAllowed      = "web_search_user_not_allowed"
	WorkspaceToolErrorCapExceeded         = "web_search_cap_exceeded"
	WorkspaceToolErrorInvalidURL          = "web_search_invalid_url"
	WorkspaceToolErrorProviderUnavailable = "web_search_provider_unavailable"
	WorkspaceToolErrorRequestBuildFailed  = "request_build_failed"
	WorkspaceToolErrorHTTPTransportFailed = "http_transport_failed"
	WorkspaceToolErrorUpstreamNon2xx      = "upstream_non_2xx"
	WorkspaceToolErrorResponseReadFailed  = "response_read_failed"
	WorkspaceToolErrorBodyParseFailed     = "body_parse_failed"
	WorkspaceToolErrorEmptySearchHits     = "empty_search_hits"
	WorkspaceToolErrorTimeout             = "timeout"
	WorkspaceToolErrorResponseTooLarge    = "response_too_large"
)

var ErrWorkspaceToolUnavailable = errors.New("workspace tool unavailable")

type WorkspaceToolRequest struct {
	UserID          int64
	Tool            WorkspaceToolName
	WebSearch       WebSearchRequest
	UsageCountToday int
	RequestedAt     time.Time
}

type WorkspaceToolResult struct {
	Tool      WorkspaceToolName
	Status    WorkspaceToolStatus
	ErrorCode string
	Message   string
	WebSearch *WebSearchResult
	Citations []Citation
	UsageLog  WorkspaceToolUsageLogPayload
	Metadata  map[string]any
}

type WebSearchRequest struct {
	Query       string
	ReadURLs    []string
	MaxResults  int
	MaxReadURLs int
	TimeoutMS   int
}

type WebSearchResult struct {
	Query     string
	Summary   string
	Citations []Citation
}

type Citation struct {
	Index       int
	Title       string
	URL         string
	Domain      string
	Snippet     string
	RetrievedAt time.Time
}

type WorkspaceToolUsageLogPayload struct {
	UserID       int64
	Tool         WorkspaceToolName
	Provider     string
	Status       WorkspaceToolStatus
	ResultCount  int
	ReadURLCount int
	LatencyMS    int64
	ErrorCode    string
	CreatedAt    time.Time
}

type WorkspaceToolAvailability struct {
	Available bool   `json:"available"`
	Provider  string `json:"provider"`
	Reason    string `json:"reason,omitempty"`
}

type WebSearchTool interface {
	Search(ctx context.Context, req WebSearchRequest) (WebSearchResult, error)
}

type WorkspaceToolService struct {
	cfg       config.WorkspaceWebSearchConfig
	webSearch WebSearchTool
}

func NewWorkspaceToolService(cfg *config.Config, webSearch WebSearchTool) *WorkspaceToolService {
	var webSearchConfig config.WorkspaceWebSearchConfig
	if cfg != nil {
		webSearchConfig = cfg.Workspace.WebSearch
	}
	return &WorkspaceToolService{
		cfg:       webSearchConfig,
		webSearch: webSearch,
	}
}

func (s *WorkspaceToolService) WebSearchAvailability(userID int64) WorkspaceToolAvailability {
	if s == nil {
		return WorkspaceToolAvailability{Available: false, Reason: WorkspaceToolErrorDisabled}
	}
	if !s.cfg.Enabled {
		return WorkspaceToolAvailability{Available: false, Provider: s.cfg.Provider, Reason: WorkspaceToolErrorDisabled}
	}
	if s.cfg.KillSwitch {
		return WorkspaceToolAvailability{Available: false, Provider: s.cfg.Provider, Reason: WorkspaceToolErrorKillSwitch}
	}
	if !workspaceToolUserAllowed(userID, s.cfg.AllowedUserIDs) {
		return WorkspaceToolAvailability{Available: false, Provider: s.cfg.Provider, Reason: WorkspaceToolErrorUserNotAllowed}
	}
	if s.webSearch == nil {
		return WorkspaceToolAvailability{Available: false, Provider: s.cfg.Provider, Reason: WorkspaceToolErrorProviderUnavailable}
	}
	return WorkspaceToolAvailability{Available: true, Provider: s.cfg.Provider}
}

func (s *WorkspaceToolService) SearchWeb(ctx context.Context, req WorkspaceToolRequest) (WorkspaceToolResult, error) {
	start := time.Now()
	if req.Tool == "" {
		req.Tool = WorkspaceToolWebSearch
	}
	if req.RequestedAt.IsZero() {
		req.RequestedAt = start
	}

	if result, ok := s.blockedResult(req, start); ok {
		return result, ErrWorkspaceToolUnavailable
	}

	for _, rawURL := range req.WebSearch.ReadURLs {
		if err := ValidateWorkspaceWebSearchURL(rawURL); err != nil {
			return s.failureResult(req, start, WorkspaceToolErrorInvalidURL, err.Error(), nil), ErrWorkspaceToolUnavailable
		}
	}

	webReq := req.WebSearch
	if webReq.MaxResults <= 0 {
		webReq.MaxResults = s.cfg.MaxResults
	}
	if webReq.MaxReadURLs <= 0 {
		webReq.MaxReadURLs = s.cfg.MaxReadURLs
	}
	if webReq.TimeoutMS <= 0 {
		webReq.TimeoutMS = s.cfg.TimeoutMS
	}

	result, err := s.webSearch.Search(ctx, webReq)
	if err != nil {
		code, metadata := classifyWorkspaceWebSearchProviderError(err)
		return s.failureResult(req, start, code, "web search provider unavailable", metadata), err
	}

	usage := s.usageLog(req, start, WorkspaceToolStatusCompleted, "")
	usage.ResultCount = len(result.Citations)
	usage.ReadURLCount = len(webReq.ReadURLs)
	return WorkspaceToolResult{
		Tool:      WorkspaceToolWebSearch,
		Status:    WorkspaceToolStatusCompleted,
		WebSearch: &result,
		Citations: result.Citations,
		UsageLog:  usage,
	}, nil
}

func (s *WorkspaceToolService) blockedResult(req WorkspaceToolRequest, start time.Time) (WorkspaceToolResult, bool) {
	if s == nil {
		return WorkspaceToolResult{
			Tool:      WorkspaceToolWebSearch,
			Status:    WorkspaceToolStatusUnavailable,
			ErrorCode: WorkspaceToolErrorDisabled,
			Message:   "Web search is unavailable.",
		}, true
	}
	switch {
	case !s.cfg.Enabled:
		return s.failureResult(req, start, WorkspaceToolErrorDisabled, "Web search is unavailable.", nil), true
	case s.cfg.KillSwitch:
		return s.failureResult(req, start, WorkspaceToolErrorKillSwitch, "Web search is unavailable.", nil), true
	case !workspaceToolUserAllowed(req.UserID, s.cfg.AllowedUserIDs):
		return s.failureResult(req, start, WorkspaceToolErrorUserNotAllowed, "Web search is not enabled for this account.", nil), true
	case s.cfg.DailyCapPerUser <= 0 || req.UsageCountToday >= s.cfg.DailyCapPerUser:
		return s.failureResult(req, start, WorkspaceToolErrorCapExceeded, "Web search daily limit reached.", nil), true
	case s.webSearch == nil:
		return s.failureResult(req, start, WorkspaceToolErrorProviderUnavailable, "Web search is unavailable.", nil), true
	default:
		return WorkspaceToolResult{}, false
	}
}

func (s *WorkspaceToolService) failureResult(req WorkspaceToolRequest, start time.Time, code, message string, metadata map[string]any) WorkspaceToolResult {
	status := WorkspaceToolStatusBlocked
	if isWorkspaceToolUnavailableCode(code) {
		status = WorkspaceToolStatusUnavailable
	}
	return WorkspaceToolResult{
		Tool:      WorkspaceToolWebSearch,
		Status:    status,
		ErrorCode: code,
		Message:   message,
		UsageLog:  s.usageLog(req, start, status, code),
		Metadata:  cloneWorkspaceToolMetadata(metadata),
	}
}

func isWorkspaceToolUnavailableCode(code string) bool {
	switch code {
	case WorkspaceToolErrorProviderUnavailable,
		WorkspaceToolErrorDisabled,
		WorkspaceToolErrorKillSwitch,
		WorkspaceToolErrorRequestBuildFailed,
		WorkspaceToolErrorHTTPTransportFailed,
		WorkspaceToolErrorUpstreamNon2xx,
		WorkspaceToolErrorResponseReadFailed,
		WorkspaceToolErrorBodyParseFailed,
		WorkspaceToolErrorEmptySearchHits,
		WorkspaceToolErrorTimeout,
		WorkspaceToolErrorResponseTooLarge:
		return true
	default:
		return false
	}
}

func (s *WorkspaceToolService) usageLog(req WorkspaceToolRequest, start time.Time, status WorkspaceToolStatus, code string) WorkspaceToolUsageLogPayload {
	provider := ""
	if s != nil {
		provider = s.cfg.Provider
	}
	return WorkspaceToolUsageLogPayload{
		UserID:    req.UserID,
		Tool:      WorkspaceToolWebSearch,
		Provider:  provider,
		Status:    status,
		ErrorCode: code,
		LatencyMS: time.Since(start).Milliseconds(),
		CreatedAt: time.Now().UTC(),
	}
}

func classifyWorkspaceWebSearchProviderError(err error) (string, map[string]any) {
	var adapterErr *jinaAdapterError
	if errors.As(err, &adapterErr) {
		return adapterErr.Code, adapterErr.Metadata()
	}
	return WorkspaceToolErrorProviderUnavailable, nil
}

func cloneWorkspaceToolMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func workspaceToolUserAllowed(userID int64, allowed []int64) bool {
	if userID <= 0 || len(allowed) == 0 {
		return false
	}
	for _, allowedID := range allowed {
		if userID == allowedID {
			return true
		}
	}
	return false
}

func ValidateWorkspaceWebSearchURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url")
	}
	if !parsed.IsAbs() {
		return fmt.Errorf("url must be absolute")
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("unsupported url scheme")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("missing url host")
	}
	if isWorkspaceWebSearchBlockedHost(host) {
		return fmt.Errorf("blocked internal url host")
	}
	return nil
}

func isWorkspaceWebSearchBlockedHost(host string) bool {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if normalized == "" {
		return true
	}
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return true
	}
	if normalized == "metadata.google.internal" || strings.HasSuffix(normalized, ".metadata.google.internal") {
		return true
	}
	if strings.HasSuffix(normalized, ".local") || strings.HasSuffix(normalized, ".lan") || strings.HasSuffix(normalized, ".internal") {
		return true
	}
	if ip := net.ParseIP(normalized); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		return ok && isWorkspaceWebSearchBlockedIP(addr.Unmap())
	}
	if addr, err := netip.ParseAddr(normalized); err == nil {
		return isWorkspaceWebSearchBlockedIP(addr.Unmap())
	}
	return false
}

func isWorkspaceWebSearchBlockedIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
		return true
	}
	blockedPrefixes := []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("169.254.169.254/32"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("::/128"),
		netip.MustParsePrefix("fc00::/7"),
		netip.MustParsePrefix("fe80::/10"),
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
