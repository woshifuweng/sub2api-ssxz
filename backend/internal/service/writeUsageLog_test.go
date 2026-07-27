//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type usageLogSyncFallbackRepoStub struct {
	UsageLogRepository

	bestEffortErr error
	createErr     error
	createCalls   int
}

func (r *usageLogSyncFallbackRepoStub) CreateBestEffort(_ context.Context, _ *UsageLog) error {
	return r.bestEffortErr
}

func (r *usageLogSyncFallbackRepoStub) Create(_ context.Context, _ *UsageLog) (bool, error) {
	r.createCalls++
	return r.createErr == nil, r.createErr
}

func TestWriteUsageLogBestEffort_DroppedTriesSyncFallback(t *testing.T) {
	repo := &usageLogSyncFallbackRepoStub{
		bestEffortErr: MarkUsageLogCreateDropped(errors.New("queue full")),
	}

	writeUsageLogBestEffort(context.Background(), repo, &UsageLog{
		UserID:    42,
		RequestID: "request-dropped",
	}, "test.usage_log")

	require.Equal(t, 1, repo.createCalls)
}

func TestWriteUsageLogBestEffort_DroppedAndSyncFails_NoError(t *testing.T) {
	repo := &usageLogSyncFallbackRepoStub{
		bestEffortErr: MarkUsageLogCreateDropped(errors.New("queue full")),
		createErr:     errors.New("database unavailable"),
	}

	require.NotPanics(t, func() {
		writeUsageLogBestEffort(context.Background(), repo, &UsageLog{
			UserID:    42,
			RequestID: "request-sync-failed",
		}, "test.usage_log")
	})
	require.Equal(t, 1, repo.createCalls)
}
