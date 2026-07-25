package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dashboardOperationsRepositoryStub struct {
	limit int
}

func (s *dashboardOperationsRepositoryStub) GetOperationsSummary(_ context.Context, startTime, endTime time.Time, topLimit int) (*DashboardOperationsSummary, error) {
	s.limit = topLimit
	return &DashboardOperationsSummary{
		StartDate:    startTime,
		EndDate:      endTime,
		TopCustomers: []DashboardOperationsTopCustomer{},
	}, nil
}

func TestDashboardOperationsService_GetSummaryValidatesRangeAndBoundsLimit(t *testing.T) {
	repo := &dashboardOperationsRepositoryStub{}
	svc := NewDashboardOperationsService(repo)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	_, err := svc.GetSummary(context.Background(), end, start, 10)
	require.ErrorIs(t, err, ErrInvalidDashboardOperationsRange)

	_, err = svc.GetSummary(context.Background(), start, end, 0)
	require.NoError(t, err)
	require.Equal(t, 10, repo.limit)

	_, err = svc.GetSummary(context.Background(), start, end, 999)
	require.NoError(t, err)
	require.Equal(t, 50, repo.limit)
}
