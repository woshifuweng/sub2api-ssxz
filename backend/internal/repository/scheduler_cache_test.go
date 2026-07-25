package repository

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSchedulerCacheForTest(t *testing.T) (*schedulerCache, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return NewSchedulerCache(client).(*schedulerCache), client
}

func publishSchedulerSnapshot(t *testing.T, cache *schedulerCache, bucket service.SchedulerBucket, accounts []service.Account) {
	t.Helper()
	token, err := cache.CaptureBucketWriteToken(context.Background(), bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(context.Background(), bucket, token, accounts))
}

func TestSchedulerCacheGetSnapshotUsesSlimMetadata(t *testing.T) {
	cache, client := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	accounts := []service.Account{
		{ID: 1, Name: "acc-1", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1}},
		{ID: 2, Name: "acc-2", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1}},
	}

	publishSchedulerSnapshot(t, cache, bucket, accounts)
	require.NoError(t, client.Del(ctx, schedulerAccountKey("1"), schedulerAccountKey("2")).Err())

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, int64(2), got[1].ID)
}

func TestSchedulerCacheSetSnapshotLargeBucketPreservesOrder(t *testing.T) {
	cache, _ := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformAnthropic, Mode: service.SchedulerModeMixed}
	accounts := make([]service.Account, 0, 1200)
	for i := 1; i <= 1200; i++ {
		accounts = append(accounts, service.Account{
			ID: int64(i), Name: "acc-" + strconv.Itoa(i), Platform: service.PlatformAnthropic,
			Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1},
		})
	}

	publishSchedulerSnapshot(t, cache, bucket, accounts)
	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 1200)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, int64(600), got[599].ID)
	require.Equal(t, int64(1200), got[1199].ID)
}

func TestSchedulerCacheUpdateLastUsedUpdatesSideKey(t *testing.T) {
	cache, _ := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	accounts := []service.Account{{
		ID: 11, Name: "acc-11", Platform: service.PlatformOpenAI,
		Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1},
	}}
	publishSchedulerSnapshot(t, cache, bucket, accounts)

	usedAt := time.Unix(1712345678, 0).UTC()
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{11: usedAt}))

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].LastUsedAt)
	require.Equal(t, usedAt, got[0].LastUsedAt.UTC())
}

func TestSchedulerCacheGetSnapshotMissesWhenMetadataIsMissing(t *testing.T) {
	cache, client := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 9, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	accounts := []service.Account{{
		ID: 101, Name: "missing-meta", Platform: service.PlatformOpenAI,
		Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{9},
	}}
	publishSchedulerSnapshot(t, cache, bucket, accounts)
	require.NoError(t, client.Del(ctx, schedulerAccountMetaKey("101")).Err())

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.False(t, hit)
	require.Nil(t, got)
}

func TestFilterSchedulerCredentialsKeepsSubscriptionPlanType(t *testing.T) {
	filtered := filterSchedulerCredentials(map[string]any{
		"plan_type": "plus", "access_token": "secret-access-token", "refresh_token": "secret-refresh-token",
	})
	require.Equal(t, "plus", filtered["plan_type"])
	require.NotContains(t, filtered, "access_token")
	require.NotContains(t, filtered, "refresh_token")
}

func TestSchedulerMetadataAccountKeepsOpenAISubscriptionIdentity(t *testing.T) {
	account := service.Account{
		ID: 24, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"plan_type": "plus", "access_token": "secret-access-token"},
	}
	metadata := buildSchedulerMetadataAccount(account)
	require.True(t, metadata.IsOpenAIChatGPTSubscription())
	require.Empty(t, metadata.GetCredential("access_token"))
}

func TestSchedulerMetadataAccountProjectsUpstreamBillingProbe(t *testing.T) {
	lastError := strings.Repeat("upstream diagnostic ", 512)
	probe := map[string]any{
		"status": "ok",
		"data": map[string]any{
			"billing_scope": "token", "resolved_rate_multiplier": 0.03, "peak_rate_enabled": true,
			"peak_start": "09:00", "peak_end": "18:00", "peak_rate_multiplier": 2.0,
			"timezone": "Asia/Shanghai", "effective_rate_multiplier": 0.03, "remote_diagnostic": lastError,
		},
		"received_at": "2026-07-13T10:00:00Z", "fresh_until": "2026-07-13T11:00:00Z",
		"next_probe_at": "2026-07-13T10:30:00Z", "http_status": 502, "last_error": lastError,
	}
	account := service.Account{ID: 42, Extra: map[string]any{"upstream_billing_probe": probe, "unused_large_field": "drop-me"}}

	metadata := buildSchedulerMetadataAccount(account)
	fullPayload, metaPayload, err := marshalSchedulerCacheAccount(account)
	require.NoError(t, err)
	filtered, ok := metadata.Extra["upstream_billing_probe"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ok", filtered["status"])
	require.NotContains(t, filtered, "http_status")
	require.NotContains(t, filtered, "last_error")
	filteredData, ok := filtered["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "token", filteredData["billing_scope"])
	require.NotContains(t, filteredData, "effective_rate_multiplier")
	require.NotContains(t, filteredData, "remote_diagnostic")
	require.NotContains(t, metadata.Extra, "unused_large_field")
	require.Contains(t, string(fullPayload), lastError)
	require.NotContains(t, string(metaPayload), "last_error")
	require.Less(t, len(metaPayload)*4, len(fullPayload))
}

func TestSchedulerMetadataAccountDropsInvalidUpstreamBillingProbe(t *testing.T) {
	for _, probe := range []any{"invalid", map[string]any{}, map[string]any{"status": ""}} {
		metadata := buildSchedulerMetadataAccount(service.Account{Extra: map[string]any{service.UpstreamBillingProbeExtraKey: probe}})
		require.NotContains(t, metadata.Extra, service.UpstreamBillingProbeExtraKey)
	}
}
