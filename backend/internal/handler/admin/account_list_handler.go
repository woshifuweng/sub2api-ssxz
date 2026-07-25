package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/rustbridge/ffi"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

func (h *AccountHandler) listAccountSchedulerScoreFilterPool(
	ctx context.Context,
	platform, accountType, status, search string,
	groupID int64,
	privacyMode string,
) []service.Account {
	if h.adminService == nil || (platform != "" && platform != service.PlatformOpenAI) {
		return nil
	}
	accounts, err := h.adminService.ListAccountsForSchedulerScoreFilter(
		ctx, service.PlatformOpenAI, accountType, status, search, groupID, privacyMode,
	)
	if err != nil {
		slog.Warn("openai_scheduler_filter_score_pool_failed", "error", err)
		return nil
	}
	return accounts
}

// List handles listing all accounts with pagination.
// GET /api/v1/admin/accounts
func (h *AccountHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *AccountHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	platform := strings.TrimSpace(c.QueryValue("platform"))
	accountType := strings.TrimSpace(c.QueryValue("type"))
	status := strings.TrimSpace(c.QueryValue("status"))
	search := strings.TrimSpace(c.QueryValue("search"))
	privacyMode := strings.TrimSpace(c.QueryValue("privacy_mode"))
	sortBy := strings.TrimSpace(c.QueryValue("sort_by"))
	sortOrder := strings.TrimSpace(c.QueryValue("sort_order"))
	if sortBy == "" {
		sortBy = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}
	if len(search) > 100 {
		search = search[:100]
	}
	lite := parseBoolQueryWithDefault(c.QueryValue("lite"), false)
	includeSchedulerScore := parseBoolQueryWithDefault(c.QueryValue("include_scheduler_score"), false)

	var groupID int64
	if groupIDStr := strings.TrimSpace(c.QueryValue("group")); groupIDStr != "" {
		if groupIDStr == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, err := strconv.ParseInt(groupIDStr, 10, 64)
			if err != nil || parsedGroupID < 0 {
				response.ErrorFromContext(
					gatewayJSONResponder{ctx: c},
					infraerrors.BadRequest("INVALID_GROUP_FILTER", "invalid group filter"),
				)
				return
			}
			groupID = parsedGroupID
		}
	}

	ctx := c.Request().Context()
	cachedPayload, hit, err := h.getAccountListCached(
		ctx,
		page, pageSize,
		platform, accountType, status, search, privacyMode, sortBy, sortOrder,
		groupID, lite, includeSchedulerScore,
		func(ctx context.Context) ([]AccountWithConcurrency, int64, error) {
			accounts, total, err := h.adminService.ListAccounts(
				ctx, page, pageSize, platform, accountType, status, search,
				groupID, privacyMode, sortBy, sortOrder,
			)
			if err != nil {
				return nil, 0, err
			}

			accountIDs := make([]int64, len(accounts))
			for i := range accounts {
				accountIDs[i] = accounts[i].ID
			}
			concurrencyCounts := make(map[int64]int)
			if h.concurrencyService != nil && len(accountIDs) > 0 {
				if counts, countErr := h.concurrencyService.GetAccountConcurrencyBatch(ctx, accountIDs); countErr == nil && counts != nil {
					concurrencyCounts = counts
				}
			}

			var schedulerScores map[int64]*AccountSchedulerScore
			var schedulerGroupScores map[int64][]AccountSchedulerGroupScore
			if includeSchedulerScore {
				pageHasOpenAIAccounts := false
				for i := range accounts {
					if accounts[i].Platform == service.PlatformOpenAI {
						pageHasOpenAIAccounts = true
						break
					}
				}
				if pageHasOpenAIAccounts {
					filterPool := h.listAccountSchedulerScoreFilterPool(
						ctx, platform, accountType, status, search, groupID, privacyMode,
					)
					schedulerScores, schedulerGroupScores = h.buildOpenAIAccountSchedulerScores(ctx, accounts, filterPool)
				}
			}

			windowCostAccountIDs := make([]int64, 0)
			sessionLimitAccountIDs := make([]int64, 0)
			rpmAccountIDs := make([]int64, 0)
			sessionIdleTimeouts := make(map[int64]time.Duration)
			for i := range accounts {
				account := &accounts[i]
				if !account.IsAnthropicOAuthOrSetupToken() {
					continue
				}
				if account.GetWindowCostLimit() > 0 {
					windowCostAccountIDs = append(windowCostAccountIDs, account.ID)
				}
				if account.GetMaxSessions() > 0 {
					sessionLimitAccountIDs = append(sessionLimitAccountIDs, account.ID)
					sessionIdleTimeouts[account.ID] = time.Duration(account.GetSessionIdleTimeoutMinutes()) * time.Minute
				}
				if account.GetBaseRPM() > 0 {
					rpmAccountIDs = append(rpmAccountIDs, account.ID)
				}
			}

			var rpmCounts map[int64]int
			if len(rpmAccountIDs) > 0 && h.rpmCache != nil {
				rpmCounts, _ = h.rpmCache.GetRPMBatch(ctx, rpmAccountIDs)
				if rpmCounts == nil {
					rpmCounts = make(map[int64]int)
				}
			}

			var activeSessions map[int64]int
			if len(sessionLimitAccountIDs) > 0 && h.sessionLimitCache != nil {
				activeSessions, _ = h.sessionLimitCache.GetActiveSessionCountBatch(
					ctx, sessionLimitAccountIDs, sessionIdleTimeouts,
				)
				if activeSessions == nil {
					activeSessions = make(map[int64]int)
				}
			}

			var windowCosts map[int64]float64
			if len(windowCostAccountIDs) > 0 && h.accountUsageService != nil {
				windowCosts = make(map[int64]float64, len(windowCostAccountIDs))
				idsByWindowStart := make(map[int64][]int64)
				windowStartTimes := make(map[int64]time.Time)
				for i := range accounts {
					account := &accounts[i]
					if !account.IsAnthropicOAuthOrSetupToken() || account.GetWindowCostLimit() <= 0 {
						continue
					}
					startTime := account.GetCurrentWindowStartTime()
					key := startTime.Unix()
					idsByWindowStart[key] = append(idsByWindowStart[key], account.ID)
					windowStartTimes[key] = startTime
				}

				var mu sync.Mutex
				group, groupCtx := errgroup.WithContext(ctx)
				group.SetLimit(4)
				for windowStartKey, ids := range idsByWindowStart {
					startTime := windowStartTimes[windowStartKey]
					idsCopy := append([]int64(nil), ids...)
					group.Go(func() error {
						statsByAccount, statsErr := h.accountUsageService.GetAccountWindowStatsBatch(
							groupCtx, idsCopy, startTime,
						)
						if statsErr != nil {
							return nil
						}
						mu.Lock()
						for accountID, stats := range statsByAccount {
							if stats != nil {
								windowCosts[accountID] = stats.StandardCost
							}
						}
						mu.Unlock()
						return nil
					})
				}
				_ = group.Wait()
			}

			result := make([]AccountWithConcurrency, len(accounts))
			for i := range accounts {
				account := &accounts[i]
				item := AccountWithConcurrency{
					Account:            dto.AccountFromService(account),
					CurrentConcurrency: concurrencyCounts[account.ID],
					SchedulerScore:     schedulerScores[account.ID],
					SchedulerScores:    schedulerGroupScores[account.ID],
				}
				if cost, ok := windowCosts[account.ID]; ok {
					item.CurrentWindowCost = &cost
				}
				if count, ok := activeSessions[account.ID]; ok {
					item.ActiveSessions = &count
				}
				if rpm, ok := rpmCounts[account.ID]; ok {
					item.CurrentRPM = &rpm
				}
				result[i] = item
			}

			h.enrichShadowParents(ctx, result)
			return result, total, nil
		},
	)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))
	etag := buildAccountsListETag(
		cachedPayload.Items, cachedPayload.Total, page, pageSize,
		platform, accountType, status, search, privacyMode, sortBy, sortOrder,
		groupID, lite, includeSchedulerScore,
	)
	if etag != "" {
		c.SetHeader("ETag", etag)
		c.SetHeader("Vary", "If-None-Match")
		if ifNoneMatchMatched(c.HeaderValue("If-None-Match"), etag) {
			_, _ = c.WriteBytes(http.StatusNotModified, nil)
			return
		}
	}

	response.PaginatedContext(
		gatewayJSONResponder{ctx: c}, cachedPayload.Items, cachedPayload.Total, page, pageSize,
	)
}

func buildAccountsListETag(
	items []AccountWithConcurrency,
	total int64,
	page, pageSize int,
	platform, accountType, status, search, privacyMode, sortBy, sortOrder string,
	groupID int64,
	lite, includeSchedulerScore bool,
) string {
	payload := struct {
		Total                 int64                    `json:"total"`
		Page                  int                      `json:"page"`
		PageSize              int                      `json:"page_size"`
		Platform              string                   `json:"platform"`
		AccountType           string                   `json:"type"`
		Status                string                   `json:"status"`
		Search                string                   `json:"search"`
		PrivacyMode           string                   `json:"privacy_mode"`
		SortBy                string                   `json:"sort_by"`
		SortOrder             string                   `json:"sort_order"`
		GroupID               int64                    `json:"group_id"`
		Lite                  bool                     `json:"lite"`
		IncludeSchedulerScore bool                     `json:"include_scheduler_score"`
		Items                 []AccountWithConcurrency `json:"items"`
	}{
		Total:                 total,
		Page:                  page,
		PageSize:              pageSize,
		Platform:              platform,
		AccountType:           accountType,
		Status:                status,
		Search:                search,
		PrivacyMode:           privacyMode,
		SortBy:                sortBy,
		SortOrder:             sortOrder,
		GroupID:               groupID,
		Lite:                  lite,
		IncludeSchedulerScore: includeSchedulerScore,
		Items:                 items,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return ffi.BuildETagFromBytes(raw)
}

func ifNoneMatchMatched(ifNoneMatch, etag string) bool {
	if etag == "" || ifNoneMatch == "" {
		return false
	}
	for _, token := range strings.Split(ifNoneMatch, ",") {
		candidate := strings.TrimSpace(token)
		if candidate == "*" || candidate == etag {
			return true
		}
		if strings.HasPrefix(candidate, "W/") && strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
