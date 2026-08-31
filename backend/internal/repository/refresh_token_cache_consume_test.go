//go:build unit

package repository

import (
	"context"
	"errors"
	"fmt"
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

func TestDeleteRefreshToken_RemovesSecondaryIndexes(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &refreshTokenCache{rdb: rdb}
	data := refreshTokenTestData(42, "family-single")

	require.NoError(t, cache.StoreRefreshToken(context.Background(), "single-hash", data, time.Hour))
	require.NoError(t, cache.DeleteRefreshToken(context.Background(), "single-hash"))

	assertRefreshTokenAbsent(t, cache, "single-hash")
	assertRefreshTokenIndexesEmpty(t, cache, data.UserID, data.FamilyID)
}

func TestDeleteUserRefreshTokens_IsAtomicAgainstRotation(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &refreshTokenCache{rdb: rdb}

	for iteration := 0; iteration < 40; iteration++ {
		oldHash := fmt.Sprintf("user-old-%d", iteration)
		newHash := fmt.Sprintf("user-new-%d", iteration)
		data := refreshTokenTestData(100+int64(iteration), fmt.Sprintf("user-family-%d", iteration))
		require.NoError(t, cache.StoreRefreshToken(context.Background(), oldHash, data, time.Hour))

		start := make(chan struct{})
		var wg sync.WaitGroup
		var rotateErr error
		var deleteErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, rotateErr = cache.RotateRefreshToken(context.Background(), oldHash, newHash, data, time.Hour)
		}()
		go func() {
			defer wg.Done()
			<-start
			deleteErr = cache.DeleteUserRefreshTokens(context.Background(), data.UserID)
		}()
		close(start)
		wg.Wait()

		require.NoError(t, rotateErr)
		require.NoError(t, deleteErr)
		assertRefreshTokenAbsent(t, cache, oldHash)
		assertRefreshTokenAbsent(t, cache, newHash)
		assertRefreshTokenIndexesEmpty(t, cache, data.UserID, data.FamilyID)
	}
}

func TestDeleteTokenFamily_IsAtomicAgainstRotation(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &refreshTokenCache{rdb: rdb}

	for iteration := 0; iteration < 40; iteration++ {
		oldHash := fmt.Sprintf("family-old-%d", iteration)
		newHash := fmt.Sprintf("family-new-%d", iteration)
		data := refreshTokenTestData(200+int64(iteration), fmt.Sprintf("family-%d", iteration))
		require.NoError(t, cache.StoreRefreshToken(context.Background(), oldHash, data, time.Hour))

		start := make(chan struct{})
		var wg sync.WaitGroup
		var rotateErr error
		var deleteErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, rotateErr = cache.RotateRefreshToken(context.Background(), oldHash, newHash, data, time.Hour)
		}()
		go func() {
			defer wg.Done()
			<-start
			deleteErr = cache.DeleteTokenFamily(context.Background(), data.FamilyID)
		}()
		close(start)
		wg.Wait()

		require.NoError(t, rotateErr)
		require.NoError(t, deleteErr)
		assertRefreshTokenAbsent(t, cache, oldHash)
		assertRefreshTokenAbsent(t, cache, newHash)
		assertRefreshTokenIndexesEmpty(t, cache, data.UserID, data.FamilyID)
	}
}

func refreshTokenTestData(userID int64, familyID string) *service.RefreshTokenData {
	now := time.Now().UTC()
	return &service.RefreshTokenData{
		UserID:    userID,
		FamilyID:  familyID,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
}

func assertRefreshTokenAbsent(t *testing.T, cache *refreshTokenCache, tokenHash string) {
	t.Helper()
	_, err := cache.GetRefreshToken(context.Background(), tokenHash)
	require.ErrorIs(t, err, service.ErrRefreshTokenNotFound)
}

func assertRefreshTokenIndexesEmpty(t *testing.T, cache *refreshTokenCache, userID int64, familyID string) {
	t.Helper()
	userMembers, err := cache.GetUserTokenHashes(context.Background(), userID)
	require.NoError(t, err)
	require.Empty(t, userMembers)
	familyMembers, err := cache.GetFamilyTokenHashes(context.Background(), familyID)
	require.NoError(t, err)
	require.Empty(t, familyMembers)
}
