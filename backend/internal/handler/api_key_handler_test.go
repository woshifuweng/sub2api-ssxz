//go:build unit

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

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
