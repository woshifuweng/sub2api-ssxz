//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type affiliateRepoStub struct {
	ensureCalls   int
	thawCalls     int
	transferCalls int

	summary *AffiliateSummary
}

func (r *affiliateRepoStub) EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error) {
	r.ensureCalls++
	if r.summary != nil {
		return r.summary, nil
	}
	return &AffiliateSummary{UserID: userID, AffCode: "SSXZTEST", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (r *affiliateRepoStub) GetAffiliateByCode(ctx context.Context, code string) (*AffiliateSummary, error) {
	panic("unexpected GetAffiliateByCode call")
}

func (r *affiliateRepoStub) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	panic("unexpected BindInviter call")
}

func (r *affiliateRepoStub) AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int, _ *int64) (bool, error) {
	panic("unexpected AccrueQuota call")
}

func (r *affiliateRepoStub) GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeUserID int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (r *affiliateRepoStub) ThawFrozenQuota(ctx context.Context, userID int64) (float64, error) {
	r.thawCalls++
	return 0, nil
}

func (r *affiliateRepoStub) TransferQuotaToBalance(ctx context.Context, userID int64) (float64, float64, error) {
	r.transferCalls++
	return 1.25, 9.5, nil
}

func (r *affiliateRepoStub) ListInvitees(ctx context.Context, inviterID int64, limit int) ([]AffiliateInvitee, error) {
	return nil, nil
}

func (r *affiliateRepoStub) UpdateUserAffCode(ctx context.Context, userID int64, newCode string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (r *affiliateRepoStub) ResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}

func (r *affiliateRepoStub) SetUserRebateRate(ctx context.Context, userID int64, ratePercent *float64) error {
	panic("unexpected SetUserRebateRate call")
}

func (r *affiliateRepoStub) BatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *affiliateRepoStub) ListUsersWithCustomSettings(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func newAffiliateTestSettingService(enabled bool) *SettingService {
	value := "false"
	if enabled {
		value = "true"
	}
	return NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled: value,
	}}, &config.Config{})
}

func TestAffiliateServiceUserMethodsRespectDisabledFlag(t *testing.T) {
	repo := &affiliateRepoStub{}
	svc := NewAffiliateService(repo, newAffiliateTestSettingService(false), nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 7)
	require.Nil(t, detail)
	require.ErrorIs(t, err, ErrAffiliateDisabled)
	require.Equal(t, 0, repo.thawCalls)
	require.Equal(t, 0, repo.ensureCalls)

	transferred, balance, err := svc.TransferAffiliateQuota(context.Background(), 7)
	require.Zero(t, transferred)
	require.Zero(t, balance)
	require.ErrorIs(t, err, ErrAffiliateDisabled)
	require.Equal(t, 0, repo.transferCalls)
}

func TestAffiliateServiceUserMethodsRemainAvailableWhenEnabled(t *testing.T) {
	repo := &affiliateRepoStub{}
	svc := NewAffiliateService(repo, newAffiliateTestSettingService(true), nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), detail.UserID)
	require.Equal(t, 1, repo.thawCalls)
	require.Equal(t, 1, repo.ensureCalls)

	transferred, balance, err := svc.TransferAffiliateQuota(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 1.25, transferred)
	require.Equal(t, 9.5, balance)
	require.Equal(t, 1, repo.transferCalls)
}
