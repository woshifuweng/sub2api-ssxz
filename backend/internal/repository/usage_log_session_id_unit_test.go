//go:build unit

package repository

import (
	"database/sql"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newSessionIDUsageLog(sessionID *string) *service.UsageLog {
	return &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "req-session-id",
		Model:        "claude-3",
		InputTokens:  10,
		OutputTokens: 5,
		TotalCost:    1.0,
		ActualCost:   1.0,
		SessionID:    sessionID,
		CreatedAt:    time.Now().UTC(),
	}
}

// TestPrepareUsageLogInsert_SessionIDArgWiring pins the trailing usage-log
// columns to the arg slice / arg-type table.
func TestPrepareUsageLogInsert_SessionIDArgWiring(t *testing.T) {
	require.Len(t, usageLogInsertArgTypes, 58, "arg-type table must include billing_model")

	sessionID := "sess-persisted-123"
	log := newSessionIDUsageLog(&sessionID)
	log.BillingModel = "claude-sonnet-4-5"
	prepared := prepareUsageLogInsert(log)

	require.Len(t, prepared.args, len(usageLogInsertArgTypes),
		"prepared args must match the arg-type table length")

	sessionArg := prepared.args[len(prepared.args)-3]
	ns, ok := sessionArg.(sql.NullString)
	require.True(t, ok, "session_id arg should be a sql.NullString, got %T", sessionArg)
	require.True(t, ns.Valid)
	require.Equal(t, sessionID, ns.String)

	billingModelArg, ok := prepared.args[len(prepared.args)-1].(sql.NullString)
	require.True(t, ok, "billing_model arg should be a sql.NullString")
	require.True(t, billingModelArg.Valid)
	require.Equal(t, log.BillingModel, billingModelArg.String)

	require.Equal(t, "text", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-3],
		"session_id arg type must be text")
	require.Equal(t, "timestamptz", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-2],
		"created_at arg type must be timestamptz")
	require.Equal(t, "text", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-1],
		"billing_model arg type must be text")
}

// TestPrepareUsageLogInsert_SessionIDNullWhenAbsent proves an absent session id is
// persisted as SQL NULL rather than an empty string.
func TestPrepareUsageLogInsert_SessionIDNullWhenAbsent(t *testing.T) {
	prepared := prepareUsageLogInsert(newSessionIDUsageLog(nil))
	sessionArg := prepared.args[len(prepared.args)-3]
	ns, ok := sessionArg.(sql.NullString)
	require.True(t, ok, "session_id arg should be a sql.NullString, got %T", sessionArg)
	require.False(t, ns.Valid, "absent session id must be NULL, not empty string")

	empty := ""
	preparedEmpty := prepareUsageLogInsert(newSessionIDUsageLog(&empty))
	nsEmpty := preparedEmpty.args[len(preparedEmpty.args)-3].(sql.NullString)
	require.False(t, nsEmpty.Valid, "empty session id must also be NULL")
}

// TestUsageLogInsertQueries_IncludeSessionID guards that every generated INSERT path
// and the SELECT column list reference session_id.
func TestUsageLogInsertQueries_IncludeSessionID(t *testing.T) {
	require.Contains(t, usageLogSelectColumns, "session_id",
		"SELECT column list must include session_id")
	require.Contains(t, usageLogSelectColumns, "billing_model",
		"SELECT column list must include billing_model")

	sessionID := "sess-in-query"
	log := newSessionIDUsageLog(&sessionID)
	log.BillingModel = "claude-sonnet-4-5"
	prepared := prepareUsageLogInsert(log)
	key := usageLogBatchKey(log.RequestID, log.APIKeyID)

	batchQuery, batchArgs := buildUsageLogBatchInsertQuery([]string{key},
		map[string]usageLogInsertPrepared{key: prepared})
	require.Contains(t, batchQuery, "session_id")
	// Two column references (INSERT column list + SELECT ... FROM input) plus the CTE def.
	require.GreaterOrEqual(t, strings.Count(batchQuery, "session_id"), 3)
	require.Len(t, batchArgs, len(prepared.args)+1,
		"batch args include the synthetic input_index before usage-log values")

	bestEffortQuery, bestEffortArgs := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	require.Contains(t, bestEffortQuery, "session_id")
	require.Contains(t, bestEffortQuery, "billing_model")
	require.Len(t, bestEffortArgs, len(prepared.args))
}

func TestUsageLogInsert_ColumnCountConsistency(t *testing.T) {
	source, err := os.ReadFile("usage_log_repo_insert.go")
	require.NoError(t, err)

	insertPattern := regexp.MustCompile(`(?s)INSERT INTO usage_logs\s*\((.*?)\)\s*(?:VALUES|SELECT)`)
	matches := insertPattern.FindAllSubmatch(source, -1)
	require.Len(t, matches, 4, "all four usage_logs INSERT paths must be checked")

	for i, match := range matches {
		rawColumns := strings.Split(string(match[1]), ",")
		columns := make([]string, 0, len(rawColumns))
		for _, rawColumn := range rawColumns {
			if column := strings.TrimSpace(rawColumn); column != "" {
				columns = append(columns, column)
			}
		}
		require.Lenf(t, columns, len(usageLogInsertArgTypes),
			"INSERT path %d column count must match prepared args", i+1)
		require.Equalf(t, "billing_model", columns[len(columns)-1],
			"INSERT path %d must append billing_model at the end", i+1)
	}

	require.GreaterOrEqual(t, strings.Count(string(source), "$58"), 2,
		"both static VALUES inserts must bind the trailing billing_model argument")
}
