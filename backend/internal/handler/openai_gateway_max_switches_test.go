//go:build unit

package handler

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// TestClampOpenAIMaxSwitches_FloorAtOne verifies the fix for P0-MaxSwitches:
// when the admin has configured switching (limit > 0), the huge-body path
// (cap == 0) must return 1, not 0, so at least one failover attempt is allowed.
func TestClampOpenAIMaxSwitches_FloorAtOne(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		cap   int
		want  int
	}{
		// P0 regression: huge body (cap=0) with limit>0 must return 1, not 0.
		{"huge body, limit=5", 5, 0, 1},
		{"huge body, limit=1", 1, 0, 1},
		// Admin-disabled switching (limit=0 or limit<0) must still return 0.
		{"admin disabled, limit=0", 0, 0, 0},
		{"admin disabled, limit=0 xlarge", 0, 1, 0},
		{"admin disabled, limit=-1", -1, 0, 0},
		// Normal clamp behaviour.
		{"xlarge body, limit=5 cap=1", 5, 1, 1},
		{"large body, limit=5 cap=2", 5, 2, 2},
		{"large body, limit=1 cap=2", 1, 2, 1},
		{"no body limit, limit=5", 5, 10, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampOpenAIMaxSwitches(tc.limit, tc.cap)
			require.Equal(t, tc.want, got,
				"clampOpenAIMaxSwitches(%d, %d)", tc.limit, tc.cap)
		})
	}
}

// TestAdjustOpenAIFailoverForLargeStreamingRequest_No5xxCooldown verifies that
// transient upstream 5xx responses (e.g. 503, 529 rate-limit) do NOT receive
// a long-context cooldown from adjustOpenAIFailoverForLargeStreamingRequest.
func TestAdjustOpenAIFailoverForLargeStreamingRequest_No5xxCooldown(t *testing.T) {
	// Use a body large enough to trigger the "large" threshold (default 128 KiB).
	largeBody := make([]byte, openAIStreamLargeBodyThresholdBytes+1)

	t.Run("5xx upstream — no cooldown added", func(t *testing.T) {
		for _, code := range []int{500, 503, 529} {
			fe := &service.UpstreamFailoverError{StatusCode: code}
			got := adjustOpenAIFailoverForLargeStreamingRequest(fe, largeBody, true, nil)
			require.NotNil(t, got)
			require.Zero(t, got.TempUnscheduleFor,
				"StatusCode %d: expected no cooldown for transient 5xx", code)
		}
	})

	t.Run("4xx upstream — cooldown is applied", func(t *testing.T) {
		fe := &service.UpstreamFailoverError{StatusCode: http.StatusRequestEntityTooLarge}
		got := adjustOpenAIFailoverForLargeStreamingRequest(fe, largeBody, true, nil)
		require.NotNil(t, got)
		require.Positive(t, got.TempUnscheduleFor,
			"413: expected cooldown for body-size error")
	})

	t.Run("network error (StatusCode=0) — cooldown is applied", func(t *testing.T) {
		fe := &service.UpstreamFailoverError{StatusCode: 0}
		got := adjustOpenAIFailoverForLargeStreamingRequest(fe, largeBody, true, nil)
		require.NotNil(t, got)
		require.Positive(t, got.TempUnscheduleFor,
			"StatusCode=0: expected cooldown for network-level failure on large body")
	})

	t.Run("non-streaming — unchanged", func(t *testing.T) {
		fe := &service.UpstreamFailoverError{StatusCode: 500}
		got := adjustOpenAIFailoverForLargeStreamingRequest(fe, largeBody, false, nil)
		require.Equal(t, fe, got, "non-streaming request must be returned unchanged")
	})
}
