package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageOwnershipRepo struct {
	service.UsageLogRepository

	logs                map[int64]*service.UsageLog
	listCalled          bool
	listFilters         usagestats.UsageLogFilters
	apiKeyStatsCalled   bool
	batchAPIKeyUsageIDs []int64
}

func (r *usageOwnershipRepo) GetByID(_ context.Context, id int64) (*service.UsageLog, error) {
	if log, ok := r.logs[id]; ok {
		return log, nil
	}
	return nil, service.ErrUsageLogNotFound
}

func (r *usageOwnershipRepo) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	r.listCalled = true
	r.listFilters = filters
	return []service.UsageLog{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (r *usageOwnershipRepo) GetAPIKeyStatsAggregated(_ context.Context, apiKeyID int64, _ time.Time, _ time.Time) (*usagestats.UsageStats, error) {
	r.apiKeyStatsCalled = true
	return &usagestats.UsageStats{TotalRequests: 1, TotalActualCost: float64(apiKeyID)}, nil
}

func (r *usageOwnershipRepo) GetBatchAPIKeyUsageStats(_ context.Context, apiKeyIDs []int64, _ time.Time, _ time.Time) (map[int64]*usagestats.BatchAPIKeyUsageStats, error) {
	r.batchAPIKeyUsageIDs = append([]int64(nil), apiKeyIDs...)
	stats := make(map[int64]*usagestats.BatchAPIKeyUsageStats, len(apiKeyIDs))
	for _, id := range apiKeyIDs {
		stats[id] = &usagestats.BatchAPIKeyUsageStats{APIKeyID: id, TotalActualCost: float64(id)}
	}
	return stats, nil
}

type usageOwnershipAPIKeyRepo struct {
	service.APIKeyRepository

	keys         map[int64]*service.APIKey
	verifyUserID int64
	verifyKeyIDs []int64
}

func (r *usageOwnershipAPIKeyRepo) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	if key, ok := r.keys[id]; ok {
		return key, nil
	}
	return nil, service.ErrAPIKeyNotFound
}

func (r *usageOwnershipAPIKeyRepo) VerifyOwnership(_ context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	r.verifyUserID = userID
	r.verifyKeyIDs = append([]int64(nil), apiKeyIDs...)
	valid := make([]int64, 0, len(apiKeyIDs))
	for _, id := range apiKeyIDs {
		if key, ok := r.keys[id]; ok && key.UserID == userID {
			valid = append(valid, id)
		}
	}
	return valid, nil
}

func newUserUsageOwnershipTestRouter(usageRepo *usageOwnershipRepo, apiKeyRepo *usageOwnershipAPIKeyRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, &config.Config{})
	handler := NewUsageHandler(usageSvc, apiKeySvc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage", handler.List)
	router.GET("/usage/:id", handler.GetByID)
	router.GET("/usage/stats", handler.Stats)
	router.POST("/usage/dashboard/api-keys-usage", handler.DashboardAPIKeysUsage)
	return router
}

func TestUserUsageGetByIDRejectsOtherUsersRecord(t *testing.T) {
	usageRepo := &usageOwnershipRepo{
		logs: map[int64]*service.UsageLog{
			99: {
				ID:        99,
				UserID:    7,
				RequestID: "req_other_user_secret",
				Model:     "secret-model",
			},
		},
	}
	apiKeyRepo := &usageOwnershipAPIKeyRepo{keys: map[int64]*service.APIKey{}}
	router := newUserUsageOwnershipTestRouter(usageRepo, apiKeyRepo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/usage/99", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NotContains(t, rec.Body.String(), "req_other_user_secret")
	require.NotContains(t, rec.Body.String(), "secret-model")
}

func TestUserUsageListRejectsOtherUsersAPIKeyFilter(t *testing.T) {
	usageRepo := &usageOwnershipRepo{logs: map[int64]*service.UsageLog{}}
	apiKeyRepo := &usageOwnershipAPIKeyRepo{
		keys: map[int64]*service.APIKey{
			77: {
				ID:     77,
				UserID: 7,
				Key:    "sk_other_user_secret",
				Status: service.StatusAPIKeyActive,
			},
		},
	}
	router := newUserUsageOwnershipTestRouter(usageRepo, apiKeyRepo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/usage?api_key_id=77", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.False(t, usageRepo.listCalled)
	require.NotContains(t, rec.Body.String(), "sk_other_user_secret")
}

func TestUserUsageStatsRejectsOtherUsersAPIKeyFilter(t *testing.T) {
	usageRepo := &usageOwnershipRepo{logs: map[int64]*service.UsageLog{}}
	apiKeyRepo := &usageOwnershipAPIKeyRepo{
		keys: map[int64]*service.APIKey{
			77: {
				ID:     77,
				UserID: 7,
				Key:    "sk_other_user_secret",
				Status: service.StatusAPIKeyActive,
			},
		},
	}
	router := newUserUsageOwnershipTestRouter(usageRepo, apiKeyRepo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/usage/stats?api_key_id=77", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.False(t, usageRepo.apiKeyStatsCalled)
	require.NotContains(t, rec.Body.String(), "sk_other_user_secret")
}

func TestUserUsageDashboardAPIKeysUsageFiltersToOwnedKeys(t *testing.T) {
	usageRepo := &usageOwnershipRepo{logs: map[int64]*service.UsageLog{}}
	apiKeyRepo := &usageOwnershipAPIKeyRepo{
		keys: map[int64]*service.APIKey{
			101: {
				ID:     101,
				UserID: 42,
				Key:    "sk_owned_key",
				Status: service.StatusAPIKeyActive,
			},
			202: {
				ID:     202,
				UserID: 7,
				Key:    "sk_other_user_secret",
				Status: service.StatusAPIKeyActive,
			},
		},
	}
	router := newUserUsageOwnershipTestRouter(usageRepo, apiKeyRepo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/usage/dashboard/api-keys-usage", bytes.NewBufferString(`{"api_key_ids":[101,202]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), apiKeyRepo.verifyUserID)
	require.Equal(t, []int64{101, 202}, apiKeyRepo.verifyKeyIDs)
	require.Equal(t, []int64{101}, usageRepo.batchAPIKeyUsageIDs)
	require.Contains(t, rec.Body.String(), `"api_key_id":101`)
	require.NotContains(t, rec.Body.String(), `"api_key_id":202`)
	require.NotContains(t, rec.Body.String(), "sk_other_user_secret")
}
