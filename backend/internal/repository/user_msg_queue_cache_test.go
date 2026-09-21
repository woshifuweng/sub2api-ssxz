package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Reject commands that miniredis accepts but Redis 6.0 does not.
type redis60QueueSessionHook struct{}

func (redis60QueueSessionHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func redis60QueueSessionCommandError(cmd redis.Cmder) error {
	if cmd.Name() == "getdel" {
		return errors.New("ERR unknown command 'getdel' (Redis 6.0)")
	}
	if cmd.Name() == "zrange" {
		for _, arg := range cmd.Args()[4:] {
			switch strings.ToLower(fmt.Sprint(arg)) {
			case "byscore", "rev":
				return errors.New("ERR syntax error (Redis 6.0 ZRANGE)")
			}
		}
	}
	return nil
}

func (redis60QueueSessionHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := redis60QueueSessionCommandError(cmd); err != nil {
			return err
		}
		return next(ctx, cmd)
	}
}

func (redis60QueueSessionHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			if err := redis60QueueSessionCommandError(cmd); err != nil {
				cmd.SetErr(err)
				return err
			}
		}
		return next(ctx, cmds)
	}
}

func newRedis60QueueSessionClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	srv := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	client.AddHook(redis60QueueSessionHook{})
	return srv, client
}

func TestUserMsgQueueReconcileSupportsRedis60(t *testing.T) {
	srv, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	cache := NewUserMsgQueueCache(client)
	now := time.Unix(1700000000, 0)
	srv.SetTime(now)

	acquired, err := cache.AcquireLock(ctx, 1, "expired", 1000)
	require.NoError(t, err)
	require.True(t, acquired)
	srv.FastForward(2 * time.Second)
	now = now.Add(2 * time.Second)
	srv.SetTime(now)

	acquired, err = cache.AcquireLock(ctx, 2, "live", 60000)
	require.NoError(t, err)
	require.True(t, acquired)
	// Simulate a renewed lock whose index update was lost, plus an abnormal
	// lock with no TTL and a future index entry that must not be visited.
	require.NoError(t, client.Set(ctx, umqLockKey(3), "no-ttl", 0).Err())
	require.NoError(t, client.ZAdd(ctx, umqLockIndexKey,
		redis.Z{Score: float64(now.UnixMilli()), Member: "2"},
		redis.Z{Score: float64(now.UnixMilli()), Member: "3"},
		redis.Z{Score: float64(now.Add(time.Hour).UnixMilli()), Member: "4"},
	).Err())

	cleaned, err := cache.ReconcileExpiredLockCandidates(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, 1, cleaned, "only the no-TTL lock counts as deleted")
	require.False(t, srv.Exists(umqLockKey(1)))
	require.False(t, srv.Exists(umqLockKey(3)))
	require.Equal(t, []string{"2", "4"}, client.ZRange(ctx, umqLockIndexKey, 0, -1).Val())
	require.Equal(t, "live", client.Get(ctx, umqLockKey(2)).Val())
	require.Equal(t, time.Minute, client.PTTL(ctx, umqLockKey(2)).Val())
	require.Equal(t, float64(now.Add(time.Minute).UnixMilli()), client.ZScore(ctx, umqLockIndexKey, "2").Val())
	acquired, err = cache.AcquireLock(ctx, 2, "other", 60000)
	require.NoError(t, err)
	require.False(t, acquired, "reconciliation must not release a live lock")
}

func TestUserMsgQueueReconcileRedis60ScoreAndCount(t *testing.T) {
	srv, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	cache := NewUserMsgQueueCache(client)
	now := time.Unix(1700000000, 0)
	srv.SetTime(now)
	for i, member := range []string{"1", "2", "3"} {
		require.NoError(t, client.Set(ctx, umqLockKey(int64(i+1)), "no-ttl", 0).Err())
		require.NoError(t, client.ZAdd(ctx, umqLockIndexKey, redis.Z{
			Score: float64(now.UnixMilli() + int64(i) - 1), Member: member,
		}).Err())
	}
	for _, remaining := range [][]string{{"2", "3"}, {"3"}} {
		cleaned, err := cache.ReconcileExpiredLockCandidates(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, 1, cleaned)
		require.Equal(t, remaining, client.ZRange(ctx, umqLockIndexKey, 0, -1).Val())
	}
	cleaned, err := cache.ReconcileExpiredLockCandidates(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, cleaned)
	require.True(t, srv.Exists(umqLockKey(3)), "future candidates must remain untouched")
}

type queueReconcileTimeAdvanceHook struct {
	redis60QueueSessionHook
	before func(redis.Cmder)
}

func (h queueReconcileTimeAdvanceHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.before(cmd)
		return next(ctx, cmd)
	}
}

func TestUserMsgQueueReconcileRedis60TimeAdvancesWithinBatch(t *testing.T) {
	srv, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	cache := NewUserMsgQueueCache(client)
	now := time.Unix(1700000000, 0)
	srv.SetTime(now)
	for _, id := range []int64{1, 2} {
		acquired, err := cache.AcquireLock(ctx, id, "live", 60000)
		require.NoError(t, err)
		require.True(t, acquired)
		require.NoError(t, client.ZAdd(ctx, umqLockIndexKey, redis.Z{
			Score: float64(now.UnixMilli()), Member: fmt.Sprint(id),
		}).Err())
	}

	// Move both Redis TIME and TTLs before each candidate is checked, including
	// time spent since the batch cutoff. NOSCRIPT fallback must not tick twice.
	checked := make(map[string]bool)
	clock := now
	client.AddHook(queueReconcileTimeAdvanceHook{before: func(cmd redis.Cmder) {
		if (cmd.Name() != "evalsha" && cmd.Name() != "eval") || len(cmd.Args()) < 4 {
			return
		}
		key := fmt.Sprint(cmd.Args()[3])
		if checked[key] {
			return
		}
		checked[key] = true
		clock = clock.Add(5 * time.Second)
		srv.FastForward(5 * time.Second)
		srv.SetTime(clock)
	}})

	cleaned, err := cache.ReconcileExpiredLockCandidates(ctx, 2)
	require.NoError(t, err)
	require.Zero(t, cleaned)
	require.Len(t, checked, 2)
	for _, id := range []int64{1, 2} {
		score, err := client.ZScore(ctx, umqLockIndexKey, fmt.Sprint(id)).Result()
		require.NoError(t, err)
		require.Equal(t, float64(now.Add(time.Minute).UnixMilli()), score,
			"batch latency must not move the actual lock expiry earlier")
		require.Equal(t, 50*time.Second, client.PTTL(ctx, umqLockKey(id)).Val())
		require.Equal(t, "live", client.Get(ctx, umqLockKey(id)).Val())
	}
}
