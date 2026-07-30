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
	setRateCalls  int
	accrueCalls   int

	summary   *AffiliateSummary
	summaries map[int64]*AffiliateSummary
	overview  *AffiliateUserOverview

	// lastSetRate is the last value handed to SetUserRebateRate. nil means the override was
	// cleared (NULL in the DB); a pointer to 0 means an explicit 0% override.
	lastSetRate *float64
}

func (r *affiliateRepoStub) EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error) {
	r.ensureCalls++
	if summary := r.summaries[userID]; summary != nil {
		return summary, nil
	}
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
	r.accrueCalls++
	return true, nil
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
	r.setRateCalls++
	r.lastSetRate = ratePercent
	if r.summary != nil {
		r.summary.AffRebateRatePercent = ratePercent
	}
	return nil
}

func (r *affiliateRepoStub) BatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *affiliateRepoStub) ListUsersWithCustomSettings(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (r *affiliateRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (r *affiliateRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (r *affiliateRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}

func (r *affiliateRepoStub) GetAffiliateUserOverview(_ context.Context, userID int64) (*AffiliateUserOverview, error) {
	if r.overview != nil {
		return r.overview, nil
	}
	return &AffiliateUserOverview{UserID: userID}, nil
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

// newAffiliateTestSettingServiceWithRate pins the global rebate rate so tests can assert
// on the difference between "fall back to global" and "explicit per-user override".
func newAffiliateTestSettingServiceWithRate(enabled bool, globalRatePercent string) *SettingService {
	enabledValue := "false"
	if enabled {
		enabledValue = "true"
	}
	return NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:    enabledValue,
		SettingKeyAffiliateRebateRate: globalRatePercent,
	}}, &config.Config{})
}

// TestAffiliateRebateRateZeroIsNotUnset locks the 0-vs-NULL semantics of the per-user
// exclusive rebate rate.
//
// NULL (AffRebateRatePercent == nil) means "no exclusive rate" and must fall back to the
// global rate. An explicit 0 is a valid business value meaning "this user's rebate is
// switched off" and must resolve to 0 — it must never be treated as unset, because
// AffiliateRebateRateMin is 0.0 and so clamping cannot distinguish the two.
func TestAffiliateRebateRateZeroIsNotUnset(t *testing.T) {
	svc := NewAffiliateService(&affiliateRepoStub{}, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil)
	ctx := context.Background()

	require.Equal(t, 5.0, svc.resolveRebateRatePercent(ctx, &AffiliateSummary{UserID: 7}),
		"nil exclusive rate must fall back to the global rate")

	require.Equal(t, 0.0, svc.resolveRebateRatePercent(ctx, &AffiliateSummary{
		UserID:               7,
		AffRebateRatePercent: float64Ptr(0),
	}), "explicit 0 must stay 0 (rebate disabled), not fall back to the global rate")

	require.Equal(t, 12.5, svc.resolveRebateRatePercent(ctx, &AffiliateSummary{
		UserID:               7,
		AffRebateRatePercent: float64Ptr(12.5),
	}))

	// A nil inviter (no affiliate row at all) also follows the global rate.
	require.Equal(t, 5.0, svc.resolveRebateRatePercent(ctx, nil))
}

func TestAffiliateAccrualSkipsDisabledOrRevokedReseller(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name      string
		status    string
		revokedAt *time.Time
	}{
		{name: "disabled", status: ResellerStatusDisabled},
		{name: "revoked", status: ResellerStatusActive, revokedAt: &now},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inviterID := int64(9)
			repo := &affiliateRepoStub{summaries: map[int64]*AffiliateSummary{
				7: {UserID: 7, InviterID: &inviterID, CreatedAt: now},
				9: {
					UserID:            9,
					HasResellerRole:   true,
					ResellerStatus:    tc.status,
					ResellerRevokedAt: tc.revokedAt,
				},
			}}
			svc := NewAffiliateService(repo, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil)

			rebate, err := svc.AccrueInviteRebate(context.Background(), 7, 100)

			require.NoError(t, err)
			require.Zero(t, rebate)
			require.Zero(t, repo.accrueCalls)
		})
	}
}

func TestAffiliateAccrualKeepsOrdinaryAffiliateAndActiveReseller(t *testing.T) {
	for _, tc := range []struct {
		name            string
		hasResellerRole bool
		status          string
	}{
		{name: "ordinary affiliate"},
		{name: "active reseller", hasResellerRole: true, status: ResellerStatusActive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inviterID := int64(9)
			repo := &affiliateRepoStub{summaries: map[int64]*AffiliateSummary{
				7: {UserID: 7, InviterID: &inviterID, CreatedAt: time.Now()},
				9: {UserID: 9, HasResellerRole: tc.hasResellerRole, ResellerStatus: tc.status},
			}}
			svc := NewAffiliateService(repo, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil)

			rebate, err := svc.AccrueInviteRebate(context.Background(), 7, 100)

			require.NoError(t, err)
			require.Equal(t, 5.0, rebate)
			require.Equal(t, 1, repo.accrueCalls)
		})
	}
}

// TestAffiliateDetailEffectiveRateDistinguishesZeroFromUnset covers the same semantics
// through the user-facing surface (`/affiliate` shows "what will I earn").
func TestAffiliateDetailEffectiveRateDistinguishesZeroFromUnset(t *testing.T) {
	ctx := context.Background()

	unset := &affiliateRepoStub{summary: &AffiliateSummary{UserID: 7, AffCode: "SSXZTEST"}}
	detail, err := NewAffiliateService(unset, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil).
		GetAffiliateDetail(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 5.0, detail.EffectiveRebateRatePercent)

	disabled := &affiliateRepoStub{summary: &AffiliateSummary{
		UserID:               7,
		AffCode:              "SSXZTEST",
		AffRebateRatePercent: float64Ptr(0),
	}}
	detail, err = NewAffiliateService(disabled, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil).
		GetAffiliateDetail(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 0.0, detail.EffectiveRebateRatePercent)
}

// TestAdminSetUserRebateRatePreservesClearVsZero asserts the admin write path keeps the
// two states apart: clearing stores NULL, an explicit 0 stores 0.
func TestAdminSetUserRebateRatePreservesClearVsZero(t *testing.T) {
	repo := &affiliateRepoStub{}
	svc := NewAffiliateService(repo, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil)
	ctx := context.Background()

	// Clear -> NULL, and the effective rate falls back to the global 5%.
	require.NoError(t, svc.AdminSetUserRebateRate(ctx, 7, nil))
	require.Equal(t, 1, repo.setRateCalls)
	require.Nil(t, repo.lastSetRate)
	require.Equal(t, 5.0, svc.resolveRebateRatePercent(ctx, &AffiliateSummary{
		UserID:               7,
		AffRebateRatePercent: repo.lastSetRate,
	}))

	// Explicit 0 -> stored as 0, and the effective rate is 0.
	require.NoError(t, svc.AdminSetUserRebateRate(ctx, 7, float64Ptr(0)))
	require.Equal(t, 2, repo.setRateCalls)
	require.NotNil(t, repo.lastSetRate)
	require.Equal(t, 0.0, *repo.lastSetRate)
	require.Equal(t, 0.0, svc.resolveRebateRatePercent(ctx, &AffiliateSummary{
		UserID:               7,
		AffRebateRatePercent: repo.lastSetRate,
	}))

	// 0 is in range and must not be rejected as invalid.
	require.NoError(t, validateExclusiveRate(float64Ptr(0)))
	require.Error(t, validateExclusiveRate(float64Ptr(-1)))
	require.Error(t, validateExclusiveRate(float64Ptr(100.01)))
}

// TestAdminGetUserOverviewExposesCustomFlag pins the flag the admin UI relies on to render
// "unset (follows global X%)" differently from "explicit 0% (disabled)".
func TestAdminGetUserOverviewExposesCustomFlag(t *testing.T) {
	ctx := context.Background()

	unset := &affiliateRepoStub{overview: &AffiliateUserOverview{UserID: 7}}
	overview, err := NewAffiliateService(unset, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil).
		AdminGetUserOverview(ctx, 7)
	require.NoError(t, err)
	require.False(t, overview.RebateRateCustom)
	require.Equal(t, 5.0, overview.RebateRatePercent, "unset resolves to the global rate")

	disabled := &affiliateRepoStub{overview: &AffiliateUserOverview{
		UserID:            7,
		RebateRatePercent: 0,
		RebateRateCustom:  true,
	}}
	overview, err = NewAffiliateService(disabled, newAffiliateTestSettingServiceWithRate(true, "5"), nil, nil).
		AdminGetUserOverview(ctx, 7)
	require.NoError(t, err)
	require.True(t, overview.RebateRateCustom, "explicit 0 must stay flagged as a custom override")
	require.Equal(t, 0.0, overview.RebateRatePercent)
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
