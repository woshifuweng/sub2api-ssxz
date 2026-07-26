//go:build unit

package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupUserAPIKeyHandlerOwnershipTestRouter(repo *stubAPIKeyRepoForHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1, Concurrency: 1})
		c.Next()
	})

	h := NewAPIKeyHandler(newTestAPIKeyService(repo))
	router.GET("/api/v1/keys/:id", h.GetByID)
	router.POST("/api/v1/keys/:id/reveal", h.Reveal)
	router.PUT("/api/v1/keys/:id", h.Update)
	router.DELETE("/api/v1/keys/:id", h.Delete)
	return router
}

type apiKeyCreateReplayRepo struct {
	*stubAPIKeyRepoForHandler
	nextID  int64
	creates atomic.Int32
}

func newAPIKeyCreateReplayRepo() *apiKeyCreateReplayRepo {
	return &apiKeyCreateReplayRepo{
		stubAPIKeyRepoForHandler: newStubAPIKeyRepoForHandler(),
		nextID:                   100,
	}
}

func (r *apiKeyCreateReplayRepo) Create(_ context.Context, key *service.APIKey) error {
	id := r.nextID
	r.nextID++
	clone := *key
	clone.ID = id
	key.ID = id
	r.keys[id] = &clone
	r.creates.Add(1)
	return nil
}

type apiKeyCreateHandlerGroupRepo struct {
	group *service.Group
}

func (r *apiKeyCreateHandlerGroupRepo) GetByID(_ context.Context, id int64) (*service.Group, error) {
	if r.group == nil || r.group.ID != id {
		return nil, service.ErrGroupNotFound
	}
	out := *r.group
	return &out, nil
}

func (r *apiKeyCreateHandlerGroupRepo) Create(context.Context, *service.Group) error {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) GetByIDLite(context.Context, int64) (*service.Group, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) Update(context.Context, *service.Group) error {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) Delete(context.Context, int64) error { panic("unexpected") }
func (r *apiKeyCreateHandlerGroupRepo) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) ListActive(context.Context) ([]service.Group, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *apiKeyCreateHandlerGroupRepo) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	panic("unexpected")
}

func setupUserAPIKeyCreateTestRouter(repo *apiKeyCreateReplayRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1, Concurrency: 1})
		c.Next()
	})

	userRepo := newStubUserRepoForHandler()
	userRepo.users[1] = &service.User{ID: 1}
	groupRepo := &apiKeyCreateHandlerGroupRepo{group: &service.Group{
		ID:               10,
		Name:             "default",
		Platform:         service.PlatformOpenAI,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}}
	apiKeyService := service.NewAPIKeyService(repo, userRepo, groupRepo, nil, nil, nil, &config.Config{})
	h := NewAPIKeyHandler(apiKeyService)
	router.POST("/api/v1/keys", func(c *gin.Context) {
		h.CreateGateway(gatewayctx.FromGin(c))
	})
	return router
}

func TestAPIKeyHandler_GetByID_RejectsOtherUsersKey(t *testing.T) {
	repo := newStubAPIKeyRepoForHandler()
	repo.keys[42] = &service.APIKey{
		ID:     42,
		UserID: 2,
		Key:    "sk-other-user-key",
		Status: service.StatusAPIKeyActive,
	}
	router := setupUserAPIKeyHandlerOwnershipTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/42", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-other-user-key")
}

func TestAPIKeyHandler_Reveal_ReturnsOwnedPlaintextWithoutCaching(t *testing.T) {
	repo := newStubAPIKeyRepoForHandler()
	repo.keys[42] = &service.APIKey{
		ID:     42,
		UserID: 1,
		Key:    "sk-owned-full-key",
		Status: service.StatusAPIKeyActive,
	}
	router := setupUserAPIKeyHandlerOwnershipTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/42/reveal", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", rec.Header().Get("Pragma"))
	require.Contains(t, rec.Body.String(), "sk-owned-full-key")
}

func TestAPIKeyHandler_Reveal_RejectsOtherUsersKey(t *testing.T) {
	repo := newStubAPIKeyRepoForHandler()
	repo.keys[42] = &service.APIKey{
		ID:     42,
		UserID: 2,
		Key:    "sk-other-user-key",
		Status: service.StatusAPIKeyActive,
	}
	router := setupUserAPIKeyHandlerOwnershipTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/42/reveal", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-other-user-key")
}

func TestAPIKeyHandler_Create_IdempotencyReplayMasksFullKey(t *testing.T) {
	repo := newAPIKeyCreateReplayRepo()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newUserMemoryIdempotencyRepoStub(), nil, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})
	router := setupUserAPIKeyCreateTestRouter(repo)

	const plaintextKey = "sk-created-full-key-only-shown-once"
	body := `{"name":"client-key","group_id":10,"custom_key":"` + plaintextKey + `"}`
	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/keys", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "create-key-once")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := call()
	require.Equal(t, http.StatusOK, first.Code)
	require.Contains(t, first.Body.String(), plaintextKey)

	second := call()
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	require.NotContains(t, second.Body.String(), plaintextKey)
	require.Contains(t, second.Body.String(), "sk-creat...once")
	require.Equal(t, int32(1), repo.creates.Load())
	require.Len(t, repo.keys, 1)
}

func TestAPIKeyHandler_Update_RejectsOtherUsersKey(t *testing.T) {
	repo := newStubAPIKeyRepoForHandler()
	repo.keys[42] = &service.APIKey{
		ID:     42,
		UserID: 2,
		Key:    "sk-other-user-key",
		Status: service.StatusAPIKeyActive,
	}
	router := setupUserAPIKeyHandlerOwnershipTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/42", bytes.NewBufferString(`{"name":"stolen"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-other-user-key")
}

func TestAPIKeyHandler_Delete_RejectsOtherUsersKey(t *testing.T) {
	repo := newStubAPIKeyRepoForHandler()
	repo.keys[42] = &service.APIKey{
		ID:     42,
		UserID: 2,
		Key:    "sk-other-user-key",
		Status: service.StatusAPIKeyActive,
	}
	router := setupUserAPIKeyHandlerOwnershipTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/keys/42", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-other-user-key")
}

func (r *apiKeyCreateReplayRepo) DeleteWithAudit(context.Context, int64) error { return nil }
