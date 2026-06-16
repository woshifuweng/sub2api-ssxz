package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	defaultJinaSearchBaseURL = "https://s.jina.ai"
	defaultJinaReaderBaseURL = "https://r.jina.ai"
	defaultJinaTimeout       = 8 * time.Second
	defaultCitationSnippet   = 320
)

var jinaMarkdownLinkPattern = regexp.MustCompile(`\[(?P<title>[^\]]+)\]\((?P<url>https?://[^)\s]+)\)`)

type jinaHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type jinaSearchHit struct {
	Title   string
	URL     string
	Snippet string
}

type jinaReaderDocument struct {
	Title   string
	URL     string
	Snippet string
}

type JinaWebSearchAdapter struct {
	apiKey                string
	searchBaseURL         string
	readerBaseURL         string
	maxContentLengthBytes int64
	httpClient            jinaHTTPDoer
	now                   func() time.Time
}

type jinaAdapterError struct {
	Code               string
	Message            string
	HTTPStatus         int
	ResponseBodyLength int
	Err                error
}

func (e *jinaAdapterError) Error() string {
	if e == nil {
		return "jina adapter error"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "jina adapter error"
}

func (e *jinaAdapterError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *jinaAdapterError) Metadata() map[string]any {
	if e == nil {
		return nil
	}
	metadata := map[string]any{}
	if e.HTTPStatus > 0 {
		metadata["http_status"] = e.HTTPStatus
	}
	if e.ResponseBodyLength > 0 {
		metadata["response_body_length"] = e.ResponseBodyLength
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func ProvideWorkspaceWebSearchTool(cfg *config.Config) WebSearchTool {
	if cfg == nil {
		return nil
	}
	webCfg := cfg.Workspace.WebSearch
	if strings.ToLower(strings.TrimSpace(webCfg.Provider)) != "jina" {
		return nil
	}
	if strings.TrimSpace(webCfg.APIKey) == "" {
		return nil
	}

	timeout := time.Duration(webCfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultJinaTimeout
	}
	return NewJinaWebSearchAdapter(webCfg, &http.Client{Timeout: timeout})
}

func NewJinaWebSearchAdapter(cfg config.WorkspaceWebSearchConfig, client jinaHTTPDoer) *JinaWebSearchAdapter {
	if client == nil {
		timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = defaultJinaTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	maxLen := cfg.MaxContentLengthBytes
	if maxLen <= 0 {
		maxLen = 1024 * 1024
	}
	return &JinaWebSearchAdapter{
		apiKey:                strings.TrimSpace(cfg.APIKey),
		searchBaseURL:         strings.TrimRight(defaultJinaSearchBaseURL, "/"),
		readerBaseURL:         strings.TrimRight(defaultJinaReaderBaseURL, "/"),
		maxContentLengthBytes: maxLen,
		httpClient:            client,
		now:                   time.Now,
	}
}

func (a *JinaWebSearchAdapter) Search(ctx context.Context, req WebSearchRequest) (WebSearchResult, error) {
	if a == nil || a.httpClient == nil || a.apiKey == "" {
		return WebSearchResult{}, &jinaAdapterError{Code: WorkspaceToolErrorProviderUnavailable, Message: "jina web search unavailable"}
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return WebSearchResult{}, &jinaAdapterError{Code: WorkspaceToolErrorProviderUnavailable, Message: "jina web search unavailable"}
	}

	timeout := time.Duration(req.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultJinaTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	hits, err := a.searchHits(ctx, query)
	if err != nil {
		return WebSearchResult{}, err
	}

	citations := make([]Citation, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		if hit.URL == "" {
			continue
		}
		if err := ValidateWorkspaceWebSearchURL(hit.URL); err != nil {
			continue
		}
		if _, ok := seen[hit.URL]; ok {
			continue
		}
		seen[hit.URL] = struct{}{}
		citations = append(citations, Citation{
			Title:       normalizeSnippet(hit.Title, 160),
			URL:         hit.URL,
			Domain:      citationDomain(hit.URL),
			Snippet:     normalizeSnippet(hit.Snippet, defaultCitationSnippet),
			RetrievedAt: a.now().UTC(),
		})
		if req.MaxResults > 0 && len(citations) >= req.MaxResults {
			break
		}
	}

	readTargets := make([]string, 0, req.MaxReadURLs)
	for _, citation := range citations {
		readTargets = append(readTargets, citation.URL)
		if req.MaxReadURLs > 0 && len(readTargets) >= req.MaxReadURLs {
			break
		}
	}
	for _, rawURL := range req.ReadURLs {
		if req.MaxReadURLs > 0 && len(readTargets) >= req.MaxReadURLs {
			break
		}
		if _, ok := seen[rawURL]; ok {
			continue
		}
		if err := ValidateWorkspaceWebSearchURL(rawURL); err != nil {
			continue
		}
		readTargets = append(readTargets, rawURL)
		seen[rawURL] = struct{}{}
	}

	enriched := make(map[string]jinaReaderDocument, len(readTargets))
	for _, rawURL := range readTargets {
		doc, err := a.readDocument(ctx, rawURL)
		if err != nil {
			continue
		}
		enriched[rawURL] = doc
	}

	for i := range citations {
		if doc, ok := enriched[citations[i].URL]; ok {
			if doc.Title != "" {
				citations[i].Title = normalizeSnippet(doc.Title, 160)
			}
			if doc.Snippet != "" {
				citations[i].Snippet = normalizeSnippet(doc.Snippet, defaultCitationSnippet)
			}
		}
		citations[i].Index = i + 1
	}

	return WebSearchResult{
		Query:     query,
		Summary:   buildCitationSummary(citations),
		Citations: citations,
	}, nil
}

func (a *JinaWebSearchAdapter) searchHits(ctx context.Context, query string) ([]jinaSearchHit, error) {
	endpoint := a.searchBaseURL + "/?q=" + url.QueryEscape(query)
	body, err := a.doTextRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	hits, parseCode := parseJinaSearchHits(body)
	if len(hits) == 0 {
		return nil, &jinaAdapterError{
			Code:    parseCode,
			Message: "jina search unavailable",
		}
	}
	return hits, nil
}

func (a *JinaWebSearchAdapter) readDocument(ctx context.Context, targetURL string) (jinaReaderDocument, error) {
	body, err := a.doTextRequest(ctx, a.readerBaseURL+"/"+targetURL)
	if err != nil {
		return jinaReaderDocument{}, fmt.Errorf("jina reader unavailable")
	}
	doc := parseJinaReaderDocument(body)
	if doc.URL == "" {
		doc.URL = targetURL
	}
	if doc.Snippet == "" && doc.Title == "" {
		return jinaReaderDocument{}, fmt.Errorf("jina reader unavailable")
	}
	return doc, nil
}

func (a *JinaWebSearchAdapter) doTextRequest(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", &jinaAdapterError{
			Code:    WorkspaceToolErrorRequestBuildFailed,
			Message: "request build failed",
			Err:     err,
		}
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("User-Agent", "sub2api-workspace-web-search/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", classifyJinaHTTPError(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited := io.LimitReader(resp.Body, a.maxContentLengthBytes+1)
		body, readErr := io.ReadAll(limited)
		if readErr != nil {
			return "", &jinaAdapterError{
				Code:       WorkspaceToolErrorUpstreamNon2xx,
				Message:    "request failed",
				HTTPStatus: resp.StatusCode,
			}
		}
		bodyLen := len(body)
		if int64(bodyLen) > a.maxContentLengthBytes {
			bodyLen = int(a.maxContentLengthBytes)
		}
		return "", &jinaAdapterError{
			Code:               WorkspaceToolErrorUpstreamNon2xx,
			Message:            "request failed",
			HTTPStatus:         resp.StatusCode,
			ResponseBodyLength: bodyLen,
		}
	}

	limited := io.LimitReader(resp.Body, a.maxContentLengthBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		if isJinaTimeoutError(err) || isJinaTimeoutError(ctx.Err()) {
			return "", &jinaAdapterError{
				Code:    WorkspaceToolErrorTimeout,
				Message: "request failed",
				Err:     err,
			}
		}
		return "", &jinaAdapterError{
			Code:    WorkspaceToolErrorResponseReadFailed,
			Message: "response read failed",
			Err:     err,
		}
	}
	if int64(len(body)) > a.maxContentLengthBytes {
		return "", &jinaAdapterError{
			Code:               WorkspaceToolErrorResponseTooLarge,
			Message:            "response too large",
			ResponseBodyLength: len(body),
		}
	}
	return string(body), nil
}

func parseJinaSearchHits(body string) ([]jinaSearchHit, string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, WorkspaceToolErrorEmptySearchHits
	}

	type jinaResultEnvelope struct {
		Results []map[string]any `json:"results"`
		Data    []map[string]any `json:"data"`
		Items   []map[string]any `json:"items"`
	}
	var envelope jinaResultEnvelope
	if json.Unmarshal([]byte(body), &envelope) == nil {
		if hits := normalizeJinaSearchMaps(envelope.Results); len(hits) > 0 {
			return hits, ""
		}
		if hits := normalizeJinaSearchMaps(envelope.Data); len(hits) > 0 {
			return hits, ""
		}
		if hits := normalizeJinaSearchMaps(envelope.Items); len(hits) > 0 {
			return hits, ""
		}
		return nil, WorkspaceToolErrorEmptySearchHits
	}

	var rawList []map[string]any
	if json.Unmarshal([]byte(body), &rawList) == nil {
		if hits := normalizeJinaSearchMaps(rawList); len(hits) > 0 {
			return hits, ""
		}
		return nil, WorkspaceToolErrorEmptySearchHits
	}

	if hits := parseJinaSearchMarkdown(body); len(hits) > 0 {
		return hits, ""
	}
	return nil, WorkspaceToolErrorBodyParseFailed
}

func classifyJinaHTTPError(err error) error {
	if isJinaTimeoutError(err) {
		return &jinaAdapterError{
			Code:    WorkspaceToolErrorTimeout,
			Message: "request failed",
			Err:     err,
		}
	}
	return &jinaAdapterError{
		Code:    WorkspaceToolErrorHTTPTransportFailed,
		Message: "request failed",
		Err:     err,
	}
}

func isJinaTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func normalizeJinaSearchMaps(items []map[string]any) []jinaSearchHit {
	hits := make([]jinaSearchHit, 0, len(items))
	for _, item := range items {
		urlValue := normalizeSnippet(stringValue(item["url"]), 2048)
		if urlValue == "" {
			urlValue = normalizeSnippet(stringValue(item["link"]), 2048)
		}
		if urlValue == "" {
			continue
		}
		hits = append(hits, jinaSearchHit{
			Title: normalizeSnippet(firstNonEmptyWebSearchString(
				stringValue(item["title"]),
				stringValue(item["name"]),
			), 160),
			URL: urlValue,
			Snippet: normalizeSnippet(firstNonEmptyWebSearchString(
				stringValue(item["snippet"]),
				stringValue(item["description"]),
				stringValue(item["content"]),
				stringValue(item["text"]),
			), defaultCitationSnippet),
		})
	}
	return hits
}

func parseJinaSearchMarkdown(body string) []jinaSearchHit {
	lines := strings.Split(body, "\n")
	hits := make([]jinaSearchHit, 0)
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		matches := jinaMarkdownLinkPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		title := normalizeSnippet(matches[1], 160)
		rawURL := normalizeSnippet(matches[2], 2048)
		snippet := ""
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				break
			}
			if jinaMarkdownLinkPattern.MatchString(next) {
				break
			}
			if snippet != "" {
				snippet += " "
			}
			snippet += next
		}
		hits = append(hits, jinaSearchHit{
			Title:   title,
			URL:     rawURL,
			Snippet: normalizeSnippet(snippet, defaultCitationSnippet),
		})
	}
	return hits
}

func parseJinaReaderDocument(body string) jinaReaderDocument {
	body = strings.TrimSpace(body)
	if body == "" {
		return jinaReaderDocument{}
	}

	type readerJSON struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
		Text    string `json:"text"`
		Excerpt string `json:"excerpt"`
	}
	var doc readerJSON
	if json.Unmarshal([]byte(body), &doc) == nil {
		return jinaReaderDocument{
			Title:   normalizeSnippet(doc.Title, 160),
			URL:     normalizeSnippet(doc.URL, 2048),
			Snippet: normalizeSnippet(firstNonEmptyWebSearchString(doc.Excerpt, doc.Text, doc.Content), defaultCitationSnippet),
		}
	}

	title := ""
	snippet := ""
	for _, rawLine := range strings.Split(body, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if title == "" {
			title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		snippet = line
		break
	}

	if snippet == "" {
		snippet = body
	}

	return jinaReaderDocument{
		Title:   normalizeSnippet(title, 160),
		Snippet: normalizeSnippet(snippet, defaultCitationSnippet),
	}
}

func buildCitationSummary(citations []Citation) string {
	if len(citations) == 0 {
		return ""
	}
	parts := make([]string, 0, minInt(len(citations), 3))
	for i, citation := range citations {
		if i >= 3 {
			break
		}
		title := strings.TrimSpace(citation.Title)
		snippet := strings.TrimSpace(citation.Snippet)
		switch {
		case title != "" && snippet != "":
			parts = append(parts, title+": "+snippet)
		case title != "":
			parts = append(parts, title)
		case snippet != "":
			parts = append(parts, snippet)
		}
	}
	return strings.Join(parts, "\n")
}

func citationDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func normalizeSnippet(value string, maxLen int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if maxLen > 0 && len(value) > maxLen {
		return strings.TrimSpace(value[:maxLen]) + "..."
	}
	return value
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func firstNonEmptyWebSearchString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
