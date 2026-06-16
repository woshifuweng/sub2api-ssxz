package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type fakeWebSearchTool struct {
	called bool
	err    error
}

func (f *fakeWebSearchTool) Search(_ context.Context, req WebSearchRequest) (WebSearchResult, error) {
	f.called = true
	if f.err != nil {
		return WebSearchResult{}, f.err
	}
	return WebSearchResult{
		Query: req.Query,
		Citations: []Citation{{
			Index:       1,
			Title:       "Result",
			URL:         "https://example.com/article",
			Domain:      "example.com",
			Snippet:     "Snippet",
			RetrievedAt: time.Now().UTC(),
		}},
	}, nil
}

func TestWorkspaceToolServiceWebSearchDefaultDisabled(t *testing.T) {
	tool := &fakeWebSearchTool{}
	svc := NewWorkspaceToolService(&config.Config{}, tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{
		UserID: 1,
		WebSearch: WebSearchRequest{
			Query: "hello",
		},
	})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolStatusUnavailable, result.Status)
	require.Equal(t, WorkspaceToolErrorDisabled, result.ErrorCode)
	require.False(t, tool.called)
}

func TestWorkspaceToolServiceWebSearchKillSwitchBlocks(t *testing.T) {
	tool := &fakeWebSearchTool{}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:        true,
		KillSwitch:     true,
		AllowedUserIDs: []int64{1},
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{UserID: 1})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolErrorKillSwitch, result.ErrorCode)
	require.False(t, tool.called)
}

func TestWorkspaceToolServiceWebSearchUserAllowlistBlocks(t *testing.T) {
	tool := &fakeWebSearchTool{}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:        true,
		KillSwitch:     false,
		AllowedUserIDs: []int64{2},
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{UserID: 1})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolErrorUserNotAllowed, result.ErrorCode)
	require.False(t, tool.called)
}

func TestWorkspaceToolServiceWebSearchDailyCapBlocks(t *testing.T) {
	tool := &fakeWebSearchTool{}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		AllowedUserIDs:  []int64{1},
		DailyCapPerUser: 2,
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{
		UserID:          1,
		UsageCountToday: 2,
	})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolErrorCapExceeded, result.ErrorCode)
	require.False(t, tool.called)
}

func TestWorkspaceToolServiceWebSearchProviderUnavailableWithoutImplementation(t *testing.T) {
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		AllowedUserIDs:  []int64{1},
		DailyCapPerUser: 2,
	}), nil)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{UserID: 1})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolErrorProviderUnavailable, result.ErrorCode)
}

func TestWorkspaceToolServiceWebSearchRejectsUnsafeReadURLs(t *testing.T) {
	unsafeURLs := []string{
		"http://localhost/page",
		"http://127.0.0.1/page",
		"http://10.0.0.1/page",
		"http://172.16.0.1/page",
		"http://192.168.1.1/page",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/page",
		"http://metadata.google.internal/computeMetadata/v1/",
		"file:///etc/passwd",
	}

	for _, rawURL := range unsafeURLs {
		t.Run(rawURL, func(t *testing.T) {
			require.Error(t, ValidateWorkspaceWebSearchURL(rawURL))
		})
	}
}

func TestWorkspaceToolServiceWebSearchAcceptsHTTPURLs(t *testing.T) {
	require.NoError(t, ValidateWorkspaceWebSearchURL("https://example.com/a"))
	require.NoError(t, ValidateWorkspaceWebSearchURL("http://example.com/a"))
}

func TestWorkspaceToolServiceWebSearchCallsInjectedToolOnlyWhenGatePasses(t *testing.T) {
	tool := &fakeWebSearchTool{}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		AllowedUserIDs:  []int64{1},
		MaxResults:      5,
		MaxReadURLs:     3,
		TimeoutMS:       8000,
		DailyCapPerUser: 2,
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{
		UserID: 1,
		WebSearch: WebSearchRequest{
			Query:    "test",
			ReadURLs: []string{"https://example.com/a"},
		},
	})

	require.NoError(t, err)
	require.True(t, tool.called)
	require.Equal(t, WorkspaceToolStatusCompleted, result.Status)
	require.Len(t, result.Citations, 1)
	require.Equal(t, 1, result.UsageLog.ResultCount)
	require.Equal(t, 1, result.UsageLog.ReadURLCount)
}

func TestWorkspaceToolServiceWebSearchProviderErrorDoesNotHideBoundary(t *testing.T) {
	tool := &fakeWebSearchTool{err: errors.New("boom")}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		AllowedUserIDs:  []int64{1},
		DailyCapPerUser: 2,
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{UserID: 1})

	require.Error(t, err)
	require.True(t, tool.called)
	require.Equal(t, WorkspaceToolStatusUnavailable, result.Status)
	require.Equal(t, WorkspaceToolErrorProviderUnavailable, result.ErrorCode)
}

func webSearchTestConfig(webSearch config.WorkspaceWebSearchConfig) *config.Config {
	if webSearch.Provider == "" {
		webSearch.Provider = "jina"
	}
	return &config.Config{
		Workspace: config.WorkspaceConfig{
			WebSearch: webSearch,
		},
	}
}
