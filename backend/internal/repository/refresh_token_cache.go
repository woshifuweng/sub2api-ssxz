package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	refreshTokenKeyPrefix   = "refresh_token:"
	userRefreshTokensPrefix = "user_refresh_tokens:"
	tokenFamilyPrefix       = "token_family:"
)

// refreshTokenKey generates the Redis key for a refresh token.
func refreshTokenKey(tokenHash string) string {
	return refreshTokenKeyPrefix + tokenHash
}

// userRefreshTokensKey generates the Redis key for user's token set.
func userRefreshTokensKey(userID int64) string {
	return fmt.Sprintf("%s%d", userRefreshTokensPrefix, userID)
}

// tokenFamilyKey generates the Redis key for token family set.
func tokenFamilyKey(familyID string) string {
	return tokenFamilyPrefix + familyID
}

type refreshTokenCache struct {
	rdb *redis.Client
}

var storeRefreshTokenScript = redis.NewScript(`
redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[3])
redis.call("SADD", KEYS[2], ARGV[2])
redis.call("PEXPIRE", KEYS[2], ARGV[3])
redis.call("SADD", KEYS[3], ARGV[2])
redis.call("PEXPIRE", KEYS[3], ARGV[3])
return 1
`)

var rotateRefreshTokenScript = redis.NewScript(`
local oldValue = redis.call("GET", KEYS[1])
if not oldValue then
    return 0
end

redis.call("DEL", KEYS[1])
redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[3])
redis.call("SREM", KEYS[3], ARGV[4])
redis.call("SADD", KEYS[3], ARGV[2])
redis.call("PEXPIRE", KEYS[3], ARGV[3])
redis.call("SREM", KEYS[4], ARGV[4])
redis.call("SADD", KEYS[4], ARGV[2])
redis.call("PEXPIRE", KEYS[4], ARGV[3])
return 1
`)

// NewRefreshTokenCache creates a new RefreshTokenCache implementation.
func NewRefreshTokenCache(rdb *redis.Client) service.RefreshTokenCache {
	return &refreshTokenCache{rdb: rdb}
}

func (c *refreshTokenCache) StoreRefreshToken(ctx context.Context, tokenHash string, data *service.RefreshTokenData, ttl time.Duration) error {
	if data == nil {
		return errors.New("refresh token data is nil")
	}
	if ttl <= 0 {
		return errors.New("refresh token ttl must be positive")
	}
	val, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal refresh token data: %w", err)
	}
	return storeRefreshTokenScript.Run(
		ctx,
		c.rdb,
		[]string{
			refreshTokenKey(tokenHash),
			userRefreshTokensKey(data.UserID),
			tokenFamilyKey(data.FamilyID),
		},
		val,
		tokenHash,
		ttl.Milliseconds(),
	).Err()
}

func (c *refreshTokenCache) GetRefreshToken(ctx context.Context, tokenHash string) (*service.RefreshTokenData, error) {
	key := refreshTokenKey(tokenHash)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, service.ErrRefreshTokenNotFound
		}
		return nil, err
	}
	var data service.RefreshTokenData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal refresh token data: %w", err)
	}
	return &data, nil
}

func (c *refreshTokenCache) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash string,
	newTokenHash string,
	newData *service.RefreshTokenData,
	ttl time.Duration,
) (bool, error) {
	if newData == nil {
		return false, errors.New("refresh token data is nil")
	}
	if ttl <= 0 {
		return false, errors.New("refresh token ttl must be positive")
	}
	payload, err := json.Marshal(newData)
	if err != nil {
		return false, fmt.Errorf("marshal refresh token data: %w", err)
	}
	result, err := rotateRefreshTokenScript.Run(
		ctx,
		c.rdb,
		[]string{
			refreshTokenKey(oldTokenHash),
			refreshTokenKey(newTokenHash),
			userRefreshTokensKey(newData.UserID),
			tokenFamilyKey(newData.FamilyID),
		},
		payload,
		newTokenHash,
		ttl.Milliseconds(),
		oldTokenHash,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *refreshTokenCache) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	key := refreshTokenKey(tokenHash)
	return c.rdb.Del(ctx, key).Err()
}

func (c *refreshTokenCache) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	// Get all token hashes for this user
	tokenHashes, err := c.GetUserTokenHashes(ctx, userID)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get user token hashes: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	// Build keys to delete
	keys := make([]string, 0, len(tokenHashes)+1)
	for _, hash := range tokenHashes {
		keys = append(keys, refreshTokenKey(hash))
	}
	keys = append(keys, userRefreshTokensKey(userID))

	// Delete all keys in a pipeline
	pipe := c.rdb.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *refreshTokenCache) DeleteTokenFamily(ctx context.Context, familyID string) error {
	// Get all token hashes in this family
	tokenHashes, err := c.GetFamilyTokenHashes(ctx, familyID)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get family token hashes: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	// Build keys to delete
	keys := make([]string, 0, len(tokenHashes)+1)
	for _, hash := range tokenHashes {
		keys = append(keys, refreshTokenKey(hash))
	}
	keys = append(keys, tokenFamilyKey(familyID))

	// Delete all keys in a pipeline
	pipe := c.rdb.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *refreshTokenCache) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	key := userRefreshTokensKey(userID)
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, key, tokenHash)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *refreshTokenCache) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
	key := tokenFamilyKey(familyID)
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, key, tokenHash)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *refreshTokenCache) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
	key := userRefreshTokensKey(userID)
	return c.rdb.SMembers(ctx, key).Result()
}

func (c *refreshTokenCache) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
	key := tokenFamilyKey(familyID)
	return c.rdb.SMembers(ctx, key).Result()
}

func (c *refreshTokenCache) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	key := tokenFamilyKey(familyID)
	return c.rdb.SIsMember(ctx, key, tokenHash).Result()
}
