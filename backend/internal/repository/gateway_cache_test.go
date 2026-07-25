package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newGatewayCacheForTest(t *testing.T) (*gatewayCache, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return NewGatewayCache(client).(*gatewayCache), client, mr
}

func TestGatewayCacheSessionLifecycleUsesGroupScopedRedisKey(t *testing.T) {
	cache, client, _ := newGatewayCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSessionAccountID(ctx, 7, "session-a", 42, time.Minute))
	require.NoError(t, cache.SetSessionAccountID(ctx, 8, "session-a", 99, time.Minute))

	accountID, err := cache.GetSessionAccountID(ctx, 7, "session-a")
	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)

	otherGroupID, err := cache.GetSessionAccountID(ctx, 8, "session-a")
	require.NoError(t, err)
	require.Equal(t, int64(99), otherGroupID)
	require.Equal(t, "42", client.Get(ctx, buildSessionKey(7, "session-a")).Val())
}

func TestGatewayCacheRefreshSessionTTLExtendsRedisEntry(t *testing.T) {
	cache, client, mr := newGatewayCacheForTest(t)
	ctx := context.Background()
	key := buildSessionKey(1, "session-b")

	require.NoError(t, cache.SetSessionAccountID(ctx, 1, "session-b", 7, 10*time.Second))
	mr.FastForward(6 * time.Second)
	require.NoError(t, cache.RefreshSessionTTL(ctx, 1, "session-b", 20*time.Second))

	ttl, err := client.TTL(ctx, key).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, 19*time.Second)
	mr.FastForward(11 * time.Second)
	accountID, err := cache.GetSessionAccountID(ctx, 1, "session-b")
	require.NoError(t, err)
	require.Equal(t, int64(7), accountID)
}

func TestGatewayCacheDeleteSessionAccountIDRemovesBinding(t *testing.T) {
	cache, _, _ := newGatewayCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSessionAccountID(ctx, 1, "session-c", 99, time.Minute))
	require.NoError(t, cache.DeleteSessionAccountID(ctx, 1, "session-c"))

	_, err := cache.GetSessionAccountID(ctx, 1, "session-c")
	require.ErrorIs(t, err, redis.Nil)
}

func TestGatewayCacheCyberSessionBlockExpires(t *testing.T) {
	cache, _, mr := newGatewayCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, cache.SetCyberSessionBlocked(ctx, "blocked-session", 5*time.Second))
	blocked, err := cache.IsCyberSessionBlocked(ctx, "blocked-session")
	require.NoError(t, err)
	require.True(t, blocked)

	mr.FastForward(6 * time.Second)
	blocked, err = cache.IsCyberSessionBlocked(ctx, "blocked-session")
	require.NoError(t, err)
	require.False(t, blocked)
}
