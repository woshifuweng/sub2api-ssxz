package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestOpsErrorLogInsertDoesNotPersistRequestReplayFields(t *testing.T) {
	disallowedColumns := []string{
		"request_body",
		"request_headers",
		"request_body_truncated",
		"request_body_bytes",
		"is_retryable",
		"retry_count",
		"resolved_retry_id",
	}

	insertSQL := strings.ToLower(insertOpsErrorLogSQL)
	for _, column := range disallowedColumns {
		if strings.Contains(insertSQL, column) {
			t.Fatalf("ops error log insert still references dropped replay column %q", column)
		}
	}

	requestBody := `{"private":"replay-body-sentinel"}`
	requestHeaders := `{"authorization":"replay-header-sentinel"}`
	requestBodyBytes := 9173457
	input := &service.OpsInsertErrorLogInput{
		ErrorPhase:           "upstream",
		ErrorType:            "upstream_error",
		RequestBodyJSON:      &requestBody,
		RequestBodyTruncated: true,
		RequestBodyBytes:     &requestBodyBytes,
		RequestHeadersJSON:   &requestHeaders,
		IsRetryable:          true,
		RetryCount:           771239,
		CreatedAt:            time.Unix(1, 0).UTC(),
	}

	args := opsInsertErrorLogArgs(input)
	if len(args) != 38 {
		t.Fatalf("unexpected ops insert argument count: got %d want 38", len(args))
	}
	for _, arg := range args {
		switch value := arg.(type) {
		case string:
			if value == requestBody || value == requestHeaders {
				t.Fatalf("ops insert arguments still carry replay string %q", value)
			}
		case sql.NullString:
			if value.Valid && (value.String == requestBody || value.String == requestHeaders) {
				t.Fatalf("ops insert arguments still carry replay string %q", value.String)
			}
		case int:
			if value == requestBodyBytes || value == input.RetryCount {
				t.Fatalf("ops insert arguments still carry replay integer %d", value)
			}
		case sql.NullInt64:
			if value.Valid && (value.Int64 == int64(requestBodyBytes) || value.Int64 == int64(input.RetryCount)) {
				t.Fatalf("ops insert arguments still carry replay integer %d", value.Int64)
			}
		}
	}
}
