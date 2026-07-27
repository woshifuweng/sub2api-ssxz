//go:build unit

package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunOpsAlertEvaluationLoopEvaluatesEveryTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time, 2)
	var evaluations atomic.Int32
	done := make(chan struct{})

	go func() {
		runOpsAlertEvaluationLoop(ctx, ticks, func() {
			evaluations.Add(1)
		})
		close(done)
	}()

	ticks <- time.Now()
	ticks <- time.Now()
	require.Eventually(t, func() bool {
		return evaluations.Load() >= 2
	}, time.Second, time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("evaluation loop did not stop after context cancellation")
	}
}

func TestRunOpsAlertEvaluationLoopRecoversAndContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticks := make(chan time.Time, 2)
	var evaluations atomic.Int32
	done := make(chan struct{})

	go func() {
		runOpsAlertEvaluationLoop(ctx, ticks, func() {
			if evaluations.Add(1) == 1 {
				panic("test panic")
			}
		})
		close(done)
	}()

	ticks <- time.Now()
	ticks <- time.Now()
	require.Eventually(t, func() bool {
		return evaluations.Load() >= 2
	}, time.Second, time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("evaluation loop did not stop after context cancellation")
	}
}
