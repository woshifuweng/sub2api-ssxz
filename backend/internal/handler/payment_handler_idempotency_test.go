package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentHandlerCreateOrderIdempotencyReplayDoesNotCreateDuplicateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := service.DefaultIdempotencyConfig()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newUserMemoryIdempotencyRepoStub(), nil, cfg))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	var calls atomic.Int32
	h := &PaymentHandler{
		createOrderFn: func(_ context.Context, req service.CreateOrderRequest) (*service.CreateOrderResponse, error) {
			calls.Add(1)
			return &service.CreateOrderResponse{
				OrderID:     77,
				Amount:      req.Amount,
				PayAmount:   req.Amount,
				FeeRate:     0,
				Status:      payment.OrderStatusPending,
				PaymentType: req.PaymentType,
				OutTradeNo:  "payment-idempotent-order",
				ExpiresAt:   time.Now().Add(15 * time.Minute),
			}, nil
		},
	}

	router := gin.New()
	router.Use(withUserSubject(42))
	router.POST("/api/v1/payment/orders", func(c *gin.Context) {
		h.CreateOrderGateway(gatewayctx.FromGin(c))
	})

	body := `{"amount":10,"payment_type":"alipay","order_type":"balance","return_url":"/payment/result"}`
	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payment/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "payment-order-once")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := call()
	second := call()

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, int32(1), calls.Load(), "same payment order idempotency key should execute CreateOrder only once")
	require.Contains(t, second.Body.String(), "payment-idempotent-order")
}
