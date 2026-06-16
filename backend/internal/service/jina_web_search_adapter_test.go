package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type fakeJinaHTTPDoer struct {
	requests []*http.Request
	err      error
	handlers map[string]fakeJinaHTTPResponse
}

type fakeJinaHTTPResponse struct {
	status int
	body   string
}

func (f *fakeJinaHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	key := req.URL.String()
	handler, ok := f.handlers[key]
	if !ok {
		return nil, fmt.Errorf("unexpected request %s", key)
	}
	status := handler.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(handler.body)),
		Header:     make(http.Header),
	}, nil
}

func TestProvideWorkspaceWebSearchToolRequiresAPIKey(t *testing.T) {
	cfg := &config.Config{
		Workspace: config.WorkspaceConfig{
			WebSearch: config.WorkspaceWebSearchConfig{
				Provider: "jina",
			},
		},
	}
	require.Nil(t, ProvideWorkspaceWebSearchTool(cfg))

	cfg.Workspace.WebSearch.APIKey = "test-key"
	require.NotNil(t, ProvideWorkspaceWebSearchTool(cfg))
}

func TestJinaWebSearchAdapterSearchSuccessParsesSearchAndReaderIntoCitations(t *testing.T) {
	doer := &fakeJinaHTTPDoer{
		handlers: map[string]fakeJinaHTTPResponse{
			"https://s.jina.ai/?q=latest+weather": {
				body: "[Example Result](https://example.com/article)\nA search snippet from Jina Search.\n",
			},
			"https://r.jina.ai/https://example.com/article": {
				body: "# Example Result\nA reader snippet from Jina Reader with more context.\n",
			},
		},
	}
	adapter := NewJinaWebSearchAdapter(config.WorkspaceWebSearchConfig{
		APIKey:                "test-key",
		MaxContentLengthBytes: 2048,
	}, doer)
	adapter.now = func() time.Time { return time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC) }

	result, err := adapter.Search(context.Background(), WebSearchRequest{
		Query:       "latest weather",
		MaxResults:  5,
		MaxReadURLs: 1,
		TimeoutMS:   1000,
	})

	require.NoError(t, err)
	require.Len(t, result.Citations, 1)
	require.Equal(t, "latest weather", result.Query)
	require.Equal(t, "Example Result", result.Citations[0].Title)
	require.Equal(t, "https://example.com/article", result.Citations[0].URL)
	require.Equal(t, "example.com", result.Citations[0].Domain)
	require.Contains(t, result.Citations[0].Snippet, "reader snippet")
	require.Contains(t, result.Summary, "Example Result")
	require.Len(t, doer.requests, 2)
	require.Equal(t, "Bearer test-key", doer.requests[0].Header.Get("Authorization"))
}

func TestJinaWebSearchAdapterReaderJSONSuccess(t *testing.T) {
	doer := &fakeJinaHTTPDoer{
		handlers: map[string]fakeJinaHTTPResponse{
			"https://s.jina.ai/?q=market+news": {
				body: `[{"title":"Market News","url":"https://example.com/market","snippet":"Search snippet"}]`,
			},
			"https://r.jina.ai/https://example.com/market": {
				body: `{"title":"Market News","url":"https://example.com/market","content":"Reader JSON content with extra details."}`,
			},
		},
	}
	adapter := NewJinaWebSearchAdapter(config.WorkspaceWebSearchConfig{
		APIKey:                "test-key",
		MaxContentLengthBytes: 2048,
	}, doer)

	result, err := adapter.Search(context.Background(), WebSearchRequest{
		Query:       "market news",
		MaxResults:  3,
		MaxReadURLs: 1,
	})

	require.NoError(t, err)
	require.Len(t, result.Citations, 1)
	require.Contains(t, result.Citations[0].Snippet, "Reader JSON content")
}

func TestJinaWebSearchAdapterOversizedReaderBodyFallsBackToSearchSnippet(t *testing.T) {
	oversized := strings.Repeat("x", 300)
	doer := &fakeJinaHTTPDoer{
		handlers: map[string]fakeJinaHTTPResponse{
			"https://s.jina.ai/?q=deepseek+beta": {
				body: "[DeepSeek Beta](https://example.com/deepseek)\nSearch snippet remains available.\n",
			},
			"https://r.jina.ai/https://example.com/deepseek": {
				body: oversized,
			},
		},
	}
	adapter := NewJinaWebSearchAdapter(config.WorkspaceWebSearchConfig{
		APIKey:                "test-key",
		MaxContentLengthBytes: 128,
	}, doer)

	result, err := adapter.Search(context.Background(), WebSearchRequest{
		Query:       "deepseek beta",
		MaxResults:  3,
		MaxReadURLs: 1,
	})

	require.NoError(t, err)
	require.Len(t, result.Citations, 1)
	require.Equal(t, "Search snippet remains available.", result.Citations[0].Snippet)
	require.NotContains(t, result.Summary, oversized[:40])
}

func TestWorkspaceToolServiceWebSearchWithJinaFailureReturnsUnavailable(t *testing.T) {
	adapter := NewJinaWebSearchAdapter(config.WorkspaceWebSearchConfig{
		APIKey:                "test-key",
		MaxContentLengthBytes: 1024,
	}, &fakeJinaHTTPDoer{err: fmt.Errorf("boom")})
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		Provider:        "jina",
		APIKey:          "test-key",
		AllowedUserIDs:  []int64{1},
		DailyCapPerUser: 2,
	}), adapter)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{
		UserID: 1,
		WebSearch: WebSearchRequest{
			Query: "test",
		},
	})

	require.Error(t, err)
	require.Equal(t, WorkspaceToolStatusUnavailable, result.Status)
	require.Equal(t, WorkspaceToolErrorProviderUnavailable, result.ErrorCode)
}
