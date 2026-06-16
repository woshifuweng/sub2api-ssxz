package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type scriptedWebSearchTool struct {
	queries []string
	results map[string]WebSearchResult
}

func (s *scriptedWebSearchTool) Search(_ context.Context, req WebSearchRequest) (WebSearchResult, error) {
	s.queries = append(s.queries, req.Query)
	if result, ok := s.results[req.Query]; ok {
		return result, nil
	}
	return WebSearchResult{Query: req.Query}, nil
}

func TestBuildWorkspaceWebSearchPlanRewritesChineseWorldCupDateQuery(t *testing.T) {
	plan := buildWorkspaceWebSearchPlan("2026 世界杯 6月16日 有哪些比赛？请给出来源。", time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC))

	require.Equal(t, workspaceWebSearchStrategySportsSchedule, plan.Strategy)
	require.True(t, plan.Intent.SportsSchedule)
	require.Equal(t, "6月16日", plan.Intent.Date.Chinese)
	require.Equal(t, "June 16", plan.Intent.Date.English)
	require.Contains(t, plan.Attempts[0], "site:fifa.com")
	require.Contains(t, plan.Attempts[0], "2026 世界杯 6月16日 比赛 赛程 来源")
	require.Contains(t, plan.Attempts[1], "FIFA World Cup 2026 June 16 fixtures site:fifa.com")
}

func TestBuildWorkspaceWebSearchPlanDetectsRealtimeSportsScheduleIntent(t *testing.T) {
	plan := buildWorkspaceWebSearchPlan("今天 2026 世界杯有哪些比赛？请给出来源。", time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC))

	require.True(t, plan.Intent.Realtime)
	require.True(t, plan.Intent.SportsSchedule)
	require.Equal(t, "6月16日", plan.Intent.Date.Chinese)
	require.Equal(t, "June 16", plan.Intent.Date.English)
}

func TestApplyWorkspaceWebSearchQualityGuardRejectsSnakeCitationForWorldCup(t *testing.T) {
	plan := buildWorkspaceWebSearchPlan("2026 世界杯 6月16日 有哪些比赛？请给出来源。", time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC))
	result := WebSearchResult{
		Query: "bad query",
		Citations: []Citation{
			{Index: 1, Title: "Snake (zodiac) - Wikipedia", Domain: "en.wikipedia.org", URL: "https://en.wikipedia.org/wiki/Snake_(zodiac)", Snippet: "Snake zodiac overview and astrology background."},
		},
	}

	filtered, relevance := applyWorkspaceWebSearchQualityGuard(plan, result)

	require.Empty(t, filtered.Citations)
	require.Equal(t, workspaceWebSearchRelevanceBandLow, relevance.Band)
	require.Zero(t, relevance.StrongCount)
}

func TestApplyWorkspaceWebSearchQualityGuardPrefersOfficialFIFAResult(t *testing.T) {
	plan := buildWorkspaceWebSearchPlan("2026 世界杯 6月16日 有哪些比赛？请给出来源。", time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC))
	result := WebSearchResult{
		Query: "query",
		Citations: []Citation{
			{Index: 1, Title: "World Cup roundup", Domain: "example.com", URL: "https://example.com/world-cup", Snippet: "World Cup 2026 news and commentary."},
			{Index: 2, Title: "FIFA World Cup 2026 fixtures", Domain: "fifa.com", URL: "https://www.fifa.com/worldcup/fixtures", Snippet: "FIFA World Cup 2026 June 16 fixtures and kickoff schedule."},
		},
	}

	filtered, relevance := applyWorkspaceWebSearchQualityGuard(plan, result)

	require.Len(t, filtered.Citations, 2)
	require.Equal(t, "fifa.com", filtered.Citations[0].Domain)
	require.Equal(t, workspaceWebSearchRelevanceBandHigh, relevance.Band)
	require.GreaterOrEqual(t, relevance.StrongCount, 1)
}

func TestWorkspaceToolServiceSearchWebRetriesWithFIFAFocusedQueryAndFailsLowRelevance(t *testing.T) {
	tool := &scriptedWebSearchTool{
		results: map[string]WebSearchResult{
			"2026 世界杯 6月16日 比赛 赛程 来源 site:fifa.com": {
				Query: "2026 世界杯 6月16日 比赛 赛程 来源 site:fifa.com",
				Citations: []Citation{
					{Index: 1, Title: "Snake (zodiac) - Wikipedia", Domain: "en.wikipedia.org", URL: "https://en.wikipedia.org/wiki/Snake_(zodiac)", Snippet: "Snake zodiac astrology."},
				},
			},
			"FIFA World Cup 2026 June 16 fixtures site:fifa.com": {
				Query: "FIFA World Cup 2026 June 16 fixtures site:fifa.com",
				Citations: []Citation{
					{Index: 1, Title: "Astrology and snake sign", Domain: "example.com", URL: "https://example.com/snake", Snippet: "Snake zodiac and astrology."},
				},
			},
		},
	}
	svc := NewWorkspaceToolService(webSearchTestConfig(config.WorkspaceWebSearchConfig{
		Enabled:         true,
		KillSwitch:      false,
		AllowedUserIDs:  []int64{1},
		MaxResults:      3,
		MaxReadURLs:     1,
		TimeoutMS:       30000,
		DailyCapPerUser: 10,
	}), tool)

	result, err := svc.SearchWeb(context.Background(), WorkspaceToolRequest{
		UserID:      1,
		RequestedAt: time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC),
		WebSearch: WebSearchRequest{
			Query: "2026 世界杯 6月16日 有哪些比赛？请给出来源。",
		},
	})

	require.ErrorIs(t, err, ErrWorkspaceToolUnavailable)
	require.Equal(t, WorkspaceToolStatusLowRelevance, result.Status)
	require.Equal(t, WorkspaceToolErrorLowRelevance, result.ErrorCode)
	require.Equal(t, workspaceWebSearchStrategySportsSchedule, result.Metadata["strategy"])
	require.Equal(t, 2, result.Metadata["attempts"])
	require.Equal(t, []string{
		"2026 世界杯 6月16日 比赛 赛程 来源 site:fifa.com",
		"FIFA World Cup 2026 June 16 fixtures site:fifa.com",
	}, tool.queries)
}
