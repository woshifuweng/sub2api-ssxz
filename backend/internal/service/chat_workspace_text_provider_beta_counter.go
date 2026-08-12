package service

import (
	"strings"
	"sync"
	"time"
)

const (
	WorkspaceTextProviderBetaCounterReasonMissingCounter       = "missing_beta_request_counter"
	WorkspaceTextProviderBetaCounterReasonMissingUser          = "missing_beta_user"
	WorkspaceTextProviderBetaCounterReasonMissingProvider      = "missing_beta_provider_label"
	WorkspaceTextProviderBetaCounterReasonMissingModel         = "missing_beta_model"
	WorkspaceTextProviderBetaCounterReasonInvalidUserLimit     = "invalid_beta_daily_request_cap"
	WorkspaceTextProviderBetaCounterReasonInvalidProviderLimit = "invalid_beta_provider_request_cap"
	WorkspaceTextProviderBetaCounterReasonInvalidModelLimit    = "invalid_beta_model_request_cap"
	WorkspaceTextProviderBetaCounterReasonInvalidTestRunLimit  = "invalid_beta_test_run_request_cap"
	WorkspaceTextProviderBetaCounterReasonUserLimitExceeded    = "beta_user_daily_request_cap_exceeded"
	WorkspaceTextProviderBetaCounterReasonProviderExceeded     = "beta_provider_daily_request_cap_exceeded"
	WorkspaceTextProviderBetaCounterReasonModelExceeded        = "beta_model_daily_request_cap_exceeded"
	WorkspaceTextProviderBetaCounterReasonTestRunExceeded      = "beta_test_run_request_cap_exceeded"
)

type WorkspaceTextProviderBetaRequestCounter struct {
	mu     sync.Mutex
	caps   WorkspaceTextProviderBetaRequestCaps
	counts map[workspaceTextProviderBetaRequestCounterKey]int64
	now    func() time.Time
}

type WorkspaceTextProviderBetaRequestCounterDecision struct {
	Allowed       bool
	Reasons       []string
	Date          string
	UserID        int64
	ProviderLabel string
	Model         string
	UserUsed      int64
	UserLimit     int64
	ProviderUsed  int64
	ProviderLimit int64
	ModelUsed     int64
	ModelLimit    int64
	TestRunUsed   int64
	TestRunLimit  int64
}

type workspaceTextProviderBetaRequestCounterScope string

const (
	workspaceTextProviderBetaRequestCounterScopeUser     workspaceTextProviderBetaRequestCounterScope = "user"
	workspaceTextProviderBetaRequestCounterScopeProvider workspaceTextProviderBetaRequestCounterScope = "provider"
	workspaceTextProviderBetaRequestCounterScopeModel    workspaceTextProviderBetaRequestCounterScope = "model"
	workspaceTextProviderBetaRequestCounterScopeTestRun  workspaceTextProviderBetaRequestCounterScope = "test_run"
)

type workspaceTextProviderBetaRequestCounterKey struct {
	Scope         workspaceTextProviderBetaRequestCounterScope
	Date          string
	UserID        int64
	ProviderLabel string
	Model         string
}

func NewWorkspaceTextProviderBetaRequestCounter(decision WorkspaceTextProviderGateDecision) *WorkspaceTextProviderBetaRequestCounter {
	return &WorkspaceTextProviderBetaRequestCounter{
		caps:   decision.BetaRequestCaps,
		counts: make(map[workspaceTextProviderBetaRequestCounterKey]int64),
		now:    time.Now,
	}
}

func reserveWorkspaceTextProviderBetaRequestCounter(counter *WorkspaceTextProviderBetaRequestCounter, userID int64, providerLabel, model string) WorkspaceTextProviderBetaRequestCounterDecision {
	if counter == nil {
		return WorkspaceTextProviderBetaRequestCounterDecision{
			Allowed:       false,
			Reasons:       []string{WorkspaceTextProviderBetaCounterReasonMissingCounter},
			UserID:        userID,
			ProviderLabel: strings.TrimSpace(providerLabel),
			Model:         strings.TrimSpace(model),
		}
	}
	return counter.Reserve(userID, providerLabel, model)
}

func (counter *WorkspaceTextProviderBetaRequestCounter) Reserve(userID int64, providerLabel, model string) WorkspaceTextProviderBetaRequestCounterDecision {
	if counter == nil {
		return reserveWorkspaceTextProviderBetaRequestCounter(nil, userID, providerLabel, model)
	}

	providerLabel = strings.TrimSpace(providerLabel)
	model = strings.TrimSpace(model)
	now := time.Now
	if counter.now != nil {
		now = counter.now
	}
	date := now().UTC().Format("2006-01-02")
	decision := WorkspaceTextProviderBetaRequestCounterDecision{
		Date:          date,
		UserID:        userID,
		ProviderLabel: providerLabel,
		Model:         model,
		UserLimit:     int64(counter.caps.DailyRequestCap),
		ProviderLimit: int64(counter.caps.ProviderRequestCap),
		ModelLimit:    int64(counter.caps.ModelRequestCap),
		TestRunLimit:  int64(counter.caps.TestRunRequestCap),
	}

	if userID <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonMissingUser)
	}
	if providerLabel == "" {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonMissingProvider)
	}
	if model == "" {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonMissingModel)
	}
	if counter.caps.DailyRequestCap <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidUserLimit)
	}
	if counter.caps.ProviderRequestCap <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidProviderLimit)
	}
	if counter.caps.ModelRequestCap <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidModelLimit)
	}
	if counter.caps.TestRunRequestCap <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidTestRunLimit)
	}

	userKey, providerKey, modelKey, testRunKey := workspaceTextProviderBetaRequestCounterKeys(date, userID, providerLabel, model)

	counter.mu.Lock()
	defer counter.mu.Unlock()

	decision.UserUsed = counter.counts[userKey]
	decision.ProviderUsed = counter.counts[providerKey]
	decision.ModelUsed = counter.counts[modelKey]
	decision.TestRunUsed = counter.counts[testRunKey]

	if decision.UserLimit > 0 && decision.UserUsed >= decision.UserLimit {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonUserLimitExceeded)
	}
	if decision.ProviderLimit > 0 && decision.ProviderUsed >= decision.ProviderLimit {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonProviderExceeded)
	}
	if decision.ModelLimit > 0 && decision.ModelUsed >= decision.ModelLimit {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonModelExceeded)
	}
	if decision.TestRunLimit > 0 && decision.TestRunUsed >= decision.TestRunLimit {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaCounterReasonTestRunExceeded)
	}
	if len(decision.Reasons) > 0 {
		return decision
	}

	counter.counts[userKey]++
	counter.counts[providerKey]++
	counter.counts[modelKey]++
	counter.counts[testRunKey]++
	decision.UserUsed++
	decision.ProviderUsed++
	decision.ModelUsed++
	decision.TestRunUsed++
	decision.Allowed = true
	return decision
}

func (counter *WorkspaceTextProviderBetaRequestCounter) TestRunUsed() int64 {
	if counter == nil {
		return 0
	}
	counter.mu.Lock()
	defer counter.mu.Unlock()

	total := int64(0)
	for key, used := range counter.counts {
		if key.Scope == workspaceTextProviderBetaRequestCounterScopeTestRun {
			total += used
		}
	}
	return total
}

func workspaceTextProviderBetaRequestCounterKeys(date string, userID int64, providerLabel, model string) (
	workspaceTextProviderBetaRequestCounterKey,
	workspaceTextProviderBetaRequestCounterKey,
	workspaceTextProviderBetaRequestCounterKey,
	workspaceTextProviderBetaRequestCounterKey,
) {
	return workspaceTextProviderBetaRequestCounterKey{
			Scope:  workspaceTextProviderBetaRequestCounterScopeUser,
			Date:   date,
			UserID: userID,
		},
		workspaceTextProviderBetaRequestCounterKey{
			Scope:         workspaceTextProviderBetaRequestCounterScopeProvider,
			Date:          date,
			ProviderLabel: strings.ToLower(strings.TrimSpace(providerLabel)),
		},
		workspaceTextProviderBetaRequestCounterKey{
			Scope: workspaceTextProviderBetaRequestCounterScopeModel,
			Date:  date,
			Model: strings.ToLower(strings.TrimSpace(model)),
		},
		workspaceTextProviderBetaRequestCounterKey{
			Scope: workspaceTextProviderBetaRequestCounterScopeTestRun,
		}
}

func workspaceTextProviderBetaRequestCounterMetadata(decision WorkspaceTextProviderBetaRequestCounterDecision) map[string]any {
	return map[string]any{
		"beta_counter_allowed":       decision.Allowed,
		"beta_counter_block_reasons": decision.Reasons,
		"beta_counter_date":          decision.Date,
		"beta_counter_provider":      decision.ProviderLabel,
		"beta_counter_model":         decision.Model,
		"beta_user_used":             decision.UserUsed,
		"beta_user_limit":            decision.UserLimit,
		"beta_provider_used":         decision.ProviderUsed,
		"beta_provider_limit":        decision.ProviderLimit,
		"beta_model_used":            decision.ModelUsed,
		"beta_model_limit":           decision.ModelLimit,
		"beta_test_run_used":         decision.TestRunUsed,
		"beta_test_run_limit":        decision.TestRunLimit,
	}
}
