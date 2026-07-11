package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEnforceUnboundedTokenRequestLimit(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		targetPath     string
		recognized     []string
		wantChanged    bool
		wantPath       string
		wantTokenLimit int64
	}{
		{
			name:           "anthropic missing max tokens",
			body:           `{"model":"claude-sonnet-4","messages":[]}`,
			targetPath:     "max_tokens",
			recognized:     []string{"max_tokens"},
			wantChanged:    true,
			wantPath:       "max_tokens",
			wantTokenLimit: unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:           "openai chat preserves legacy max tokens",
			body:           `{"model":"gpt-5","max_tokens":321}`,
			targetPath:     "max_completion_tokens",
			recognized:     []string{"max_completion_tokens", "max_tokens"},
			wantChanged:    false,
			wantPath:       "max_tokens",
			wantTokenLimit: 321,
		},
		{
			name:           "openai responses missing max output tokens",
			body:           `{"model":"gpt-5","input":"hello"}`,
			targetPath:     "max_output_tokens",
			recognized:     []string{"max_output_tokens"},
			wantChanged:    true,
			wantPath:       "max_output_tokens",
			wantTokenLimit: unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:           "gemini creates nested generation config",
			body:           `{"contents":[]}`,
			targetPath:     "generationConfig.maxOutputTokens",
			recognized:     []string{"generationConfig.maxOutputTokens"},
			wantChanged:    true,
			wantPath:       "generationConfig.maxOutputTokens",
			wantTokenLimit: unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:           "malformed explicit limit is left for normal validation",
			body:           `{"model":"gpt-5","max_completion_tokens":"many"}`,
			targetPath:     "max_completion_tokens",
			recognized:     []string{"max_completion_tokens", "max_tokens"},
			wantChanged:    false,
			wantPath:       "max_completion_tokens",
			wantTokenLimit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := EnforceUnboundedTokenRequestLimit([]byte(tt.body), tt.targetPath, tt.recognized...)
			require.NoError(t, err)
			require.Equal(t, tt.wantChanged, changed)
			result := gjson.GetBytes(got, tt.wantPath)
			if tt.wantTokenLimit == 0 {
				require.NotEqual(t, gjson.Number, result.Type)
				return
			}
			require.Equal(t, tt.wantTokenLimit, result.Int())
		})
	}
}
