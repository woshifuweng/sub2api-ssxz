package service

import (
	"context"
	"strings"
)

func (s *SchedulerSnapshotService) SetAdmissionTester(tester SchedulerAdmissionTester) {
	if s == nil {
		return
	}
	s.admissionTester = tester
}

// RefreshAccounts makes newly imported placeholder accounts visible without
// waiting for the outbox poller while preserving the upstream bucket fencing.
func (s *SchedulerSnapshotService) RefreshAccounts(ctx context.Context, accounts []*Account, reason string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	var firstErr error
	seen := make(map[batchSeenKey]struct{})
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		var previous *Account
		if cached, err := s.cache.GetAccount(ctx, account.ID); err == nil {
			previous = cached
		}
		if err := s.cache.SetAccount(ctx, account); err != nil && firstErr == nil {
			firstErr = err
		}
		s.maybeEnqueueAdmissionProbe(previous, account)
		if err := s.rebuildByAccount(ctx, account, account.GroupIDs, reason, seen); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *SchedulerSnapshotService) maybeEnqueueAdmissionProbe(previous, current *Account) {
	if s == nil || s.admissionTester == nil || current == nil || current.ID <= 0 {
		return
	}
	if current.Type != AccountTypeOAuth || strings.TrimSpace(current.GetCredential("refresh_token")) == "" {
		return
	}
	if !current.IsSchedulable() || (previous != nil && previous.IsSchedulable()) {
		return
	}
	s.admissionTester.EnqueueSchedulerAdmissionTest(current.ID)
}
