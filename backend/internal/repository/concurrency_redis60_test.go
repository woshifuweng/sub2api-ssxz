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

type redis60RangeHook struct{}

func (redis60RangeHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (redis60RangeHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (redis60RangeHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "zrange" {
			for _, arg := range cmd.Args() {
				if strings.EqualFold(fmt.Sprint(arg), "byscore") {
					return errors.New("ERR value is not an integer or out of range (Redis 6.0)")
				}
			}
		}
		return next(ctx, cmd)
	}
}

func TestExpiredSlotCleanupSupportsRedis60(t *testing.T) {
	srv := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	client.AddHook(redis60RangeHook{})
	ctx := context.Background()
	cache := NewConcurrencyCache(client, 15, 900).(*concurrencyCache)
	for _, key := range []string{accountActiveIndexKey, userActiveIndexKey} {
		require.NoError(t, client.ZAdd(ctx, key,
			redis.Z{Score: float64(time.Now().Unix() - 10), Member: "999991"},
			redis.Z{Score: float64(time.Now().Unix() + 3600), Member: "999992"},
		).Err())
	}
	require.NoError(t, cache.CleanupExpiredAccountSlotKeys(ctx))
	for _, key := range []string{accountActiveIndexKey, userActiveIndexKey} {
		require.Equal(t, []string{"999992"}, client.ZRange(ctx, key, 0, -1).Val())
	}
}
