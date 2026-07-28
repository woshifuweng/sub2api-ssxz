package middleware

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewErrorResponseUsesOpenAIEnvelopeAndMappedType(t *testing.T) {
	tests := []struct {
		code     string
		wantType string
	}{
		{code: "INSUFFICIENT_BALANCE", wantType: "billing_error"},
		{code: "API_KEY_QUOTA_EXHAUSTED", wantType: "billing_error"},
		{code: "USAGE_LIMIT_EXCEEDED", wantType: "billing_error"},
		{code: "INVALID_API_KEY", wantType: "authentication_error"},
		{code: "API_KEY_REQUIRED", wantType: "authentication_error"},
		{code: "API_KEY_DISABLED", wantType: "authentication_error"},
		{code: "USER_INACTIVE", wantType: "authentication_error"},
		{code: "USER_NOT_FOUND", wantType: "authentication_error"},
		{code: "API_KEY_EXPIRED", wantType: "permission_error"},
		{code: "SUBSCRIPTION_NOT_FOUND", wantType: "permission_error"},
		{code: "SUBSCRIPTION_INVALID", wantType: "permission_error"},
		{code: "ACCESS_DENIED", wantType: "permission_error"},
		{code: "SOMETHING_ELSE", wantType: "api_error"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			body, err := json.Marshal(NewErrorResponse(tt.code, "message"))
			require.NoError(t, err)
			require.JSONEq(t, `{"error":{"message":"message","type":"`+tt.wantType+`","code":"`+tt.code+`"}}`, string(body))
		})
	}
}
