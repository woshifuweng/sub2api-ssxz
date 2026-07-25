package service

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type schedulerTestOpenAIAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r schedulerTestOpenAIAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			return &r.accounts[i], nil
		}
	}
	return nil, errors.New("account not found")
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]Account, error) {
	return r.listByPlatform(platform), nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]Account, error) {
	return r.listByPlatform(platform), nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]Account, error) {
	return r.listByPlatform(platform), nil
}

func (r schedulerTestOpenAIAccountRepo) listByPlatform(platform string) []Account {
	result := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			result = append(result, account)
		}
	}
	return result
}

type schedulerTestConcurrencyCache struct {
	ConcurrencyCache
	loadBatchErr    error
	loadMap         map[int64]*AccountLoadInfo
	acquireResults  map[int64]bool
	waitCounts      map[int64]int
	skipDefaultLoad bool
	acquiredIDs     *[]int64
	releasedIDs     *[]int64
}

func (c schedulerTestConcurrencyCache) AcquireAccountSlot(_ context.Context, accountID int64, _ int, _ string) (bool, error) {
	if c.acquiredIDs != nil {
		*c.acquiredIDs = append(*c.acquiredIDs, accountID)
	}
	if result, ok := c.acquireResults[accountID]; ok {
		return result, nil
	}
	return true, nil
}

func (c schedulerTestConcurrencyCache) ReleaseAccountSlot(_ context.Context, accountID int64, _ string) error {
	if c.releasedIDs != nil {
		*c.releasedIDs = append(*c.releasedIDs, accountID)
	}
	return nil
}

func (c schedulerTestConcurrencyCache) GetAccountsLoadBatch(_ context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if c.loadBatchErr != nil {
		return nil, c.loadBatchErr
	}
	out := make(map[int64]*AccountLoadInfo, len(accounts))
	for _, account := range accounts {
		if load, ok := c.loadMap[account.ID]; ok {
			out[account.ID] = load
			continue
		}
		if !c.skipDefaultLoad {
			out[account.ID] = &AccountLoadInfo{AccountID: account.ID}
		}
	}
	return out, nil
}

func (c schedulerTestConcurrencyCache) GetAccountWaitingCount(_ context.Context, accountID int64) (int, error) {
	return c.waitCounts[accountID], nil
}

type schedulerTestGatewayCache struct {
	sessionBindings map[string]int64
	deletedSessions map[string]int
}

func (c *schedulerTestGatewayCache) GetSessionAccountID(_ context.Context, _ int64, sessionHash string) (int64, error) {
	if id, ok := c.sessionBindings[sessionHash]; ok {
		return id, nil
	}
	return 0, errors.New("not found")
}

func (c *schedulerTestGatewayCache) SetSessionAccountID(_ context.Context, _ int64, sessionHash string, accountID int64, _ time.Duration) error {
	if c.sessionBindings == nil {
		c.sessionBindings = make(map[string]int64)
	}
	c.sessionBindings[sessionHash] = accountID
	return nil
}

func (c *schedulerTestGatewayCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (c *schedulerTestGatewayCache) DeleteSessionAccountID(_ context.Context, _ int64, sessionHash string) error {
	if c.deletedSessions == nil {
		c.deletedSessions = make(map[string]int)
	}
	c.deletedSessions[sessionHash]++
	delete(c.sessionBindings, sessionHash)
	return nil
}

type openAIAdvancedSchedulerSettingRepoStub struct {
	values map[string]string
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (*openAIAdvancedSchedulerSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected call to Set")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, err := s.GetValue(context.Background(), key); err == nil {
			result[key] = value
		}
	}
	return result, nil
}

func (*openAIAdvancedSchedulerSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected call to SetMultiple")
}

func (*openAIAdvancedSchedulerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected call to GetAll")
}

func (*openAIAdvancedSchedulerSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected call to Delete")
}

func newOpenAIAdvancedSchedulerRateLimitService(enabled string, values ...string) *RateLimitService {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	repo := &openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{}}
	if enabled != "" {
		repo.values[openAIAdvancedSchedulerSettingKey] = enabled
	}
	if len(values) > 0 && values[0] != "" {
		repo.values[SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled] = values[0]
	}
	if len(values) > 1 && values[1] != "" {
		repo.values[SettingKeyOpenAIAdvancedSchedulerSubscriptionPriorityEnabled] = values[1]
	}
	return &RateLimitService{settingService: NewSettingService(repo, &config.Config{})}
}
