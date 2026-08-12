package admin

import (
	"context"
	"encoding/json"
	"time"
)

type accountListCacheKey struct {
	Page                  int    `json:"page"`
	PageSize              int    `json:"page_size"`
	Platform              string `json:"platform"`
	AccountType           string `json:"type"`
	Status                string `json:"status"`
	Search                string `json:"search"`
	Group                 int64  `json:"group"`
	PrivacyMode           string `json:"privacy_mode"`
	SortBy                string `json:"sort_by"`
	SortOrder             string `json:"sort_order"`
	Lite                  bool   `json:"lite"`
	IncludeSchedulerScore bool   `json:"include_scheduler_score"`
}

type accountListCachePayload struct {
	Items    []AccountWithConcurrency `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func (h *AccountHandler) getAccountListCached(
	ctx context.Context,
	page, pageSize int,
	platform, accountType, status, search, privacyMode, sortBy, sortOrder string,
	groupID int64,
	lite, includeSchedulerScore bool,
	load func(context.Context) ([]AccountWithConcurrency, int64, error),
) (accountListCachePayload, bool, error) {
	keyRaw, _ := json.Marshal(accountListCacheKey{
		Page:                  page,
		PageSize:              pageSize,
		Platform:              platform,
		AccountType:           accountType,
		Status:                status,
		Search:                search,
		Group:                 groupID,
		PrivacyMode:           privacyMode,
		SortBy:                sortBy,
		SortOrder:             sortOrder,
		Lite:                  lite,
		IncludeSchedulerScore: includeSchedulerScore,
	})
	cache := h.accountListCache
	if cache == nil {
		cache = newSnapshotCache(5 * time.Second)
		h.accountListCache = cache
	}
	entry, hit, err := cache.GetOrLoad(string(keyRaw), func() (any, error) {
		items, total, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return accountListCachePayload{
			Items:    items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	})
	if err != nil {
		return accountListCachePayload{}, hit, err
	}
	payload, err := snapshotPayloadAs[accountListCachePayload](entry.Payload)
	return payload, hit, err
}
