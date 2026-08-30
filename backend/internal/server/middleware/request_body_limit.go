package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequestBodyLimit 使用 MaxBytesReader 限制请求体大小。
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// StrictRequestBodyLimit buffers a small public request body up to maxBytes and
// rejects overflow before the handler runs. This is intended for JSON control
// endpoints such as login and payment recovery where handlers otherwise may
// flatten http.MaxBytesError into a generic 400 response. Reading maxBytes+1
// also makes the 413 behavior deterministic for chunked requests.
func StrictRequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		originalBody := c.Request.Body
		body, err := io.ReadAll(io.LimitReader(originalBody, maxBytes+1))
		_ = originalBody.Close()
		if err != nil {
			response.ErrorWithDetails(
				c,
				http.StatusBadRequest,
				"Failed to read request body",
				"REQUEST_BODY_READ_FAILED",
				nil,
			)
			c.Abort()
			return
		}
		if int64(len(body)) > maxBytes {
			response.ErrorWithDetails(
				c,
				http.StatusRequestEntityTooLarge,
				"Request body too large",
				"REQUEST_BODY_TOO_LARGE",
				map[string]string{"limit_bytes": strconv.FormatInt(maxBytes, 10)},
			)
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Request.TransferEncoding = nil
		c.Next()
	}
}
