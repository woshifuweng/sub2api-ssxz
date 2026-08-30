//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type resellerAffiliateRepoStub struct {
	summaries   map[int64]*AffiliateSummary
	accrueCalls int
}

func (r *resellerAffiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	return r.summaries[userID], nil
}
func (r *resellerAffiliateRepoStub) GetAffiliateByCode(context.Context, string) (*AffiliateSummary, error) {
	panic("unexpected GetAffiliateByCode call")
}
func (r *resellerAffiliateRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	panic("unexpected BindInviter call")
}
func (r *resellerAffiliateRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	r.accrueCalls++
	return true, nil
}
func (r *resellerAffiliateRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}
func (r *resellerAffiliateRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	panic("unexpected ThawFrozenQuota call")
}
func (r *resellerAffiliateRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}
func (r *resellerAffiliateRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	panic("unexpected ListInvitees call")
}
func (r *resellerAffiliateRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}
func (r *resellerAffiliateRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}
func (r *resellerAffiliateRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}
func (r *resellerAffiliateRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}
func (r *resellerAffiliateRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}
func (r *resellerAffiliateRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}
func (r *resellerAffiliateRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}
func (r *resellerAffiliateRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}
func (r *resellerAffiliateRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}

func resellerAffiliateSettingService() *SettingService {
	return NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:    "true",
		SettingKeyAffiliateRebateRate: "5",
	}}, &config.Config{})
}

// TestResolveRebateRatePercent_PerUserOverride verifies that per-inviter
// AffRebateRatePercent overrides the global rate, that NULL falls back to the
// global rate, and that out-of-range exclusive rates are clamped silently.
//
// SettingService is left nil here so globalRebateRatePercent returns the
// documented default (AffiliateRebateRateDefault = 20%) — this exercises the
// fallback path without spinning up a settings stub.
func TestResolveRebateRatePercent_PerUserOverride(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}

	// nil exclusive rate → falls back to global default (20%)
	require.InDelta(t, AffiliateRebateRateDefault,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{}), 1e-9)

	// exclusive rate set → overrides global
	rate := 50.0
	require.InDelta(t, 50.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &rate}), 1e-9)

	// exclusive rate 0 → returns 0 (no rebate, intentional)
	zero := 0.0
	require.InDelta(t, 0.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &zero}), 1e-9)

	// exclusive rate above max → clamped to Max
	tooHigh := 250.0
	require.InDelta(t, AffiliateRebateRateMax,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooHigh}), 1e-9)

	// exclusive rate below min → clamped to Min
	tooLow := -5.0
	require.InDelta(t, AffiliateRebateRateMin,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooLow}), 1e-9)
}

// TestIsEnabled_NilSettingServiceReturnsDefault verifies that IsEnabled
// safely handles a nil settingService dependency by returning the default
// (off). This protects callers from nil-pointer crashes in misconfigured
// environments.
func TestIsEnabled_NilSettingServiceReturnsDefault(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}
	require.False(t, svc.IsEnabled(context.Background()))
	require.Equal(t, AffiliateEnabledDefault, svc.IsEnabled(context.Background()))
}

// TestValidateExclusiveRate_BoundaryAndInvalid covers the validator used by
// admin-facing rate setters: nil is always valid (clear), in-range values
// are accepted, NaN/Inf and out-of-range values produce a typed BadRequest.
func TestValidateExclusiveRate_BoundaryAndInvalid(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateExclusiveRate(nil))

	for _, v := range []float64{0, 0.01, 50, 99.99, 100} {
		v := v
		require.NoError(t, validateExclusiveRate(&v), "value %v should be valid", v)
	}

	for _, v := range []float64{-0.01, 100.01, -100, 200} {
		v := v
		require.Error(t, validateExclusiveRate(&v), "value %v should be rejected", v)
	}

	nan := math.NaN()
	require.Error(t, validateExclusiveRate(&nan))
	posInf := math.Inf(1)
	require.Error(t, validateExclusiveRate(&posInf))
	negInf := math.Inf(-1)
	require.Error(t, validateExclusiveRate(&negInf))
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a***@g***.com", maskEmail("alice@gmail.com"))
	require.Equal(t, "x***@d***", maskEmail("x@domain"))
	require.Equal(t, "", maskEmail(""))
}

func TestIsValidAffiliateCodeFormat(t *testing.T) {
	t.Parallel()

	// 邀请码格式校验同时服务于：
	// 1) 系统自动生成的 12 位随机码（A-Z 去 I/O，2-9 去 0/1）
	// 2) 管理员设置的自定义专属码（如 "VIP2026"、"NEW_USER-1"）
	// 因此校验放宽到 [A-Z0-9_-]{4,32}（要求调用方先 ToUpper）。
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"valid canonical 12-char", "ABCDEFGHJKLM", true},
		{"valid all digits 2-9", "234567892345", true},
		{"valid mixed", "A2B3C4D5E6F7", true},
		{"valid admin custom short", "VIP1", true},
		{"valid admin custom with hyphen", "NEW-USER", true},
		{"valid admin custom with underscore", "VIP_2026", true},
		{"valid 32-char max", "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345", true},
		// Previously-excluded chars (I/O/0/1) are now allowed since admins may use them.
		{"letter I now allowed", "IBCDEFGHJKLM", true},
		{"letter O now allowed", "OBCDEFGHJKLM", true},
		{"digit 0 now allowed", "0BCDEFGHJKLM", true},
		{"digit 1 now allowed", "1BCDEFGHJKLM", true},
		{"too short (3 chars)", "ABC", false},
		{"too long (33 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456", false},
		{"lowercase rejected (caller must ToUpper first)", "abcdefghjklm", false},
		{"empty", "", false},
		{"utf8 non-ascii", "ÄÄÄÄÄÄ", false}, // bytes out of charset
		{"ascii punctuation .", "ABCDEFGHJK.M", false},
		{"whitespace", "ABCDEFGHJK M", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isValidAffiliateCodeFormat(tc.in))
		})
	}
}

func TestAffiliateAccrualSkipsInactiveReseller(t *testing.T) {
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
			repo := &resellerAffiliateRepoStub{summaries: map[int64]*AffiliateSummary{
				7: {UserID: 7, InviterID: &inviterID, CreatedAt: now},
				9: {
					UserID:            9,
					HasResellerRole:   true,
					ResellerStatus:    tc.status,
					ResellerRevokedAt: tc.revokedAt,
				},
			}}
			svc := NewAffiliateService(repo, resellerAffiliateSettingService(), nil, nil)

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
			repo := &resellerAffiliateRepoStub{summaries: map[int64]*AffiliateSummary{
				7: {UserID: 7, InviterID: &inviterID, CreatedAt: time.Now()},
				9: {UserID: 9, HasResellerRole: tc.hasResellerRole, ResellerStatus: tc.status},
			}}
			svc := NewAffiliateService(repo, resellerAffiliateSettingService(), nil, nil)

			rebate, err := svc.AccrueInviteRebate(context.Background(), 7, 100)

			require.NoError(t, err)
			require.Equal(t, 5.0, rebate)
			require.Equal(t, 1, repo.accrueCalls)
		})
	}
}
