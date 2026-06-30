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

func setupUserAPIKeyCreateTestRouter(repo *apiKeyCreateReplayRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1, Concurrency: 1})
		c.Next()
	})

	userRepo := newStubUserRepoForHandler()
	userRepo.users[1] = &service.User{ID: 1}
	apiKeyService := service.NewAPIKeyService(repo, userRepo, nil, nil, nil, nil, &config.Config{})
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

func TestAPIKeyHandler_Create_IdempotencyReplayMasksFullKey(t *testing.T) {
	repo := newAPIKeyCreateReplayRepo()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newUserMemoryIdempotencyRepoStub(), nil, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})
	router := setupUserAPIKeyCreateTestRouter(repo)

	const plaintextKey = "sk-created-full-key-only-shown-once"
	body := `{"name":"client-key","custom_key":"` + plaintextKey + `"}`
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
