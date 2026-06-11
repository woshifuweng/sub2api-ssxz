package service

import (
	"strings"
	"sync/atomic"
)

const (
	WorkspaceTextProviderStagingQAReasonMissingHarness      = "missing_staging_qa_harness"
	WorkspaceTextProviderStagingQAReasonGateDisabled        = "staging_gate_disabled"
	WorkspaceTextProviderStagingQAReasonModelNotAllowlisted = "model_not_in_low_cost_allowlist"
	WorkspaceTextProviderStagingQAReasonRequestCapExceeded  = "max_requests_per_test_run_exceeded"
)

type WorkspaceTextProviderStagingQA struct {
	decision WorkspaceTextProviderGateDecision
	used     atomic.Int64
}

type WorkspaceTextProviderStagingQADecision struct {
	Allowed           bool
	Reasons           []string
	Model             string
	TestProviderLabel string
	Used              int64
	Limit             int64
}

func NewWorkspaceTextProviderStagingQA(decision WorkspaceTextProviderGateDecision) *WorkspaceTextProviderStagingQA {
	return &WorkspaceTextProviderStagingQA{decision: decision}
}

func reserveWorkspaceTextProviderStagingQA(q *WorkspaceTextProviderStagingQA, model string) WorkspaceTextProviderStagingQADecision {
	if q == nil {
		return WorkspaceTextProviderStagingQADecision{
			Allowed: false,
			Reasons: []string{WorkspaceTextProviderStagingQAReasonMissingHarness},
			Model:   strings.TrimSpace(model),
		}
	}
	return q.Reserve(model)
}

func (q *WorkspaceTextProviderStagingQA) Reserve(model string) WorkspaceTextProviderStagingQADecision {
	if q == nil {
		return reserveWorkspaceTextProviderStagingQA(nil, model)
	}

	model = strings.TrimSpace(model)
	decision := WorkspaceTextProviderStagingQADecision{
		Model:             model,
		TestProviderLabel: q.decision.TestProviderLabel,
		Used:              q.used.Load(),
		Limit:             int64(q.decision.MaxRequestsPerTestRun),
	}
	if !q.decision.Enabled {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderStagingQAReasonGateDisabled)
		decision.Reasons = append(decision.Reasons, q.decision.Reasons...)
		return decision
	}
	if !workspaceTextProviderModelAllowlisted(model, q.decision.LowCostModelAllowlist) {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderStagingQAReasonModelNotAllowlisted)
		return decision
	}
	if q.decision.MaxRequestsPerTestRun <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonInvalidRequestLimit)
		return decision
	}

	limit := int64(q.decision.MaxRequestsPerTestRun)
	for {
		current := q.used.Load()
		if current >= limit {
			decision.Used = current
			decision.Reasons = append(decision.Reasons, WorkspaceTextProviderStagingQAReasonRequestCapExceeded)
			return decision
		}
		if q.used.CompareAndSwap(current, current+1) {
			decision.Allowed = true
			decision.Used = current + 1
			return decision
		}
	}
}

func (q *WorkspaceTextProviderStagingQA) Used() int64 {
	if q == nil {
		return 0
	}
	return q.used.Load()
}

func workspaceTextProviderModelAllowlisted(model string, allowlist []string) bool {
	model = strings.TrimSpace(model)
	if model == "" || len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		if strings.EqualFold(model, strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func workspaceTextProviderStagingQAMetadata(decision WorkspaceTextProviderStagingQADecision) map[string]any {
	return map[string]any{
		"staging_qa_allowed":       decision.Allowed,
		"staging_qa_block_reasons": decision.Reasons,
		"staging_qa_model":         decision.Model,
		"staging_qa_provider":      decision.TestProviderLabel,
		"staging_qa_used":          decision.Used,
		"staging_qa_limit":         decision.Limit,
	}
}

func mergeWorkspaceMetadata(target map[string]any, extra map[string]any) {
	if target == nil || len(extra) == 0 {
		return
	}
	for key, value := range extra {
		target[key] = value
	}
}
