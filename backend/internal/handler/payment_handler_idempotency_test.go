package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentCreateOrderRequiresIdempotencyKeyBeforeCallingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(withUserSubject(42))
	router.POST("/api/v1/payment/orders", (&PaymentHandler{}).CreateOrder)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment/orders", strings.NewReader(`{
		"amount": 10,
		"payment_type": "alipay",
		"order_type": "balance"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "IDEMPOTENCY_KEY_REQUIRED")
}
