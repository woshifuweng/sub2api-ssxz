package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAPIKeyHandler(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	h := NewAdminAPIKeyHandler(adminSvc)
	router.GET("/api/v1/admin/api-keys", h.List)
	router.PUT("/api/v1/admin/api-keys/:id", h.UpdateGroup)
	router.PATCH("/api/v1/admin/api-keys/:id/status", h.SetEnabled)
	router.DELETE("/api/v1/admin/api-keys/:id", h.Delete)
	return router
}

func TestAdminAPIKeyHandler_ListMasksSecretsAndForwardsFilters(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	groupID := int64(12)
	rawKey := "sk-admin-inventory-secret-1234567890"
	svc := newStubAdminService()
	svc.adminAPIKeyList = &service.AdminAPIKeyListResult{
		Items: []service.AdminAPIKeyListItem{{
			APIKey: service.APIKey{
				ID: 41, UserID: 9, Key: rawKey, Name: "customer-key", GroupID: &groupID,
				Status: service.StatusAPIKeyActive, Quota: 10, QuotaUsed: 1.25,
				LastUsedAt: &now, CreatedAt: now, UpdatedAt: now,
				User:  &service.User{ID: 9, Email: "customer@example.com", Username: "customer", Balance: 8.75},
				Group: &service.Group{ID: groupID, Name: "Claude CCMAX", Platform: service.PlatformAnthropic, RateMultiplier: 1.2},
			},
			TodayActualCost: 0.3, Last30DaysActualCost: 2.4, TotalActualCost: 6.8,
		}},
		Pagination: pagination.PaginationResult{Total: 1, Page: 2, PageSize: 25, Pages: 1},
		Summary:    service.AdminAPIKeyListSummary{Total: 1, Active: 1, Last30DaysActualCost: 2.4},
	}
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys?page=2&page_size=25&search=customer&user_id=9&group_id=12&status=active&sort_by=total_actual_cost&sort_order=asc", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), rawKey)
	require.Contains(t, rec.Body.String(), "sk-admin...7890")
	require.Equal(t, "customer", svc.adminAPIKeyListParams.Filters.Search)
	require.Equal(t, int64(9), *svc.adminAPIKeyListParams.Filters.UserID)
	require.Equal(t, int64(12), *svc.adminAPIKeyListParams.Filters.GroupID)
	require.Equal(t, "active", svc.adminAPIKeyListParams.Filters.Status)
	require.Equal(t, service.AdminAPIKeySortTotalActualCost, svc.adminAPIKeyListParams.Pagination.SortBy)
	require.Equal(t, "asc", svc.adminAPIKeyListParams.Pagination.SortOrder)
	require.Contains(t, rec.Body.String(), `"last_30_days_actual_cost":2.4`)
}

func TestAdminAPIKeyHandler_ListRejectsInvalidFilters(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())

	for _, target := range []string{
		"/api/v1/admin/api-keys?user_id=bad",
		"/api/v1/admin/api-keys?group_id=-1",
		"/api/v1/admin/api-keys?status=unknown",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, target)
	}
}

func TestAdminAPIKeyHandler_ListNormalizesUnsafeSort(t *testing.T) {
	svc := newStubAdminService()
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys?sort_by=created_at%20DESC%3BDELETE&sort_order=sideways", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.AdminAPIKeySortLastUsedAt, svc.adminAPIKeyListParams.Pagination.SortBy)
	require.Equal(t, "desc", svc.adminAPIKeyListParams.Pagination.SortOrder)
}

func TestAdminAPIKeyHandler_UpdateGroup_InvalidID(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())
	body := `{"group_id": 2}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid API key ID")
}

func TestAdminAPIKeyHandler_UpdateGroup_InvalidJSON(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid request")
}

func TestAdminAPIKeyHandler_UpdateGroupRejectsUnknownFields(t *testing.T) {
	svc := newStubAdminService()
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{"group_id":2,"quota":999}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, svc.adminAPIKeyMutations)
}

func TestAdminAPIKeyHandler_UpdateGroup_KeyNotFound(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())
	body := `{"group_id": 2}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	// ErrAPIKeyNotFound maps to 404
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminAPIKeyHandler_UpdateGroup_BindGroup(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())
	body := `{"group_id": 2}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var data struct {
		APIKey struct {
			ID      int64  `json:"id"`
			GroupID *int64 `json:"group_id"`
		} `json:"api_key"`
		AutoGrantedGroupAccess bool `json:"auto_granted_group_access"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.Equal(t, int64(10), data.APIKey.ID)
	require.NotNil(t, data.APIKey.GroupID)
	require.Equal(t, int64(2), *data.APIKey.GroupID)
}

func TestAdminAPIKeyHandler_UpdateGroup_Unbind(t *testing.T) {
	svc := newStubAdminService()
	gid := int64(2)
	svc.apiKeys[0].GroupID = &gid
	router := setupAPIKeyHandler(svc)
	body := `{"group_id": 0}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			APIKey struct {
				GroupID *int64 `json:"group_id"`
			} `json:"api_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Data.APIKey.GroupID)
}

func TestAdminAPIKeyHandler_ResetRateLimitUsage(t *testing.T) {
	svc := newStubAdminService()
	now := time.Now()
	svc.apiKeys[0].Usage5h = 1.2
	svc.apiKeys[0].Usage1d = 3.4
	svc.apiKeys[0].Usage7d = 5.6
	svc.apiKeys[0].Window5hStart = &now
	svc.apiKeys[0].Window1dStart = &now
	svc.apiKeys[0].Window7dStart = &now
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{"reset_rate_limit_usage":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			APIKey struct {
				Usage5h       float64    `json:"usage_5h"`
				Usage1d       float64    `json:"usage_1d"`
				Usage7d       float64    `json:"usage_7d"`
				Window5hStart *time.Time `json:"window_5h_start"`
				Window1dStart *time.Time `json:"window_1d_start"`
				Window7dStart *time.Time `json:"window_7d_start"`
			} `json:"api_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Zero(t, resp.Data.APIKey.Usage5h)
	require.Zero(t, resp.Data.APIKey.Usage1d)
	require.Zero(t, resp.Data.APIKey.Usage7d)
	require.Nil(t, resp.Data.APIKey.Window5hStart)
	require.Nil(t, resp.Data.APIKey.Window1dStart)
	require.Nil(t, resp.Data.APIKey.Window7dStart)
}

func TestAdminAPIKeyHandler_UpdateGroup_ServiceError(t *testing.T) {
	svc := &failingUpdateGroupService{
		stubAdminService: newStubAdminService(),
		err:              errors.New("internal failure"),
	}
	router := setupAPIKeyHandler(svc)
	body := `{"group_id": 2}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// H2: empty body → group_id is nil → no-op, returns original key
func TestAdminAPIKeyHandler_UpdateGroup_EmptyBodyRejected(t *testing.T) {
	router := setupAPIKeyHandler(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminAPIKeyHandler_SetEnabledRejectsPrivilegeFields(t *testing.T) {
	svc := newStubAdminService()
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/10/status", bytes.NewBufferString(`{"enabled":false,"user_id":99}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, svc.adminAPIKeyMutations)
}

func TestAdminAPIKeyHandler_SetEnabledUsesAuthenticatedAdminIdentity(t *testing.T) {
	svc := newStubAdminService()
	svc.apiKeys[0].Key = "sk-admin-response-must-stay-masked-7890"
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/10/status", bytes.NewBufferString(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.adminAPIKeyActorID)
	require.Equal(t, service.StatusAPIKeyDisabled, svc.apiKeys[0].Status)
	require.NotContains(t, rec.Body.String(), "sk-admin-response-must-stay-masked-7890")
	require.Contains(t, rec.Body.String(), "sk-admin...7890")
}

func TestAdminAPIKeyHandler_DeleteUsesAuthenticatedAdminIdentity(t *testing.T) {
	svc := newStubAdminService()
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/10", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.adminAPIKeyActorID)
	require.Empty(t, svc.apiKeys)
}

// M2: service returns GROUP_NOT_ACTIVE → handler maps to 400
func TestAdminAPIKeyHandler_UpdateGroup_GroupNotActive(t *testing.T) {
	svc := &failingUpdateGroupService{
		stubAdminService: newStubAdminService(),
		err:              infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active"),
	}
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{"group_id": 5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "GROUP_NOT_ACTIVE")
}

// M2: service returns INVALID_GROUP_ID → handler maps to 400
func TestAdminAPIKeyHandler_UpdateGroup_NegativeGroupID(t *testing.T) {
	svc := &failingUpdateGroupService{
		stubAdminService: newStubAdminService(),
		err:              infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative"),
	}
	router := setupAPIKeyHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10", bytes.NewBufferString(`{"group_id": -5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "INVALID_GROUP_ID")
}

// failingUpdateGroupService overrides AdminUpdateAPIKeyGroupID to return an error.
type failingUpdateGroupService struct {
	*stubAdminService
	err error
}

func (f *failingUpdateGroupService) AdminUpdateAPIKeyGroupID(_ context.Context, _ int64, _ *int64) (*service.AdminUpdateAPIKeyGroupIDResult, error) {
	return nil, f.err
}

func (f *failingUpdateGroupService) AdminChangeAPIKeyGroup(_ context.Context, _, _, _ int64) (*service.AdminUpdateAPIKeyGroupIDResult, error) {
	return nil, f.err
}
