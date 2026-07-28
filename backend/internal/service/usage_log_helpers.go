package service

import "strings"

func optionalTrimmedStringPtr(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// optionalNonEqualStringPtr returns a pointer only when value is non-empty and
// differs from compare.
func optionalNonEqualStringPtr(value, compare string) *string {
	if value == "" || value == compare {
		return nil
	}
	return &value
}

func forwardResultBillingModel(requestedModel, upstreamModel string) string {
	if trimmed := strings.TrimSpace(upstreamModel); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(requestedModel)
}

func optionalInt64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}
