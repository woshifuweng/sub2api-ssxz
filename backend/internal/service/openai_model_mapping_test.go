package service

import "testing"

func TestResolveOpenAIForwardModel(t *testing.T) {
	tests := []struct {
		name               string
		account            *Account
		requestedModel     string
		defaultMappedModel string
		expectedModel      string
	}{
		{
			name: "falls back to group default when account has no mapping",
			account: &Account{
				Credentials: map[string]any{},
			},
			requestedModel:     "gpt-5.4",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-4o-mini",
		},
		{
			name: "preserves exact passthrough mapping instead of group default",
			account: &Account{
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-5.4": "gpt-5.4",
					},
				},
			},
			requestedModel:     "gpt-5.4",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-5.4",
		},
		{
			name: "preserves wildcard passthrough mapping instead of group default",
			account: &Account{
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-*": "gpt-5.4",
					},
				},
			},
			requestedModel:     "gpt-5.4",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-5.4",
		},
		{
			name: "uses account remap when explicit target differs",
			account: &Account{
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-5": "gpt-5.4",
					},
				},
			},
			requestedModel:     "gpt-5",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-5.4",
		},
		{
			name: "reasoning suffix falls back to base model exact mapping",
			account: &Account{
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-5.4": "gpt-5.4-mini",
					},
				},
			},
			requestedModel:     "gpt-5.4-xhigh",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-5.4-mini",
		},
		{
			name: "reasoning suffix falls back to wildcard mapping",
			account: &Account{
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-5.4*": "gpt-5.4-mini",
					},
				},
			},
			requestedModel:     "gpt-5.4-xhigh",
			defaultMappedModel: "gpt-4o-mini",
			expectedModel:      "gpt-5.4-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveOpenAIForwardModel(tt.account, tt.requestedModel, tt.defaultMappedModel); got != tt.expectedModel {
				t.Fatalf("resolveOpenAIForwardModel(...) = %q, want %q", got, tt.expectedModel)
			}
		})
	}
}

func TestResolveOpenAIForwardModel_PreventsClaudeModelFromFallingBackToGpt51(t *testing.T) {
	account := &Account{
		Credentials: map[string]any{},
	}

	withoutDefault := resolveOpenAIForwardModel(account, "claude-opus-4-6", "")
	if got := normalizeCodexModel(withoutDefault); got != "gpt-5.1" {
		t.Fatalf("normalizeCodexModel(%q) = %q, want %q", withoutDefault, got, "gpt-5.1")
	}

	withDefault := resolveOpenAIForwardModel(account, "claude-opus-4-6", "gpt-5.4")
	if got := normalizeCodexModel(withDefault); got != "gpt-5.4" {
		t.Fatalf("normalizeCodexModel(%q) = %q, want %q", withDefault, got, "gpt-5.4")
	}
}

func TestResolveOpenAICompatibleChatCompletionsPassthroughModel(t *testing.T) {
	t.Run("preserves explicit DeepSeek request instead of group default", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{}}

		got := resolveOpenAICompatibleChatCompletionsPassthroughModel(account, "deepseek-v4-flash")
		if got != "deepseek-v4-flash" {
			t.Fatalf("resolveOpenAICompatibleChatCompletionsPassthroughModel(...) = %q, want deepseek-v4-flash", got)
		}
	})

	t.Run("applies account model mapping when configured", func(t *testing.T) {
		account := &Account{
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"deepseek-v4-flash": "deepseek-chat",
				},
			},
		}

		got := resolveOpenAICompatibleChatCompletionsPassthroughModel(account, "deepseek-v4-flash")
		if got != "deepseek-chat" {
			t.Fatalf("resolveOpenAICompatibleChatCompletionsPassthroughModel(...) = %q, want deepseek-chat", got)
		}
	})
}
