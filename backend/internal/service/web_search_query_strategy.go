package service

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	workspaceWebSearchStrategyDefault              = "default"
	workspaceWebSearchStrategySportsSchedule       = "sports_schedule_cn_en_official_first"
	workspaceWebSearchRelevanceBandHigh            = "high"
	workspaceWebSearchRelevanceBandMedium          = "medium"
	workspaceWebSearchRelevanceBandLow             = "low"
	workspaceWebSearchMaxStrategyAttempts          = 2
	workspaceWebSearchStrongCitationScoreThreshold = 60
	workspaceWebSearchCitationKeepScoreThreshold   = 20
)

var (
	workspaceWebSearchWorldCupAnchors    = []string{"世界杯", "world cup", "fifa"}
	workspaceWebSearchScheduleAnchors    = []string{"比赛", "赛程", "赛果", "fixtures", "fixture", "schedule", "match", "matches", "kickoff"}
	workspaceWebSearchRealtimeAnchors    = []string{"今天", "今日", "today", "current", "latest", "现在"}
	workspaceWebSearchSportsAnchors      = []string{"世界杯", "world cup", "fifa", "football", "soccer", "match", "fixture", "赛程", "比赛"}
	workspaceWebSearchBadAnchors         = []string{"snake", "zodiac", "astrology"}
	workspaceWebSearchGenericDateAnchors = []string{
		"calendar",
		"month calendar",
		"holidays",
		"holiday",
		"weekday",
		"thumbnail",
		"image 1",
		"calendar-365",
	}
	workspaceWebSearchPreferredHosts = []string{"fifa.com", "espn.com", "reuters.com", "apnews.com", "skysports.com", "bbc.com", "theathletic.com"}
)

type workspaceWebSearchPlan struct {
	Strategy string
	Attempts []string
	Intent   workspaceWebSearchIntent
}

type workspaceWebSearchIntent struct {
	Realtime       bool
	SportsSchedule bool
	HasChinese     bool
	EventAnchors   []string
	Year           string
	Date           workspaceWebSearchDate
}

type workspaceWebSearchDate struct {
	ISO     string
	Chinese string
	English string
}

type workspaceWebSearchRelevance struct {
	Score         int
	Band          string
	StrongCount   int
	FilteredCount int
}

type scoredCitation struct {
	citation Citation
	score    int
	strong   bool
}

func buildWorkspaceWebSearchPlan(query string, requestedAt time.Time) workspaceWebSearchPlan {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return workspaceWebSearchPlan{Strategy: workspaceWebSearchStrategyDefault}
	}
	intent := detectWorkspaceWebSearchIntent(trimmed, requestedAt)
	if !intent.SportsSchedule {
		return workspaceWebSearchPlan{
			Strategy: workspaceWebSearchStrategyDefault,
			Attempts: []string{trimmed},
			Intent:   intent,
		}
	}

	date := intent.Date
	year := firstNonEmptyWebSearchString(intent.Year, "2026")
	chineseDate := strings.TrimSpace(date.Chinese)
	englishDate := strings.TrimSpace(date.English)
	attempts := make([]string, 0, 4)

	if chineseDate != "" {
		attempts = append(attempts, fmt.Sprintf("%s 世界杯 %s 比赛 赛程 来源 site:fifa.com", year, chineseDate))
	}
	if englishDate != "" {
		attempts = append(attempts, fmt.Sprintf("FIFA World Cup %s %s fixtures site:fifa.com", year, englishDate))
		attempts = append(attempts, fmt.Sprintf("%s FIFA World Cup %s schedule fixtures", year, englishDate))
		attempts = append(attempts, fmt.Sprintf("%s World Cup %s fixtures Reuters OR AP OR ESPN", year, englishDate))
	}
	if len(attempts) == 0 {
		attempts = append(attempts, trimmed)
	}
	return workspaceWebSearchPlan{
		Strategy: workspaceWebSearchStrategySportsSchedule,
		Attempts: dedupeWorkspaceSearchAttempts(attempts, trimmed),
		Intent:   intent,
	}
}

func detectWorkspaceWebSearchIntent(query string, requestedAt time.Time) workspaceWebSearchIntent {
	lower := strings.ToLower(strings.TrimSpace(query))
	intent := workspaceWebSearchIntent{
		Realtime:       containsAnyWorkspaceAnchor(lower, workspaceWebSearchRealtimeAnchors),
		SportsSchedule: containsAnyWorkspaceAnchor(lower, workspaceWebSearchSportsAnchors) && containsAnyWorkspaceAnchor(lower, workspaceWebSearchScheduleAnchors),
		HasChinese:     containsCJK(query),
		Year:           detectWorkspaceQueryYear(lower),
	}
	for _, anchor := range workspaceWebSearchWorldCupAnchors {
		if strings.Contains(lower, anchor) {
			intent.EventAnchors = append(intent.EventAnchors, anchor)
		}
	}
	intent.Date = detectWorkspaceWebSearchDate(query, lower, requestedAt, intent.Realtime)
	if containsAnyWorkspaceAnchor(lower, workspaceWebSearchWorldCupAnchors) && containsAnyWorkspaceAnchor(lower, workspaceWebSearchSportsAnchors) {
		intent.SportsSchedule = true
	}
	return intent
}

func detectWorkspaceWebSearchDate(query, lower string, requestedAt time.Time, realtime bool) workspaceWebSearchDate {
	if iso := extractFirstWorkspaceDateISO(lower); iso != "" {
		parsed, err := time.Parse("2006-01-02", iso)
		if err == nil {
			return workspaceWebSearchDate{
				ISO:     iso,
				Chinese: fmt.Sprintf("%d月%d日", parsed.Month(), parsed.Day()),
				English: fmt.Sprintf("%s %d", parsed.Month().String(), parsed.Day()),
			}
		}
	}
	if month, day, ok := extractFirstChineseMonthDay(query); ok {
		return workspaceWebSearchDate{
			Chinese: fmt.Sprintf("%d月%d日", month, day),
			English: fmt.Sprintf("%s %d", time.Month(month).String(), day),
		}
	}
	if month, day, ok := extractFirstEnglishMonthDay(lower); ok {
		return workspaceWebSearchDate{
			Chinese: fmt.Sprintf("%d月%d日", month, day),
			English: fmt.Sprintf("%s %d", time.Month(month).String(), day),
		}
	}
	if realtime {
		when := requestedAt
		if when.IsZero() {
			when = time.Now()
		}
		return workspaceWebSearchDate{
			ISO:     when.Format("2006-01-02"),
			Chinese: fmt.Sprintf("%d月%d日", when.Month(), when.Day()),
			English: fmt.Sprintf("%s %d", when.Month().String(), when.Day()),
		}
	}
	return workspaceWebSearchDate{}
}

func applyWorkspaceWebSearchQualityGuard(plan workspaceWebSearchPlan, result WebSearchResult) (WebSearchResult, workspaceWebSearchRelevance) {
	if !plan.Intent.SportsSchedule {
		citations := cloneWorkspaceCitations(result.Citations)
		for i := range citations {
			citations[i].Index = i + 1
		}
		return WebSearchResult{
				Query:     result.Query,
				Summary:   buildCitationSummary(citations),
				Citations: citations,
			}, workspaceWebSearchRelevance{
				Score:         workspaceWebSearchStrongCitationScoreThreshold,
				Band:          workspaceWebSearchRelevanceBandHigh,
				StrongCount:   len(citations),
				FilteredCount: len(citations),
			}
	}
	if len(result.Citations) == 0 {
		return WebSearchResult{Query: result.Query}, workspaceWebSearchRelevance{Band: workspaceWebSearchRelevanceBandLow}
	}
	scored := make([]scoredCitation, 0, len(result.Citations))
	for _, citation := range result.Citations {
		score, strong := scoreWorkspaceCitation(plan, citation)
		if score < workspaceWebSearchCitationKeepScoreThreshold {
			continue
		}
		scored = append(scored, scoredCitation{citation: citation, score: score, strong: strong})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].citation.Index < scored[j].citation.Index
		}
		return scored[i].score > scored[j].score
	})
	filtered := make([]Citation, 0, len(scored))
	bestScore := 0
	strongCount := 0
	for i, entry := range scored {
		if entry.score > bestScore {
			bestScore = entry.score
		}
		if entry.strong {
			strongCount++
		}
		entry.citation.Index = i + 1
		filtered = append(filtered, entry.citation)
	}
	band := workspaceWebSearchRelevanceBandLow
	switch {
	case strongCount > 0:
		band = workspaceWebSearchRelevanceBandHigh
	case bestScore >= 40:
		band = workspaceWebSearchRelevanceBandMedium
	}
	return WebSearchResult{
			Query:     result.Query,
			Summary:   buildCitationSummary(filtered),
			Citations: filtered,
		}, workspaceWebSearchRelevance{
			Score:         bestScore,
			Band:          band,
			StrongCount:   strongCount,
			FilteredCount: len(filtered),
		}
}

func cloneWorkspaceCitations(input []Citation) []Citation {
	if len(input) == 0 {
		return nil
	}
	out := make([]Citation, len(input))
	copy(out, input)
	return out
}

func scoreWorkspaceCitation(plan workspaceWebSearchPlan, citation Citation) (int, bool) {
	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		citation.Title,
		citation.Domain,
		citation.URL,
		citation.Snippet,
	}, " ")))
	score := 0
	scheduleMatched := containsAnyWorkspaceAnchor(text, workspaceWebSearchScheduleAnchors)
	eventMatched := containsAnyWorkspaceAnchor(text, workspaceWebSearchWorldCupAnchors)
	genericDateOnly := workspaceCitationLooksGenericDateOnly(text, citation.Domain) && !(eventMatched && scheduleMatched)
	if strings.Contains(citation.Domain, "fifa.com") {
		score += 40
	} else if containsPreferredWorkspaceHost(citation.Domain) {
		score += 20
	}
	if eventMatched {
		score += 25
	}
	if scheduleMatched {
		score += 20
	}
	if plan.Intent.Year != "" && strings.Contains(text, plan.Intent.Year) {
		score += 10
	}
	dateMatched := false
	for _, token := range []string{plan.Intent.Date.Chinese, plan.Intent.Date.English, plan.Intent.Date.ISO} {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		if strings.Contains(text, token) {
			score += 15
			dateMatched = true
			break
		}
	}
	if containsAnyWorkspaceAnchor(text, workspaceWebSearchBadAnchors) {
		score -= 80
	}
	if genericDateOnly {
		score -= 90
	}
	if plan.Intent.SportsSchedule && !eventMatched {
		score -= 25
	}
	if plan.Intent.SportsSchedule && !scheduleMatched {
		score -= 20
	}
	if plan.Intent.SportsSchedule && eventMatched && !scheduleMatched && !containsPreferredWorkspaceHost(citation.Domain) {
		score -= 15
	}
	strong := score >= workspaceWebSearchStrongCitationScoreThreshold && eventMatched && scheduleMatched && (dateMatched || plan.Intent.Date == (workspaceWebSearchDate{}))
	return score, strong
}

func workspaceCitationLooksGenericDateOnly(text, domain string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	domain = strings.ToLower(strings.TrimSpace(domain))
	if strings.Contains(domain, "calendar-365.com") {
		return true
	}
	return containsAnyWorkspaceAnchor(text, workspaceWebSearchGenericDateAnchors)
}

func detectWorkspaceQueryYear(lower string) string {
	for _, token := range []string{"2026", "2025", "2027"} {
		if strings.Contains(lower, token) {
			return token
		}
	}
	return ""
}

func dedupeWorkspaceSearchAttempts(attempts []string, fallback string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(attempts)+1)
	for _, attempt := range attempts {
		trimmed := strings.TrimSpace(attempt)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 && strings.TrimSpace(fallback) != "" {
		out = append(out, strings.TrimSpace(fallback))
	}
	return out
}

func containsAnyWorkspaceAnchor(text string, anchors []string) bool {
	for _, anchor := range anchors {
		if anchor != "" && strings.Contains(text, strings.ToLower(anchor)) {
			return true
		}
	}
	return false
}

func containsPreferredWorkspaceHost(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, host := range workspaceWebSearchPreferredHosts {
		if strings.Contains(domain, host) {
			return true
		}
	}
	return false
}

func containsCJK(value string) bool {
	for _, r := range value {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

func extractFirstChineseMonthDay(value string) (int, int, bool) {
	for month := 1; month <= 12; month++ {
		for day := 1; day <= 31; day++ {
			if strings.Contains(value, fmt.Sprintf("%d月%d日", month, day)) {
				return month, day, true
			}
		}
	}
	return 0, 0, false
}

func extractFirstEnglishMonthDay(lower string) (int, int, bool) {
	months := []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}
	for idx, month := range months {
		for day := 1; day <= 31; day++ {
			if strings.Contains(lower, fmt.Sprintf("%s %d", month, day)) {
				return idx + 1, day, true
			}
		}
	}
	return 0, 0, false
}

func extractFirstWorkspaceDateISO(lower string) string {
	for year := 2025; year <= 2027; year++ {
		for month := 1; month <= 12; month++ {
			for day := 1; day <= 31; day++ {
				token := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
				if strings.Contains(lower, token) {
					return token
				}
			}
		}
	}
	return ""
}
