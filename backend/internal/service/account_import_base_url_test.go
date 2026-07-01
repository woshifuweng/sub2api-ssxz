//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountImportStoreBaseURLStub struct {
	created []*Account
}

func (s *accountImportStoreBaseURLStub) CreateImportPlaceholders(_ context.Context, accounts []*Account) error {
	s.created = accounts
	return nil
}

func (s *accountImportStoreBaseURLStub) LookupMinAccountIDsByDedupFingerprint(context.Context, []string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (s *accountImportStoreBaseURLStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}

func (s *accountImportStoreBaseURLStub) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}

func (s *accountImportStoreBaseURLStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}

func (s *accountImportStoreBaseURLStub) BindGroups(context.Context, int64, []int64) error {
	return nil
}

func TestAccountImportRejectsUnsafeProviderBaseURL(t *testing.T) {
	store := &accountImportStoreBaseURLStub{}
	svc := NewAccountImportService(
		store,
		nil,
		nil,
		&adminSoraGroupRepoStub{},
		nil,
		nil,
		nil,
	)

	result, err := svc.Import(context.Background(), AccountImportPayload{
		SkipDefaultGroupBind: true,
		Accounts: []AccountImportAccount{
			{
				Name:     "unsafe-openai",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://127.0.0.1",
				},
			},
		},
	}, nil)

	require.NoError(t, err)
	require.Equal(t, 1, result.AccountFailed)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0].Message, "base_url")
	require.Nil(t, store.created)
}
