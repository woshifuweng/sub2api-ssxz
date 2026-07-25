package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardOperationsRepoStub struct {
	called bool
}

func (r *dashboardOperationsRepoStub) GetOperationsSummary(
	_ context.Context,
	_, _ time.Time,
	_ int,
) (*service.DashboardOperationsSummary, error) {
	r.called = true
	return &service.DashboardOperationsSummary{}, nil
}

func TestDashboardOperationsSummary_InvalidRangeReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &dashboardOperationsRepoStub{}
	handler := NewDashboardHandler(nil, nil)
	handler.SetOperationsService(service.NewDashboardOperationsService(repo))
	router := gin.New()
	router.GET("/admin/dashboard/operations-summary", handler.GetOperationsSummary)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/dashboard/operations-summary?start_date=2026-07-14&end_date=2026-07-01&timezone=Asia%2FShanghai",
		nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Invalid dashboard operations time range")
	require.False(t, repo.called)
}
