package repository

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Run only against a freshly started, non-persistent acceptance Redis, never
// production/staging. An exact run_id from INFO guards against port reuse.
func TestRedis60HandoffIntegration(t *testing.T) {
	runID := os.Getenv("SSXZ_REDIS6_ACCEPTANCE_RUN_ID")
	if runID == "" {
		t.Skip("requires a dedicated Redis 6.0 instance on port 16381")
	}
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:16381"})
	t.Cleanup(func() { _ = rdb.Close() })
	info, err := rdb.Info(ctx, "server").Result()
	require.NoError(t, err)
	require.Contains(t, info, "redis_version:6.0.")
	require.Contains(t, info, "run_id:"+runID+"\r\n")
	require.True(t, len(runID) == 40 && !strings.ContainsAny(runID, "\r\n"))
	require.EqualValues(t, 0, rdb.DBSize(ctx).Val(), "fresh isolated instance must be empty")

	store := NewPasskeySessionStore(rdb)
	token, err := store.Store(ctx, &service.PasskeySession{}, time.Minute)
	require.NoError(t, err)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Consume(ctx, token); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, successes.Load())
	_, err = store.Consume(ctx, token)
	require.ErrorIs(t, err, service.ErrPasskeySession)

	queue := NewUserMsgQueueCache(rdb)
	locked, err := queue.AcquireLock(ctx, 900000001, "acceptance-live", 60000)
	require.NoError(t, err)
	require.True(t, locked)
	now := time.Now().UnixMilli()
	require.NoError(t, rdb.ZAdd(ctx, umqLockIndexKey, redis.Z{Score: float64(now - 1), Member: "900000001"}, redis.Z{Score: float64(now - 1), Member: "900000002"}).Err())
	_, err = queue.ReconcileExpiredLockCandidates(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, "acceptance-live", rdb.Get(ctx, umqLockKey(900000001)).Val())
	require.Greater(t, rdb.ZScore(ctx, umqLockIndexKey, "900000001").Val(), float64(now))
	require.ErrorIs(t, rdb.ZScore(ctx, umqLockIndexKey, "900000002").Err(), redis.Nil)
}
