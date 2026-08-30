package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var adminAPIKeySuffixPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{4}$`)

var adminAPIKeySortExpressions = map[string]string{
	service.AdminAPIKeySortCreatedAt:       "fk.created_at",
	service.AdminAPIKeySortLastUsedAt:      "fk.last_used_at",
	service.AdminAPIKeySortTodayCost:       "COALESCE(usage.today_actual_cost, 0)",
	service.AdminAPIKeySortLast30DaysCost:  "COALESCE(usage.last_30_days_actual_cost, 0)",
	service.AdminAPIKeySortTotalActualCost: "COALESCE(usage.total_actual_cost, 0)",
}

func (r *apiKeyRepository) ListAdminAPIKeys(ctx context.Context, params service.AdminAPIKeyListParams) (*service.AdminAPIKeyListResult, error) {
	if r.sql == nil {
		return nil, fmt.Errorf("admin API key list SQL executor is unavailable")
	}

	sortBy, sortOrder := service.NormalizeAdminAPIKeySort(params.Pagination.SortBy, params.Pagination.SortOrder)
	params.Pagination.SortBy = sortBy
	params.Pagination.SortOrder = sortOrder
	whereSQL, args := buildAdminAPIKeyWhere(params.Filters)

	summary, err := r.queryAdminAPIKeySummary(ctx, whereSQL, args)
	if err != nil {
		return nil, err
	}

	items, err := r.queryAdminAPIKeyItems(ctx, whereSQL, args, params.Pagination)
	if err != nil {
		return nil, err
	}

	page := params.Pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.Pagination.Limit()
	pages := int((summary.Total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		pages = 1
	}

	return &service.AdminAPIKeyListResult{
		Items: items,
		Pagination: pagination.PaginationResult{
			Total: summary.Total, Page: page, PageSize: pageSize, Pages: pages,
		},
		Summary: summary,
	}, nil
}

func buildAdminAPIKeyWhere(filters service.AdminAPIKeyListFilters) (string, []any) {
	conditions := []string{"k.deleted_at IS NULL", "u.deleted_at IS NULL"}
	args := make([]any, 0, 5)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filters.UserID != nil && *filters.UserID > 0 {
		conditions = append(conditions, "k.user_id = "+addArg(*filters.UserID))
	}
	if filters.GroupID != nil && *filters.GroupID > 0 {
		arg := addArg(*filters.GroupID)
		conditions = append(conditions, "(k.group_id = "+arg+" OR "+arg+" = ANY(COALESCE(k.group_ids, ARRAY[]::bigint[])))")
	}

	switch filters.Status {
	case "active":
		conditions = append(conditions, "k.status = 'active' AND (k.expires_at IS NULL OR k.expires_at > NOW())")
	case "inactive":
		conditions = append(conditions, "k.status IN ('disabled', 'inactive', 'quota_exhausted') AND (k.expires_at IS NULL OR k.expires_at > NOW())")
	case "expired":
		conditions = append(conditions, "(k.status = 'expired' OR k.expires_at <= NOW())")
	}

	search := strings.TrimSpace(filters.Search)
	if search != "" {
		patternArg := addArg("%" + strings.ToLower(search) + "%")
		exactArg := addArg(search)
		searchConditions := []string{
			"LOWER(k.name) LIKE " + patternArg,
			"LOWER(u.email) LIKE " + patternArg,
			"LOWER(u.username) LIKE " + patternArg,
			"CAST(u.id AS TEXT) = " + exactArg,
			"CAST(k.id AS TEXT) = " + exactArg,
		}
		if adminAPIKeySuffixPattern.MatchString(search) {
			searchConditions = append(searchConditions, "RIGHT(k.key, 4) = "+exactArg)
		}
		conditions = append(conditions, "("+strings.Join(searchConditions, " OR ")+")")
	}

	return strings.Join(conditions, " AND "), args
}

func adminAPIKeyFilteredCTE(whereSQL string) string {
	return `
WITH filtered_keys AS (
  SELECT
    k.id, k.user_id, k.key, k.name, k.group_id, k.status, k.last_used_at,
    k.quota, k.quota_used, k.expires_at, k.created_at, k.updated_at,
    u.email AS user_email, u.username AS user_username, u.balance AS user_balance,
    g.name AS group_name, g.platform AS group_platform, g.rate_multiplier AS group_rate_multiplier
  FROM api_keys k
  JOIN users u ON u.id = k.user_id
  LEFT JOIN groups g ON g.id = k.group_id AND g.deleted_at IS NULL
  WHERE ` + whereSQL + `
)`
}

func (r *apiKeyRepository) queryAdminAPIKeySummary(ctx context.Context, whereSQL string, args []any) (service.AdminAPIKeyListSummary, error) {
	query := adminAPIKeyFilteredCTE(whereSQL) + `,
usage_30d AS (
  SELECT ul.api_key_id, COALESCE(SUM(ul.actual_cost), 0) AS last_30_days_actual_cost
  FROM usage_logs ul
  JOIN filtered_keys fk ON fk.id = ul.api_key_id
  WHERE ul.created_at >= NOW() - INTERVAL '30 days'
  GROUP BY ul.api_key_id
)
SELECT
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE fk.status = 'active' AND (fk.expires_at IS NULL OR fk.expires_at > NOW())) AS active,
  COUNT(*) FILTER (WHERE fk.status <> 'active' AND NOT (fk.status = 'expired' OR COALESCE(fk.expires_at <= NOW(), FALSE))) AS inactive,
  COUNT(*) FILTER (WHERE fk.status = 'expired' OR fk.expires_at <= NOW()) AS expired,
  COALESCE(SUM(usage_30d.last_30_days_actual_cost), 0) AS last_30_days_actual_cost
FROM filtered_keys fk
LEFT JOIN usage_30d ON usage_30d.api_key_id = fk.id`

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return service.AdminAPIKeyListSummary{}, fmt.Errorf("query admin API key summary: %w", err)
	}
	defer rows.Close()

	var summary service.AdminAPIKeyListSummary
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return summary, fmt.Errorf("scan admin API key summary: %w", err)
		}
		return summary, nil
	}
	if err := rows.Scan(&summary.Total, &summary.Active, &summary.Inactive, &summary.Expired, &summary.Last30DaysActualCost); err != nil {
		return summary, fmt.Errorf("scan admin API key summary: %w", err)
	}
	return summary, rows.Err()
}

func (r *apiKeyRepository) queryAdminAPIKeyItems(ctx context.Context, whereSQL string, args []any, params pagination.PaginationParams) ([]service.AdminAPIKeyListItem, error) {
	sortExpression := adminAPIKeySortExpressions[params.SortBy]
	if sortExpression == "" {
		sortExpression = adminAPIKeySortExpressions[service.AdminAPIKeySortLastUsedAt]
	}
	sortDirection := "DESC"
	if params.SortOrder == pagination.SortOrderAsc {
		sortDirection = "ASC"
	}

	queryArgs := append([]any(nil), args...)
	limitPosition := len(queryArgs) + 1
	offsetPosition := len(queryArgs) + 2
	queryArgs = append(queryArgs, params.Limit(), params.Offset())

	query := adminAPIKeyFilteredCTE(whereSQL) + `,
usage AS (
  SELECT
    ul.api_key_id,
    COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= CURRENT_DATE), 0) AS today_actual_cost,
    COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= NOW() - INTERVAL '30 days'), 0) AS last_30_days_actual_cost,
    COALESCE(SUM(ul.actual_cost), 0) AS total_actual_cost
  FROM usage_logs ul
  JOIN filtered_keys fk ON fk.id = ul.api_key_id
  GROUP BY ul.api_key_id
)
SELECT
  fk.id, fk.user_id, fk.key, fk.name, fk.group_id, fk.status, fk.last_used_at,
  fk.quota, fk.quota_used, fk.expires_at, fk.created_at, fk.updated_at,
  fk.user_email, fk.user_username, fk.user_balance,
  fk.group_name, fk.group_platform, fk.group_rate_multiplier,
  COALESCE(usage.today_actual_cost, 0),
  COALESCE(usage.last_30_days_actual_cost, 0),
  COALESCE(usage.total_actual_cost, 0)
FROM filtered_keys fk
LEFT JOIN usage ON usage.api_key_id = fk.id
ORDER BY ` + sortExpression + ` ` + sortDirection + ` NULLS LAST, fk.id DESC
LIMIT $` + fmt.Sprint(limitPosition) + ` OFFSET $` + fmt.Sprint(offsetPosition)

	rows, err := r.sql.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query admin API keys: %w", err)
	}
	defer rows.Close()

	items := make([]service.AdminAPIKeyListItem, 0, params.Limit())
	for rows.Next() {
		var item service.AdminAPIKeyListItem
		item.APIKey.User = &service.User{}
		var groupID sql.NullInt64
		var lastUsedAt, expiresAt sql.NullTime
		var groupName, groupPlatform sql.NullString
		var groupRate sql.NullFloat64
		if err := rows.Scan(
			&item.APIKey.ID, &item.APIKey.UserID, &item.APIKey.Key, &item.APIKey.Name, &groupID,
			&item.APIKey.Status, &lastUsedAt, &item.APIKey.Quota, &item.APIKey.QuotaUsed, &expiresAt,
			&item.APIKey.CreatedAt, &item.APIKey.UpdatedAt,
			&item.APIKey.User.Email, &item.APIKey.User.Username, &item.APIKey.User.Balance,
			&groupName, &groupPlatform, &groupRate,
			&item.TodayActualCost, &item.Last30DaysActualCost, &item.TotalActualCost,
		); err != nil {
			return nil, fmt.Errorf("scan admin API key: %w", err)
		}
		if groupID.Valid {
			item.APIKey.GroupID = &groupID.Int64
			item.APIKey.GroupIDs = []int64{groupID.Int64}
			item.APIKey.Group = &service.Group{
				ID: groupID.Int64, Name: groupName.String, Platform: groupPlatform.String,
				RateMultiplier: groupRate.Float64, Hydrated: true,
			}
		}
		item.APIKey.User.ID = item.APIKey.UserID
		if lastUsedAt.Valid {
			item.APIKey.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			item.APIKey.ExpiresAt = &expiresAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin API keys: %w", err)
	}
	return items, nil
}

var _ service.AdminAPIKeyListRepository = (*apiKeyRepository)(nil)
