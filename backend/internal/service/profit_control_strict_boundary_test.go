package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

// Resolve real gates: a hand-built gate with an omitted D could reject every
// account and make the equality regression pass for the wrong reason.
func TestProfitControlStrictEffectiveDownstream(t *testing.T) {
	group := profitControlTestGroup(901, 0, 0)
	account := profitControlTestAccountWithRate(&Account{ID: 902}, 1)

	t.Run("openai", func(t *testing.T) {
		svc := &OpenAIGatewayService{}
		ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestCtx(group), &group.ID)
		vetoed, reason := OpenAIProfitControlVeto(ctx, account)
		require.True(t, vetoed, "U == effective D must be rejected even with zero margin and buffer")
		require.Equal(t, openAIProfitFilterReasonThreshold, reason)
	})
	t.Run("shared gateway", func(t *testing.T) {
		svc := &GatewayService{}
		ctx, _ := WithGatewayTokenRequestPricing(profitControlTestCtx(group))
		ctx = svc.withGatewayProfitControlGate(ctx, &group.ID)
		require.False(t, svc.isGatewayAccountProfitEligible(ctx, account))
	})
	t.Run("preview", func(t *testing.T) {
		report := PreviewProfitAdmission([]ProfitPreviewGroupInput{{
			Group: group, Accounts: []*Account{account},
		}}, time.Now())[0]
		require.Equal(t, ProfitPreviewClassRejectedThreshold, report.Verdicts[0].Class)
	})
}

type strictProfitUserRateRepo struct {
	UserGroupRateRepository
	t       *testing.T
	groupID int64
	rate    *float64
}

func (r strictProfitUserRateRepo) GetByUserAndGroup(_ context.Context, userID, groupID int64) (*float64, error) {
	require.Equal(r.t, int64(42), userID)
	require.Equal(r.t, r.groupID, groupID, "personal rate must belong to the billing group")
	return r.rate, nil
}

func TestProfitControlStrictBoundaryPricingPaths(t *testing.T) {
	pricingAt := time.Date(2026, 1, 15, 8, 30, 0, 0, timezone.Location())
	for _, tc := range []struct {
		name                        string
		base, personal, peak, wantD float64
		override, composite         bool
		margin, buffer              float64
	}{
		{name: "default", base: 0.5, peak: 1, wantD: 0.5},
		{name: "personal override", base: 2, personal: 0.5, override: true, peak: 1, wantD: 0.5},
		{name: "higher personal override", base: 0.5, personal: 2, override: true, peak: 1, wantD: 2},
		{name: "peak", base: 0.5, peak: 3, wantD: 1.5},
		{name: "personal peak", base: 2, personal: 0.5, override: true, peak: 3, wantD: 1.5},
		{name: "composite", base: 0.5, peak: 1, composite: true, wantD: 0.5},
		{name: "composite personal peak", base: 2, personal: 0.5, override: true, peak: 3, composite: true, wantD: 1.5},
		{name: "margin and buffer", base: 1, peak: 1, wantD: 1, margin: 0.2, buffer: 0.1},
		{name: "composite margin and buffer", base: 2, personal: 0.5, override: true, peak: 3, composite: true, wantD: 1.5, margin: 0.2, buffer: 0.1},
		{name: "tiny margin cannot admit break even via epsilon", base: 1, peak: 1, wantD: 1, margin: 1e-12},
		{name: "zero default", base: 0, peak: 1, wantD: 0},
		{name: "zero personal override", base: 2, override: true, peak: 1, wantD: 0},
		{name: "zero peak", base: 2, peak: 0, wantD: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, route := range []struct{ name, platform string }{
				{"openai", PlatformOpenAI}, {"openai grok", PlatformGrok},
				{"gateway openai", PlatformOpenAI}, {"gateway anthropic", PlatformAnthropic},
				{"gateway gemini", PlatformGemini}, {"gateway grok", PlatformGrok},
				{"gateway antigravity", PlatformAntigravity},
			} {
				t.Run(route.name, func(t *testing.T) {
					billing := profitControlTestGroup(911, tc.margin, tc.buffer)
					billing.Platform = route.platform
					billing.RateMultiplier = tc.base
					billing.SubscriptionType = SubscriptionTypeSubscription
					billing.PeakRateEnabled = true
					billing.PeakStart, billing.PeakEnd = "08:00", "09:00"
					billing.PeakRateMultiplier = tc.peak
					scheduled := billing
					if tc.composite {
						billing.Platform = PlatformComposite
						billing.ProfitMinMargin, billing.ProfitSafetyBuffer = 0.8, 0.1
						scheduled = profitControlTestGroup(912, tc.margin, tc.buffer)
						scheduled.Platform = route.platform
						scheduled.RateMultiplier = 99
					}
					var personal *float64
					if tc.override {
						personal = &tc.personal
					}
					resolver := newUserGroupRateResolver(strictProfitUserRateRepo{
						t: t, groupID: billing.ID, rate: personal,
					}, nil, time.Minute, nil, "test.profit")
					snapshot := &SchedulerSnapshotService{groupRepo: profitControlGroupRepo{group: scheduled}}
					ctx := context.WithValue(profitControlTestCtx(billing), ctxkey.UserID, int64(42))
					if route.name == "openai" || route.name == "openai grok" {
						ctx = context.WithValue(ctx, openAIPricingAtCtxKey{}, pricingAt)
						svc := &OpenAIGatewayService{userGroupRateResolver: resolver, schedulerSnapshot: snapshot}
						ctx = svc.withOpenAIProfitControlGate(ctx, &scheduled.ID)
					} else {
						ctx, _ = WithGatewayTokenRequestPricing(ctx)
						ctx = context.WithValue(ctx, gatewayTokenRequestPricingAtCtxKey{}, pricingAt)
						// Shared routing may replace ctx.Group, but must retain the frozen billing parent.
						ctx = context.WithValue(ctx, ctxkey.Group, scheduled)
						svc := &GatewayService{userGroupRateResolver: resolver, schedulerSnapshot: snapshot}
						ctx = svc.withGatewayProfitControlGate(ctx, &scheduled.ID)
					}
					gate, _ := ctx.Value(openAIProfitControlGateCtxKey{}).(*openAIProfitControlGate)
					require.NotNil(t, gate)
					require.Equal(t, tc.wantD, gate.downstream, "D must come from production price resolution, not a gate zero value")
					require.Equal(t, pricingAt, gate.pricingAt)
					threshold := tc.wantD * (1 - tc.margin - tc.buffer)
					require.InDelta(t, threshold, gate.threshold, 1e-12)
					for _, boundary := range []struct {
						name string
						u    float64
						veto bool
					}{
						{"equal D", tc.wantD, true},
						{"above D within old epsilon", tc.wantD + 1e-12, true},
						{"free", 0, tc.wantD == 0},
						{"below both", threshold / 2, tc.wantD == 0},
						{"equal discounted threshold", threshold, threshold >= tc.wantD},
						{"one float below D", math.Nextafter(tc.wantD, 0), tc.wantD == 0 || tc.margin+tc.buffer > 1e-9},
					} {
						t.Run(boundary.name, func(t *testing.T) {
							account := profitControlTestAccountWithRate(&Account{ID: 913}, boundary.u)
							vetoed, reason := OpenAIProfitControlVeto(ctx, account)
							require.Equal(t, boundary.veto, vetoed)
							if boundary.veto {
								require.Equal(t, openAIProfitFilterReasonThreshold, reason)
							} else {
								require.Empty(t, reason)
							}
						})
					}
				})
			}
		})
	}
}

func TestProfitControlStrictSchedulerAndTerminalAdmission(t *testing.T) {
	for _, advanced := range []string{"true", "false"} {
		t.Run("advanced="+advanced, func(t *testing.T) {
			resetOpenAIAdvancedSchedulerSettingCacheForTest()
			defer resetOpenAIAdvancedSchedulerSettingCacheForTest()
			group := profitControlTestGroup(931, 0, 0)
			group.RateMultiplier = 0.5
			account := profitControlTestAccountWithRate(upstreamCostTestOAuthAccount(932), 0.5)
			account.Status, account.Schedulable, account.Concurrency = StatusActive, true, 2
			svc := &OpenAIGatewayService{
				accountRepo:        stubOpenAIAccountRepo{accounts: []Account{*account}},
				cfg:                &config.Config{},
				rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService(advanced),
				concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
			}
			ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestCtx(group), &group.ID)
			selection, _, err := svc.SelectAccountWithScheduler(ctx, &group.ID, "", "", "gpt-test", nil, OpenAIUpstreamTransportAny, false)
			require.ErrorIs(t, err, ErrNoAvailableAccounts)
			require.Nil(t, selection, "break-even account must never be scheduled")

			// An admitted account can become break-even while waiting for a slot.
			profitControlTestAccountWithRate(account, 0.4)
			svc.accountRepo = stubOpenAIAccountRepo{accounts: []Account{*account}}
			selection, _, err = svc.SelectAccountWithScheduler(ctx, &group.ID, "", "", "gpt-test", nil, OpenAIUpstreamTransportAny, false)
			require.NoError(t, err)
			require.NotNil(t, selection)
			if selection.ReleaseFunc != nil {
				defer selection.ReleaseFunc()
			}
			handlerCtx := ContextWithSelectionProfitGate(context.Background(), selection)
			_, vetoed, _ := svc.ProfitControlVetoLatest(handlerCtx, selection.Account)
			require.False(t, vetoed)
			profitControlTestAccountWithRate(selection.Account, 0.5)
			_, vetoed, reason := svc.ProfitControlVetoLatest(handlerCtx, selection.Account)
			require.True(t, vetoed)
			require.Equal(t, openAIProfitFilterReasonThreshold, reason)
		})
	}
}

func TestProfitControlStrictPreviewZeroDownstream(t *testing.T) {
	for _, source := range []string{"default", "personal", "peak"} {
		t.Run(source, func(t *testing.T) {
			group := profitControlTestGroup(941, 0, 0)
			pricingAt := time.Date(2026, 1, 15, 8, 30, 0, 0, timezone.Location())
			input := ProfitPreviewGroupInput{Group: group, Accounts: []*Account{
				profitControlTestAccountWithRate(&Account{ID: 942}, 0),
			}, Models: []string{"model"}}
			switch source {
			case "default":
				group.RateMultiplier = 0
			case "personal":
				input.UserOverrides = map[int64]float64{42: 0}
			case "peak":
				group.SubscriptionType = SubscriptionTypeSubscription
				group.PeakRateEnabled = true
				group.PeakStart, group.PeakEnd = "08:00", "09:00"
				group.PeakRateMultiplier = 0
			}
			report := PreviewProfitAdmission([]ProfitPreviewGroupInput{input}, pricingAt)[0]
			require.Zero(t, report.MinEffectiveD)
			require.Zero(t, report.RemainingByModelMinD["model"])
			if source == "personal" {
				require.Equal(t, ProfitPreviewClassAdmitted, report.Verdicts[0].Class)
				require.True(t, report.Verdicts[0].RejectedUnderMinD)
			} else {
				require.Equal(t, ProfitPreviewClassRejectedThreshold, report.Verdicts[0].Class)
			}
		})
	}
}

func TestProfitControlStrictPreviewMatchesProduction(t *testing.T) {
	pricingAt := time.Date(2026, 1, 15, 8, 30, 0, 0, timezone.Location())
	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini, PlatformGrok, PlatformAntigravity} {
		for _, margin := range []float64{0, 0.2, 1e-12} {
			group := profitControlTestGroup(921, margin, 0)
			group.Platform = platform
			group.RateMultiplier = 1
			group.SubscriptionType = SubscriptionTypeSubscription
			group.PeakRateEnabled = true
			group.PeakStart, group.PeakEnd = "08:00", "09:00"
			group.PeakRateMultiplier = 2
			personal := 0.25
			accounts := make([]*Account, 0)
			for i, rate := range []float64{0, 0.5, 0.5 * (1 - margin), 2 * (1 - margin), 2, 2 + 1e-12} {
				accounts = append(accounts, profitControlTestAccountWithRate(&Account{ID: int64(i + 1), Platform: platform}, rate))
			}
			report := PreviewProfitAdmission([]ProfitPreviewGroupInput{{
				Group: group, Accounts: accounts, UserOverrides: map[int64]float64{42: personal}, Models: []string{"model"},
			}}, pricingAt)[0]
			require.Equal(t, 2.0, report.DefaultD)
			require.Equal(t, 0.5, report.MinEffectiveD)
			for _, override := range []bool{false, true} {
				ctx := profitControlTestCtx(group)
				svc := &GatewayService{}
				if override {
					ctx = context.WithValue(ctx, ctxkey.UserID, int64(42))
					svc.userGroupRateResolver = newUserGroupRateResolver(strictProfitUserRateRepo{
						t: t, groupID: group.ID, rate: &personal,
					}, nil, time.Minute, nil, "test.profit")
				}
				ctx, _ = WithGatewayTokenRequestPricing(ctx)
				ctx = context.WithValue(ctx, gatewayTokenRequestPricingAtCtxKey{}, pricingAt)
				ctx = svc.withGatewayProfitControlGate(ctx, &group.ID)
				admitted := 0
				for i, account := range accounts {
					want := *account.RateMultiplier < report.DefaultD && !profitControlOverThreshold(*account.RateMultiplier, report.ThresholdDefault)
					previewAdmitted := report.Verdicts[i].Class == ProfitPreviewClassAdmitted
					if override {
						want = *account.RateMultiplier < report.MinEffectiveD && !profitControlOverThreshold(*account.RateMultiplier, report.ThresholdMinD)
						previewAdmitted = previewAdmitted && !report.Verdicts[i].RejectedUnderMinD
					}
					require.Equal(t, want, previewAdmitted, "platform=%s margin=%g override=%t U=%g", platform, margin, override, *account.RateMultiplier)
					require.Equal(t, want, svc.isGatewayAccountProfitEligible(ctx, account))
					if want && account.IsModelSupported("model") {
						admitted++
					}
				}
				counts := report.RemainingByModel
				if override {
					counts = report.RemainingByModelMinD
				}
				require.Equal(t, admitted, counts["model"])
			}
		}
	}
}
