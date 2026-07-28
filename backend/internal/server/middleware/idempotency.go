package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyTTL     = 24 * time.Hour
	idempotencyBodyCap = 512 * 1024
)

// GatewayIdempotencyMiddleware replays completed billable POST responses for
// clients that retry the same request with an Idempotency-Key.
func GatewayIdempotencyMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil || c.Request.Method != http.MethodPost || !isIdempotencyBillingPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			idempotencyKey = c.GetHeader("X-Idempotency-Key")
		}
		if idempotencyKey == "" {
			c.Next()
			return
		}

		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey == nil || apiKey.ID <= 0 {
			c.Next()
			return
		}

		cacheKey := idempotencyCacheKey(apiKey.ID, idempotencyKey)
		cached, err := redisClient.HGetAll(c.Request.Context(), cacheKey).Result()
		if err != nil {
			logger.LegacyPrintf("middleware.idempotency", "[Idempotency] cache read failed for %s: %v", cacheKey, err)
		} else if len(cached) > 0 {
			if replayCachedResponse(c, cached) {
				return
			}
			logger.LegacyPrintf("middleware.idempotency", "[Idempotency] ignoring malformed cache entry %s", cacheKey)
		}

		capture := &idempotencyCaptureWriter{
			ResponseWriter: c.Writer,
			maxBodyBytes:  idempotencyBodyCap,
		}
		c.Writer = capture
		c.Next()
		c.Writer = capture.ResponseWriter

		if capture.tooLarge || !capture.Written() {
			if capture.tooLarge {
				logger.LegacyPrintf("middleware.idempotency", "[Idempotency] response exceeds %d byte cache cap for %s", idempotencyBodyCap, cacheKey)
			}
			return
		}

		contentType := capture.Header().Get("Content-Type")
		values := map[string]interface{}{
			"status_code":  strconv.Itoa(capture.Status()),
			"content_type": contentType,
			"body":         capture.body.Bytes(),
			"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		}
		pipe := redisClient.Pipeline()
		pipe.HSet(c.Request.Context(), cacheKey, values)
		pipe.Expire(c.Request.Context(), cacheKey, idempotencyTTL)
		if _, err := pipe.Exec(c.Request.Context()); err != nil {
			logger.LegacyPrintf("middleware.idempotency", "[Idempotency] cache write failed for %s: %v", cacheKey, err)
		}
	}
}

func isIdempotencyBillingPath(path string) bool {
	switch path {
	case "/v1/chat/completions", "/chat/completions",
		"/v1/messages", "/messages",
		"/v1/responses", "/responses",
		"/v1/images/generations", "/images/generations":
		return true
	default:
		return false
	}
}

func idempotencyCacheKey(apiKeyID int64, idempotencyKey string) string {
	digest := sha256.Sum256([]byte(idempotencyKey))
	return "idempotency:" + strconv.FormatInt(apiKeyID, 10) + ":" + hex.EncodeToString(digest[:])
}

func replayCachedResponse(c *gin.Context, cached map[string]string) bool {
	statusCode, err := strconv.Atoi(cached["status_code"])
	if err != nil || statusCode < 100 || statusCode > 599 {
		return false
	}
	contentType := cached["content_type"]
	if contentType == "" {
		contentType = "application/json"
	}
	c.Header("Content-Type", contentType)
	c.Header("Idempotency-Key-Replay", "true")
	c.Data(statusCode, contentType, []byte(cached["body"]))
	c.Abort()
	return true
}

type idempotencyCaptureWriter struct {
	gin.ResponseWriter
	body         bytes.Buffer
	maxBodyBytes int
	tooLarge     bool
	wroteHeader  bool
	status       int
}

func (w *idempotencyCaptureWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *idempotencyCaptureWriter) WriteHeaderNow() {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *idempotencyCaptureWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeaderNow()
	}
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *idempotencyCaptureWriter) WriteString(data string) (int, error) {
	if !w.wroteHeader {
		w.WriteHeaderNow()
	}
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *idempotencyCaptureWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *idempotencyCaptureWriter) Written() bool {
	return w.wroteHeader || w.ResponseWriter.Written()
}

func (w *idempotencyCaptureWriter) capture(data []byte) {
	if w.tooLarge {
		return
	}
	remaining := w.maxBodyBytes - w.body.Len()
	if len(data) > remaining {
		w.tooLarge = true
		return
	}
	_, _ = w.body.Write(data)
}
