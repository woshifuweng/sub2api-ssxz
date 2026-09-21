package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPasskeySessionConsumeSupportsRedis60(t *testing.T) {
	srv, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	store := NewPasskeySessionStore(client)
	session := &service.PasskeySession{Kind: "authentication", UserID: 42}

	t.Run("consume once and reject replay", func(t *testing.T) {
		token, err := store.Store(ctx, session, time.Minute)
		require.NoError(t, err)
		got, err := store.Consume(ctx, " "+token+" ")
		require.NoError(t, err)
		require.Equal(t, session, got)
		got, err = store.Consume(ctx, token)
		require.ErrorIs(t, err, service.ErrPasskeySession)
		require.Nil(t, got)
	})
	t.Run("missing", func(t *testing.T) {
		got, err := store.Consume(ctx, "missing")
		require.ErrorIs(t, err, service.ErrPasskeySession)
		require.Nil(t, got)
	})
	t.Run("expired", func(t *testing.T) {
		token, err := store.Store(ctx, session, time.Second)
		require.NoError(t, err)
		srv.FastForward(2 * time.Second)
		got, err := store.Consume(ctx, token)
		require.ErrorIs(t, err, service.ErrPasskeySession)
		require.Nil(t, got)
	})
}

func TestPasskeySessionConcurrentConsumeSupportsRedis60(t *testing.T) {
	_, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	store := NewPasskeySessionStore(client)
	session := &service.PasskeySession{Kind: "authentication", UserID: 42}
	token, err := store.Store(ctx, session, time.Minute)
	require.NoError(t, err)

	const consumers = 16
	type result struct {
		session *service.PasskeySession
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, consumers)
	for i := 0; i < consumers; i++ {
		go func() {
			<-start
			got, err := store.Consume(ctx, token)
			results <- result{session: got, err: err}
		}()
	}
	close(start)
	winners := 0
	for i := 0; i < consumers; i++ {
		got := <-results
		if got.err == nil {
			winners++
			require.Equal(t, session, got.session)
		} else {
			require.ErrorIs(t, got.err, service.ErrPasskeySession)
			require.Nil(t, got.session)
		}
	}
	require.Equal(t, 1, winners, "exactly one concurrent consumer may succeed")
	got, err := store.Consume(ctx, token)
	require.ErrorIs(t, err, service.ErrPasskeySession)
	require.Nil(t, got)
}

func TestPasskeySessionInvalidTokenSupportsRedis60(t *testing.T) {
	_, client := newRedis60QueueSessionClient(t)
	store := NewPasskeySessionStore(client)
	// Invalid input must be rejected before any Redis access.
	require.NoError(t, client.Close())
	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "whitespace", token: " \t\n "},
		{name: "too long", token: strings.Repeat("a", 129)},
		{name: "padded too long", token: " " + strings.Repeat("a", 129) + " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.Consume(context.Background(), tc.token)
			require.ErrorIs(t, err, service.ErrPasskeySession)
			require.Nil(t, got)
		})
	}
}

func TestPasskeySessionMalformedJSONSupportsRedis60(t *testing.T) {
	_, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	store := NewPasskeySessionStore(client)
	for _, payload := range []string{"{", `{"user_id":"not-a-number"}`} {
		t.Run(payload, func(t *testing.T) {
			const token = "malformed"
			key := passkeySessionPrefix + token
			require.NoError(t, client.Set(ctx, key, payload, time.Minute).Err())
			got, err := store.Consume(ctx, token)
			require.ErrorIs(t, err, service.ErrPasskeySession)
			require.Nil(t, got)
			require.ErrorIs(t, client.Get(ctx, key).Err(), redis.Nil,
				"invalid JSON must still be consumed, not left for replay")
			got, err = store.Consume(ctx, token)
			require.ErrorIs(t, err, service.ErrPasskeySession)
			require.Nil(t, got)
		})
	}
}

func TestPasskeySessionRedisErrorsSupportsRedis60(t *testing.T) {
	srv, client := newRedis60QueueSessionClient(t)
	ctx := context.Background()
	store := NewPasskeySessionStore(client)
	session := &service.PasskeySession{Kind: "authentication", UserID: 42}
	token, err := store.Store(ctx, session, time.Minute)
	require.NoError(t, err)

	srv.SetError("ERR injected Redis failure")
	got, err := store.Consume(ctx, token)
	require.ErrorContains(t, err, "consume passkey session: ERR injected Redis failure")
	require.NotErrorIs(t, err, service.ErrPasskeySession)
	require.Nil(t, got)
	srv.SetError("")
	got, err = store.Consume(ctx, token)
	require.NoError(t, err)
	require.Equal(t, session, got, "a failed Redis command must not consume the session")

	require.NoError(t, client.Close())
	got, err = store.Consume(ctx, "valid-token")
	require.ErrorIs(t, err, redis.ErrClosed)
	require.ErrorContains(t, err, "consume passkey session:")
	require.Nil(t, got)
	token, err = store.Store(ctx, session, time.Minute)
	require.ErrorIs(t, err, redis.ErrClosed)
	require.ErrorContains(t, err, "store passkey session:")
	require.Empty(t, token)
}
