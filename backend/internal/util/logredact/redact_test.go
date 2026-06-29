package logredact

import (
	"strings"
	"testing"
)

func TestRedactText_JSONLike(t *testing.T) {
	in := `{"access_token":"ya29.a0AfH6SMDUMMY","refresh_token":"1//0gDUMMY","other":"ok"}`
	out := RedactText(in)
	if out == in {
		t.Fatalf("expected redaction, got unchanged")
	}
	if want := `"access_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
	if want := `"refresh_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
}

func TestRedactText_QueryLike(t *testing.T) {
	in := "access_token=ya29.a0AfH6SMDUMMY refresh_token=1//0gDUMMY"
	out := RedactText(in)
	if strings.Contains(out, "ya29") || strings.Contains(out, "1//0") {
		t.Fatalf("expected tokens redacted, got %q", out)
	}
}

func TestRedactText_GOCSPX(t *testing.T) {
	in := "client_secret=GOCSPX-your-client-secret"
	out := RedactText(in)
	if strings.Contains(out, "your-client-secret") {
		t.Fatalf("expected secret redacted, got %q", out)
	}
	if !strings.Contains(out, "client_secret=***") {
		t.Fatalf("expected key redacted, got %q", out)
	}
}

func TestRedactMap_DefaultSensitiveAPIKeyFields(t *testing.T) {
	in := map[string]any{
		"Authorization": "Bearer sk-user-secret",
		"api_key":       "sk-api-secret",
		"x-api-key":     "sk-header-secret",
		"cookie":        "session=secret",
		"model":         "gpt-4.1",
		"nested": map[string]any{
			"apikey": "sk-nested-secret",
			"safe":   "kept",
		},
	}

	out := RedactMap(in)

	if out["Authorization"] != "***" {
		t.Fatalf("expected Authorization redacted, got %#v", out["Authorization"])
	}
	if out["api_key"] != "***" {
		t.Fatalf("expected api_key redacted, got %#v", out["api_key"])
	}
	if out["x-api-key"] != "***" {
		t.Fatalf("expected x-api-key redacted, got %#v", out["x-api-key"])
	}
	if out["cookie"] != "***" {
		t.Fatalf("expected cookie redacted, got %#v", out["cookie"])
	}
	if out["model"] != "gpt-4.1" {
		t.Fatalf("expected non-sensitive field preserved, got %#v", out["model"])
	}

	nested, ok := out["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map, got %#v", out["nested"])
	}
	if nested["apikey"] != "***" {
		t.Fatalf("expected nested apikey redacted, got %#v", nested["apikey"])
	}
	if nested["safe"] != "kept" {
		t.Fatalf("expected nested safe field preserved, got %#v", nested["safe"])
	}
}

func TestRedactText_DefaultSensitiveAPIKeyFields(t *testing.T) {
	in := "Authorization: Bearer sk-user-secret, x-api-key=sk-header-secret api_key=sk-api-secret cookie=session-secret"

	out := RedactText(in)

	for _, secret := range []string{
		"sk-user-secret",
		"sk-header-secret",
		"sk-api-secret",
		"session-secret",
	} {
		if strings.Contains(out, secret) {
			t.Fatalf("expected %q to be redacted from %q", secret, out)
		}
	}
	if !strings.Contains(out, "Authorization: ***") {
		t.Fatalf("expected Authorization header redacted, got %q", out)
	}
	if !strings.Contains(out, "x-api-key=***") {
		t.Fatalf("expected x-api-key redacted, got %q", out)
	}
	if !strings.Contains(out, "api_key=***") {
		t.Fatalf("expected api_key redacted, got %q", out)
	}
	if !strings.Contains(out, "cookie=***") {
		t.Fatalf("expected cookie redacted, got %q", out)
	}
}

func TestRedactText_ExtraKeyCacheUsesNormalizedSortedKey(t *testing.T) {
	clearExtraTextPatternCache()

	out1 := RedactText("custom_secret=abc", "Custom_Secret", " custom_secret ")
	out2 := RedactText("custom_secret=xyz", "custom_secret")
	if !strings.Contains(out1, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in first call, got %q", out1)
	}
	if !strings.Contains(out2, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in second call, got %q", out2)
	}

	if got := countExtraTextPatternCacheEntries(); got != 1 {
		t.Fatalf("expected 1 cached pattern set, got %d", got)
	}
}

func TestRedactText_DefaultPathDoesNotUseExtraCache(t *testing.T) {
	clearExtraTextPatternCache()

	out := RedactText("access_token=abc")
	if !strings.Contains(out, "access_token=***") {
		t.Fatalf("expected default key redacted, got %q", out)
	}
	if got := countExtraTextPatternCacheEntries(); got != 0 {
		t.Fatalf("expected extra cache to remain empty, got %d", got)
	}
}

func clearExtraTextPatternCache() {
	extraTextPatternCache.Range(func(key, value any) bool {
		extraTextPatternCache.Delete(key)
		return true
	})
}

func countExtraTextPatternCacheEntries() int {
	count := 0
	extraTextPatternCache.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
