//go:build unit

package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestRotateRefreshToken_HasSingleWinnerUnderConcurrency(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &refreshTokenCache{rdb: rdb}

	oldData := &service.RefreshTokenData{
		UserID:    42,
		FamilyID:  "family-1",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	require.NoError(t, cache.StoreRefreshToken(context.Background(), "old-hash", oldData, time.Hour))
	userMembers, err := cache.GetUserTokenHashes(context.Background(), oldData.UserID)
	require.NoError(t, err)
	require.Equal(t, []string{"old-hash"}, userMembers)
	familyMembers, err := cache.GetFamilyTokenHashes(context.Background(), oldData.FamilyID)
	require.NoError(t, err)
	require.Equal(t, []string{"old-hash"}, familyMembers)

	const workers = 16
	start := make(chan struct{})
	var successes atomic.Int64
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			newHash := "new-hash-" + string(rune('a'+worker))
			newData := *oldData
			newData.CreatedAt = time.Now().UTC()
			rotated, err := cache.RotateRefreshToken(context.Background(), "old-hash", newHash, &newData, time.Hour)
			require.NoError(t, err)
			if rotated {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Equal(t, int64(1), successes.Load())
	_, err = cache.GetRefreshToken(context.Background(), "old-hash")
	require.True(t, errors.Is(err, service.ErrRefreshTokenNotFound))
	familyMembers, err = cache.GetFamilyTokenHashes(context.Background(), oldData.FamilyID)
	require.NoError(t, err)
	require.Len(t, familyMembers, 1)
}
