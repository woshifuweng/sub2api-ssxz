//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func newIdempotencyTestRouter(t *testing.T, redisClient *redis.Client, processed *int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyAPIKey), &service.APIKey{ID: 42})
		c.Next()
	})
	router.Use(GatewayIdempotencyMiddleware(redisClient))
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		(*processed)++
		c.JSON(http.StatusOK, gin.H{"processed": *processed})
	})
	return router
}

func newIdempotencyTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	return server, redis.NewClient(&redis.Options{Addr: server.Addr()})
}

func TestIdempotency_FirstRequest_ProcessedNormally(t *testing.T) {
	server, redisClient := newIdempotencyTestRedis(t)
	defer redisClient.Close()
	defer server.Close()

	processed := 0
	router := newIdempotencyTestRouter(t, redisClient, &processed)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Idempotency-Key", "first-request")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK || processed != 1 {
		t.Fatalf("expected normal first request, status=%d processed=%d", response.Code, processed)
	}
	if response.Header().Get("Idempotency-Key-Replay") != "" {
		t.Fatal("first request must not be marked as replay")
	}
}

func TestIdempotency_SecondRequest_ReturnsCachedResponse_NoBilling(t *testing.T) {
	server, redisClient := newIdempotencyTestRedis(t)
	defer redisClient.Close()
	defer server.Close()

	processed := 0
	router := newIdempotencyTestRouter(t, redisClient, &processed)

	first := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	first.Header.Set("Idempotency-Key", "same-request")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, first)

	second := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	second.Header.Set("X-Idempotency-Key", "same-request")
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, second)

	if secondResponse.Code != http.StatusOK || processed != 1 {
		t.Fatalf("expected cached response without second billing, status=%d processed=%d", secondResponse.Code, processed)
	}
	if secondResponse.Header().Get("Idempotency-Key-Replay") != "true" {
		t.Fatal("second request must be marked as replay")
	}
	if secondResponse.Body.String() != firstResponse.Body.String() {
		t.Fatalf("cached body mismatch: first=%q second=%q", firstResponse.Body.String(), secondResponse.Body.String())
	}
}

func TestIdempotency_NoHeader_PassThrough(t *testing.T) {
	server, redisClient := newIdempotencyTestRedis(t)
	defer redisClient.Close()
	defer server.Close()

	processed := 0
	router := newIdempotencyTestRouter(t, redisClient, &processed)
	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
	}
	if processed != 2 {
		t.Fatalf("expected requests without header to pass through, processed=%d", processed)
	}
}

func TestIdempotency_DifferentKeys_BothProcessed(t *testing.T) {
	server, redisClient := newIdempotencyTestRedis(t)
	defer redisClient.Close()
	defer server.Close()

	processed := 0
	router := newIdempotencyTestRouter(t, redisClient, &processed)
	for _, key := range []string{"request-a", "request-b"} {
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		request.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
	}
	if processed != 2 {
		t.Fatalf("expected different keys to process independently, processed=%d", processed)
	}
}
