package admin

import (
	"encoding/json"
	"testing"
)

func TestBatchSetRateRequestClearDistinguishesUnsetFromExplicitZero(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantClear bool
		wantRate  *float64
	}{
		{
			name:      "clear request",
			body:      `{"user_ids":[7],"clear":true}`,
			wantClear: true,
		},
		{
			name:     "explicit zero request",
			body:     `{"user_ids":[7],"aff_rebate_rate_percent":0,"clear":false}`,
			wantRate: func() *float64 { v := 0.0; return &v }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req BatchSetRateRequest
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatal(err)
			}
			if req.Clear != tt.wantClear {
				t.Fatalf("Clear = %v, want %v", req.Clear, tt.wantClear)
			}
			if (req.AffRebateRatePercent == nil) != (tt.wantRate == nil) {
				t.Fatalf("rate nil = %v, want nil = %v", req.AffRebateRatePercent == nil, tt.wantRate == nil)
			}
			if req.AffRebateRatePercent != nil && *req.AffRebateRatePercent != *tt.wantRate {
				t.Fatalf("rate = %v, want %v", *req.AffRebateRatePercent, *tt.wantRate)
			}
		})
	}
}
