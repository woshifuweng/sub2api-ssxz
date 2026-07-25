package dto

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TestPublicSettingsInjectionPayload_SchemaDoesNotDrift guarantees the SSR
// injection struct exposes every JSON field consumed by the frontend.
//
// Why this test exists: before we extracted a named PublicSettingsInjectionPayload
// type, the inline struct was manually kept in sync with dto.PublicSettings and
// drifted — ChannelMonitorEnabled / AvailableChannelsEnabled were missing, which
// made the frontend read `undefined` on refresh and hide the "可用渠道" menu
// until the async /api/v1/settings/public round-trip finished.
//
// This test compares the two JSON-tag sets and fails if injection is missing
// any field that dto.PublicSettings exposes. Adding a new feature flag with
// only a DTO entry will fail this test until the injection struct is updated.
//
// Intentional exclusions (fields present on dto.PublicSettings that SSR does
// not need to inject) are listed in `dtoOnlyFields` below with a reason.
func TestPublicSettingsInjectionPayload_SchemaDoesNotDrift(t *testing.T) {
	injection := jsonTags(reflect.TypeOf(service.PublicSettingsInjectionPayload{}))
	dtoKeys := jsonTags(reflect.TypeOf(PublicSettings{}))

	var missing []string
	for key := range dtoKeys {
		if _, ok := injection[key]; ok {
			continue
		}
		missing = append(missing, key)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("service.PublicSettingsInjectionPayload is missing JSON fields present on dto.PublicSettings: %s\n"+
			"add the field to PublicSettingsInjectionPayload and GetPublicSettingsForInjection.", strings.Join(missing, ", "))
	}

	var extra []string
	for key := range injection {
		if _, ok := dtoKeys[key]; !ok {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Fatalf("service.PublicSettingsInjectionPayload exposes JSON fields absent from dto.PublicSettings: %s", strings.Join(extra, ", "))
	}
}

func TestPublicSettingsInjectionPayload_JSONShapeAndSecretBoundary(t *testing.T) {
	payload := service.PublicSettingsInjectionPayload{
		ForceEmailOnThirdPartySignup: true,
		PurchaseLinkCNY10:            "https://example.com/10",
		PurchaseLinkCNY30:            "https://example.com/30",
		PurchaseLinkCNY100:           "https://example.com/100",
		SoraClientEnabled:            true,
		WebSearch: service.PublicWorkspaceWebSearchSettings{
			Available: true,
			Provider:  "jina",
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal public settings injection payload: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal public settings injection payload: %v", err)
	}
	for _, key := range []string{
		"force_email_on_third_party_signup",
		"purchase_link_cny_10",
		"purchase_link_cny_30",
		"purchase_link_cny_100",
		"sora_client_enabled",
		"web_search",
	} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("public settings injection JSON is missing %q", key)
		}
	}
	webSearch, ok := decoded["web_search"].(map[string]any)
	if !ok {
		t.Fatalf("web_search has unexpected JSON shape: %#v", decoded["web_search"])
	}
	if _, ok := webSearch["available"]; !ok {
		t.Fatal("web_search.available is missing")
	}
	if _, ok := webSearch["provider"]; !ok {
		t.Fatal("web_search.provider is missing")
	}

	for _, forbidden := range []string{
		"turnstile_secret_key",
		"smtp_password",
		"oauth_client_secret",
		"secret_access_key",
		"jwt_secret",
		"upstream_api_key",
	} {
		if strings.Contains(string(raw), `"`+forbidden+`"`) {
			t.Fatalf("public settings injection leaked forbidden field %q", forbidden)
		}
	}
}

func jsonTags(t reflect.Type) map[string]struct{} {
	out := make(map[string]struct{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out
}
