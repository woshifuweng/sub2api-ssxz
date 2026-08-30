package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type AdminAPIKeyListUser struct {
	ID       int64   `json:"id"`
	Email    string  `json:"email"`
	Username string  `json:"username"`
	Balance  float64 `json:"balance"`
}

type AdminAPIKeyListGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type AdminAPIKeyListItem struct {
	ID                   int64                 `json:"id"`
	User                 AdminAPIKeyListUser   `json:"user"`
	Key                  string                `json:"key"`
	Name                 string                `json:"name"`
	Group                *AdminAPIKeyListGroup `json:"group,omitempty"`
	Status               string                `json:"status"`
	Quota                float64               `json:"quota"`
	QuotaUsed            float64               `json:"quota_used"`
	LastUsedAt           *time.Time            `json:"last_used_at"`
	ExpiresAt            *time.Time            `json:"expires_at"`
	CreatedAt            time.Time             `json:"created_at"`
	TodayActualCost      float64               `json:"today_actual_cost"`
	Last30DaysActualCost float64               `json:"last_30_days_actual_cost"`
	TotalActualCost      float64               `json:"total_actual_cost"`
}

type AdminAPIKeyListSummary struct {
	Total                int64   `json:"total"`
	Active               int64   `json:"active"`
	Inactive             int64   `json:"inactive"`
	Expired              int64   `json:"expired"`
	Last30DaysActualCost float64 `json:"last_30_days_actual_cost"`
}

func AdminAPIKeyListItemFromService(item service.AdminAPIKeyListItem, now time.Time) AdminAPIKeyListItem {
	key := item.APIKey
	status := normalizeUserAPIKeyStatus(key.Status)
	if key.ExpiresAt != nil && !key.ExpiresAt.After(now) {
		status = service.StatusAPIKeyExpired
	}

	out := AdminAPIKeyListItem{
		ID: key.ID, Key: maskAPIKey(key.Key), Name: key.Name, Status: status,
		Quota: key.Quota, QuotaUsed: key.QuotaUsed, LastUsedAt: key.LastUsedAt,
		ExpiresAt: key.ExpiresAt, CreatedAt: key.CreatedAt,
		TodayActualCost:      item.TodayActualCost,
		Last30DaysActualCost: item.Last30DaysActualCost,
		TotalActualCost:      item.TotalActualCost,
	}
	if key.User != nil {
		out.User = AdminAPIKeyListUser{
			ID: key.User.ID, Email: key.User.Email, Username: key.User.Username, Balance: key.User.Balance,
		}
	} else {
		out.User.ID = key.UserID
	}
	if key.Group != nil {
		out.Group = &AdminAPIKeyListGroup{
			ID: key.Group.ID, Name: key.Group.Name, Platform: key.Group.Platform,
			RateMultiplier: key.Group.RateMultiplier,
		}
	}
	return out
}

func normalizeUserAPIKeyStatus(status string) string {
	if status == service.StatusAPIKeyDisabled {
		return "inactive"
	}
	return status
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 12 {
		return "[redacted]"
	}
	return key[:8] + "..." + key[len(key)-4:]
}
