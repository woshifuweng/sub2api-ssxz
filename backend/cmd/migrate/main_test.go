package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	checksum001 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	checksum133 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	checksum134 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	checksum135 = "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	checksum136 = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	expectedMigrationName = migration135Name
	expectedMigrationFile = migration135Name + ".sql"
	previousMigrationFile = "134_add_api_key_allowed_models.sql"
)

func TestValidateRunRequestRequiresExplicitConfirmations(t *testing.T) {
	base := validRunOptions(t)
	base.confirm = false
	requireValidateError(t, base, validEnv(), "--confirm")

	base = validRunOptions(t)
	base.yesIUnderstand = false
	requireValidateError(t, base, validEnv(), "--yes-i-understand")

	base = validRunOptions(t)
	requireValidateError(t, base, map[string]string{envMigrationConfirm: "true"}, envRunControlledMigration)

	base = validRunOptions(t)
	requireValidateError(t, base, map[string]string{envRunControlledMigration: expectedMigrationName}, envMigrationConfirm)
}

func TestValidateRunRequestRejectsWrongMigrationAndTarget(t *testing.T) {
	opts := validRunOptions(t)
	opts.migration = "134_add_api_key_allowed_models"
	requireValidateError(t, opts, validEnv(), "unknown controlled migration")

	opts = validRunOptions(t)
	opts.target = "prod"
	requireValidateError(t, opts, validEnv(), "invalid target")

	opts = validRunOptions(t)
	opts.target = "production"
	requireValidateError(t, opts, validEnv(), "target=production")

	opts = validRunOptions(t)
	opts.target = "../135_create_chat_workspace_tables.sql"
	err := validateRunRequest(opts, envLookup(validEnv()))
	requireErrorContains(t, err, "invalid target")
	requireContains(t, err.Error(), "allowed targets")
	requireContains(t, err.Error(), "local")
	requireContains(t, err.Error(), "staging")
}

func TestValidateRunRequestDefaultsBlankTargetToLocal(t *testing.T) {
	opts := validRunOptions(t)
	opts.target = ""
	if err := validateRunRequest(opts, envLookup(validEnv())); err != nil {
		t.Fatalf("validateRunRequest() error = %v", err)
	}
}

func TestValidateRunRequestRejectsProductionFor135And136(t *testing.T) {
	tests := []struct {
		name      string
		migration string
		checksum  string
	}{
		{name: "135", migration: migration135Name, checksum: checksum135},
		{name: "136", migration: migration136Name, checksum: checksum136},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := validRunOptions(t)
			opts.migration = tt.migration
			opts.target = "production"
			opts.expectedChecksum = tt.checksum
			env := validEnv()
			env[envRunControlledMigration] = tt.migration
			requireValidateError(t, opts, env, "target=production")
		})
	}
}

func TestValidateRunRequestRejectsBadEnvConfirmation(t *testing.T) {
	opts := validRunOptions(t)
	env := validEnv()
	env[envRunControlledMigration] = "135_create_chat_workspace_tables.sql"
	requireValidateError(t, opts, env, envRunControlledMigration+" mismatch")

	env = validEnv()
	env[envMigrationConfirm] = "false"
	requireValidateError(t, opts, env, envMigrationConfirm+" must be true")

	env = validEnv()
	env[envMigrationConfirm] = "fasle"
	requireValidateError(t, opts, env, "invalid "+envMigrationConfirm)
}

func TestValidateRunRequestRejectsInvalidExpectedChecksums(t *testing.T) {
	tests := []struct {
		name     string
		checksum string
		want     string
	}{
		{name: "empty", checksum: "", want: "--expected-checksum"},
		{name: "short", checksum: "1234abcd", want: "full 64-character sha256 hex"},
		{name: "prefix", checksum: checksum135[:16], want: "full 64-character sha256 hex"},
		{name: "substring", checksum: checksum135[8:40], want: "full 64-character sha256 hex"},
		{name: "random non hex", checksum: strings.Repeat("z", 64), want: "lowercase sha256 hex"},
		{name: "leading whitespace", checksum: " " + checksum135, want: "must not contain leading or trailing whitespace"},
		{name: "trailing whitespace", checksum: checksum135 + " ", want: "must not contain leading or trailing whitespace"},
		{name: "uppercase", checksum: strings.ToUpper(checksum135), want: "lowercase sha256 hex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := validRunOptions(t)
			opts.expectedChecksum = tt.checksum
			requireValidateError(t, opts, validEnv(), tt.want)
		})
	}
}

func TestValidateRunRequestDoesNotUseFeatureFlags(t *testing.T) {
	t.Setenv(config.AutoMigrationsEnabledEnv, "false")
	t.Setenv(config.ChatWorkspaceEnabledEnv, "true")

	if err := validateRunRequest(validRunOptions(t), envLookup(validEnv())); err != nil {
		t.Fatalf("validateRunRequest() error = %v", err)
	}
}

func TestStatusAndDryRunDoNotExecute(t *testing.T) {
	for _, args := range [][]string{
		{"--status"},
		{"--dry-run", "--target=staging"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr, depsThatFailIfCalled(t))
			if code != exitOK {
				t.Fatalf("runCLI(%v) exit = %d, stderr=%s", args, code, stderr.String())
			}
			out := stdout.String()
			requireContains(t, out, "will execute migration: no")
			requireContains(t, out, "database connection: not opened")
			requireContains(t, out, "schema_migrations status: not read")
		})
	}
}

func TestModesAreMutuallyExclusive(t *testing.T) {
	tests := [][]string{
		{"--run", "--status"},
		{"--run", "--dry-run"},
		{"--run", "--db-status"},
		{"--run", "--preflight"},
		{"--status", "--dry-run"},
		{"--status", "--db-status"},
		{"--dry-run", "--preflight"},
		{"--db-status", "--preflight"},
		{"--run", "--status", "--dry-run"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr, depsThatFailIfCalled(t))
			if code == exitOK {
				t.Fatalf("runCLI(%v) exit = %d, want non-zero", args, code)
			}
			requireContains(t, stderr.String(), "choose exactly one mode")
			requireContains(t, stderr.String(), "--status, --dry-run, --db-status, --preflight, or --run")
		})
	}
}

func TestDBStatusAndPreflightRejectProductionAndInvalidTargets(t *testing.T) {
	for _, mode := range []string{"--db-status", "--preflight"} {
		t.Run(mode+"_production", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{mode, "--migration=" + expectedMigrationName, "--target=production", "--expected-checksum=" + checksum135}, &stdout, &stderr, depsThatFailIfCalled(t))
			if code == exitOK {
				t.Fatalf("runCLI exit = %d, want non-zero", code)
			}
			requireContains(t, stderr.String(), "target=production is not allowed")
		})

		t.Run(mode+"_invalid_target", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{mode, "--migration=" + expectedMigrationName, "--target=prod", "--expected-checksum=" + checksum135}, &stdout, &stderr, depsThatFailIfCalled(t))
			if code == exitOK {
				t.Fatalf("runCLI exit = %d, want non-zero", code)
			}
			requireContains(t, stderr.String(), "invalid target")
		})
	}
}

func TestDBStatusAndPreflightRejectMissingOrBadChecksumBeforeDBOpen(t *testing.T) {
	for _, mode := range []string{"--db-status", "--preflight"} {
		t.Run(mode+"_missing_checksum", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{mode, "--migration=" + expectedMigrationName, "--target=staging"}, &stdout, &stderr, depsThatFailIfCalled(t))
			if code == exitOK {
				t.Fatalf("runCLI exit = %d, want non-zero", code)
			}
			requireContains(t, stderr.String(), "full 64-character sha256 hex")
		})

		t.Run(mode+"_bad_checksum", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{mode, "--migration=" + expectedMigrationName, "--target=staging", "--expected-checksum=abc"}, &stdout, &stderr, depsThatFailIfCalled(t))
			if code == exitOK {
				t.Fatalf("runCLI exit = %d, want non-zero", code)
			}
			requireContains(t, stderr.String(), "full 64-character sha256 hex")
		})
	}
}

func TestMissingModeDoesNotExecute(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI(nil, &stdout, &stderr, depsThatFailIfCalled(t))
	if code == exitOK {
		t.Fatalf("runCLI() exit = %d, want non-zero", code)
	}
	requireContains(t, stderr.String(), "choose exactly one mode")
}

func TestRunOnlyEntersExecutionWhenAllConfirmationsPass(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, ok := findMigration(embedded, expectedMigrationFile)
	if !ok {
		t.Fatal("test embedded migrations missing 135")
	}
	args := []string{
		"--run",
		"--migration=" + expectedMigrationName,
		"--target=staging",
		"--confirm",
		"--yes-i-understand",
		"--expected-checksum=" + expected.Checksum,
	}
	t.Setenv(envRunControlledMigration, expectedMigrationName)
	t.Setenv(envMigrationConfirm, "true")

	var stdout, stderr bytes.Buffer
	var applyCalled bool
	deps := depsWithSQLMock(t, appliedBefore135(t, embedded), func(context.Context, *sql.DB) error {
		applyCalled = true
		return nil
	})
	code := runCLIWithEmbeddedForTest(args, &stdout, &stderr, deps, embedded)
	if code != exitOK {
		t.Fatalf("runCLIWithEmbeddedForTest exit = %d, stderr=%s", code, stderr.String())
	}
	if !applyCalled {
		t.Fatal("applyMigrations was not called")
	}
	requireContains(t, stdout.String(), "controlled migration completed")
}

func TestRunRefusalDoesNotOpenDatabase(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--run"}, &stdout, &stderr, depsThatFailIfCalled(t))
	if code == exitOK {
		t.Fatalf("runCLI exit = %d, want non-zero", code)
	}
	requireContains(t, stderr.String(), "missing confirmations")
}

func TestRunRejectsBadChecksumBeforeDatabaseOpen(t *testing.T) {
	args := []string{
		"--run",
		"--migration=" + expectedMigrationName,
		"--target=staging",
		"--confirm",
		"--yes-i-understand",
		"--expected-checksum=" + checksum135[:20],
	}
	t.Setenv(envRunControlledMigration, expectedMigrationName)
	t.Setenv(envMigrationConfirm, "true")

	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr, depsThatFailIfCalled(t))
	if code == exitOK {
		t.Fatalf("runCLI exit = %d, want non-zero", code)
	}
	requireContains(t, stderr.String(), "full 64-character sha256 hex")
	requireContains(t, stderr.String(), "will execute migration: no")
}

func TestRunRejectsProductionBeforeDatabaseOpen(t *testing.T) {
	args := []string{
		"--run",
		"--migration=" + expectedMigrationName,
		"--target=production",
		"--confirm",
		"--yes-i-understand",
		"--expected-checksum=" + checksum135,
	}
	t.Setenv(envRunControlledMigration, expectedMigrationName)
	t.Setenv(envMigrationConfirm, "true")

	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr, depsThatFailIfCalled(t))
	if code == exitOK {
		t.Fatalf("runCLI exit = %d, want non-zero", code)
	}
	requireContains(t, stderr.String(), "target=production")
	requireContains(t, stderr.String(), "will execute migration: no")
}

func TestRunReturnsNonZeroWhenPreflightRejectsChecksumMismatch(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	args := []string{
		"--run",
		"--migration=" + expectedMigrationName,
		"--target=staging",
		"--confirm",
		"--yes-i-understand",
		"--expected-checksum=" + strings.Repeat("a", 64),
	}
	t.Setenv(envRunControlledMigration, expectedMigrationName)
	t.Setenv(envMigrationConfirm, "true")

	var stdout, stderr bytes.Buffer
	var applyCalled bool
	deps := depsWithSQLMock(t, appliedBefore135(t, embedded), func(context.Context, *sql.DB) error {
		applyCalled = true
		return nil
	})
	code := runCLIWithEmbeddedForTest(args, &stdout, &stderr, deps, embedded)
	if code == exitOK {
		t.Fatalf("runCLIWithEmbeddedForTest exit = %d, want non-zero", code)
	}
	if applyCalled {
		t.Fatal("applyMigrations called after checksum mismatch")
	}
	requireContains(t, stderr.String(), "checksum mismatch")
}

func TestDBStatusAndPreflightPassReadOnlyWithoutApplyMigrations(t *testing.T) {
	for _, mode := range []string{"--db-status", "--preflight"} {
		t.Run(mode, func(t *testing.T) {
			embedded := testEmbeddedMigrations(t, false)
			var applyCalled bool
			deps := depsWithMigrationHistorySQLMock(t, appliedBefore135(t, embedded), func(context.Context, *sql.DB) error {
				applyCalled = true
				return nil
			})
			var stdout, stderr bytes.Buffer
			code := runCLIWithEmbeddedForTest([]string{
				mode,
				"--migration=" + expectedMigrationName,
				"--target=staging",
				"--expected-checksum=" + checksum135,
			}, &stdout, &stderr, deps, embedded)
			if code != exitOK {
				t.Fatalf("runCLIWithEmbeddedForTest exit = %d, stderr=%s", code, stderr.String())
			}
			if applyCalled {
				t.Fatal("ApplyMigrations called by db-status/preflight")
			}
			out := stdout.String()
			requireContains(t, out, "will execute migration: no")
			requireContains(t, out, "read only: yes")
			requireContains(t, out, "select only: yes")
			requireContains(t, out, "preflight result: pass")
			assertNoSensitiveOutput(t, out+stderr.String())
		})
	}
}

func TestDBStatusPreflightFailuresDoNotApply(t *testing.T) {
	tests := []struct {
		name      string
		embedded  []migrationFile
		applied   map[string]string
		checksum  string
		wantError string
	}{
		{
			name:      "134_not_applied",
			embedded:  testEmbeddedMigrations(t, false),
			applied:   map[string]string{"001_init.sql": checksum001, "133_before.sql": checksum133},
			checksum:  checksum135,
			wantError: "not recorded as applied",
		},
		{
			name:     "135_already_applied",
			embedded: testEmbeddedMigrations(t, false),
			applied: map[string]string{
				"001_init.sql":        checksum001,
				"133_before.sql":      checksum133,
				previousMigrationFile: checksum134,
				expectedMigrationFile: checksum135,
			},
			checksum:  checksum135,
			wantError: "already recorded as applied",
		},
		{
			name:      "missing_135_embedded",
			embedded:  removeMigration(testEmbeddedMigrations(t, false), expectedMigrationFile),
			applied:   appliedBefore135(t, removeMigration(testEmbeddedMigrations(t, false), expectedMigrationFile)),
			checksum:  checksum135,
			wantError: "embedded migrations missing",
		},
		{
			name:      "checksum_mismatch",
			embedded:  testEmbeddedMigrations(t, false),
			applied:   appliedBefore135(t, testEmbeddedMigrations(t, false)),
			checksum:  strings.Repeat("a", 64),
			wantError: "checksum mismatch",
		},
		{
			name:      "other_pending",
			embedded:  testEmbeddedMigrations(t, false),
			applied:   map[string]string{"001_init.sql": checksum001, previousMigrationFile: checksum134},
			checksum:  checksum135,
			wantError: "not migration 135",
		},
		{
			name:      "pending_136",
			embedded:  testEmbeddedMigrations(t, true),
			applied:   appliedBefore135(t, testEmbeddedMigrations(t, true)),
			checksum:  checksum135,
			wantError: "not migration 135",
		},
		{
			name:     "history_checksum_mismatch",
			embedded: testEmbeddedMigrations(t, false),
			applied: map[string]string{
				"001_init.sql":        checksum001,
				"133_before.sql":      checksum133,
				previousMigrationFile: strings.Repeat("e", 64),
			},
			checksum:  checksum135,
			wantError: "schema_migrations checksum mismatch",
		},
		{
			name:     "history_missing_from_embedded",
			embedded: testEmbeddedMigrations(t, false),
			applied: map[string]string{
				"001_init.sql":                  checksum001,
				"133_before.sql":                checksum133,
				previousMigrationFile:           checksum134,
				"132_missing_from_embedded.sql": strings.Repeat("e", 64),
			},
			checksum:  checksum135,
			wantError: "embedded migrations do not",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var applyCalled bool
			deps := depsWithSQLMock(t, tt.applied, func(context.Context, *sql.DB) error {
				applyCalled = true
				return nil
			})
			var stdout, stderr bytes.Buffer
			code := runCLIWithEmbeddedForTest([]string{
				"--preflight",
				"--migration=" + expectedMigrationName,
				"--target=staging",
				"--expected-checksum=" + tt.checksum,
			}, &stdout, &stderr, deps, tt.embedded)
			if code == exitOK {
				t.Fatalf("runCLIWithEmbeddedForTest exit = %d, want non-zero", code)
			}
			if applyCalled {
				t.Fatal("ApplyMigrations called after preflight failure")
			}
			requireContains(t, stderr.String(), tt.wantError)
			assertNoSensitiveOutput(t, stdout.String()+stderr.String())
		})
	}
}

func TestDBStatusAndPreflightFailureOutputIsSanitized(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)

	t.Run("preflight_error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		deps := depsWithSQLMock(t, map[string]string{
			"001_init.sql":        checksum001,
			"133_before.sql":      checksum133,
			previousMigrationFile: strings.Repeat("e", 64),
		}, func(context.Context, *sql.DB) error {
			t.Fatal("ApplyMigrations must not be called")
			return nil
		})
		code := runCLIWithEmbeddedForTest([]string{
			"--preflight",
			"--migration=" + expectedMigrationName,
			"--target=staging",
			"--expected-checksum=" + checksum135,
		}, &stdout, &stderr, deps, embedded)
		if code == exitOK {
			t.Fatalf("runCLIWithEmbeddedForTest exit = %d, want non-zero", code)
		}
		requireContains(t, stdout.String(), "preflight result: fail")
		requireContains(t, stderr.String(), "preflight failed")
		assertNoSensitiveOutput(t, stdout.String()+stderr.String())
	})

	t.Run("db_status_error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		deps := runDeps{
			loadConfig: func() (*config.Config, error) {
				return testSafeConfig(), nil
			},
			openDB: func(*config.Config) (*sql.DB, error) {
				db, _, err := sqlmock.New()
				if err != nil {
					t.Fatalf("sqlmock.New() error = %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				return db, nil
			},
			loadHistory: func(context.Context, *sql.DB) (map[string]string, error) {
				return nil, fmt.Errorf("read failed for %s with DSN token secret credential password api_key", previousMigrationFile)
			},
			applyMigration: func(context.Context, *sql.DB, migrationFile) error {
				t.Fatal("applyMigration must not be called")
				return nil
			},
			validateBackup: func() error {
				t.Fatal("validateBackup must not be called")
				return nil
			},
		}
		code := runCLIWithEmbeddedForTest([]string{
			"--db-status",
			"--migration=" + expectedMigrationName,
			"--target=staging",
			"--expected-checksum=" + checksum135,
		}, &stdout, &stderr, deps, embedded)
		if code == exitOK {
			t.Fatalf("runCLIWithEmbeddedForTest exit = %d, want non-zero", code)
		}
		requireContains(t, stderr.String(), "db status failed")
		assertNoSensitiveOutput(t, stdout.String()+stderr.String())
	})
}

func TestPendingPreflightAcceptsOnly135Pending(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, _ := findMigration(embedded, expectedMigrationFile)

	result, err := pendingPreflightFor135(embedded, appliedBefore135(t, embedded), expected.Checksum)
	if err != nil {
		t.Fatalf("pendingPreflight() error = %v", err)
	}
	if got := strings.Join(result.Pending, ","); got != expectedMigrationFile {
		t.Fatalf("pending = %q, want %q", got, expectedMigrationFile)
	}
}

func TestPendingPreflightRejects136OrHigherPending(t *testing.T) {
	embedded := testEmbeddedMigrations(t, true)
	expected, _ := findMigration(embedded, expectedMigrationFile)

	_, err := pendingPreflightFor135(embedded, appliedBefore135(t, embedded), expected.Checksum)
	requireErrorContains(t, err, "not 135_create_chat_workspace_tables.sql")
}

func TestPendingPreflightRejectsOtherPendingMigration(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, _ := findMigration(embedded, expectedMigrationFile)
	applied := appliedBefore135(t, embedded)
	delete(applied, "133_before.sql")

	_, err := pendingPreflightFor135(embedded, applied, expected.Checksum)
	requireErrorContains(t, err, "not "+expectedMigrationFile)
}

func TestPendingPreflightRejects134NotApplied(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, _ := findMigration(embedded, expectedMigrationFile)
	applied := appliedBefore135(t, embedded)
	delete(applied, previousMigrationFile)

	_, err := pendingPreflightFor135(embedded, applied, expected.Checksum)
	requireErrorContains(t, err, previousMigrationFile+" is not recorded as applied")
}

func TestPendingPreflightRejects135AlreadyApplied(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, _ := findMigration(embedded, expectedMigrationFile)
	applied := appliedBefore135(t, embedded)
	applied[expectedMigrationFile] = expected.Checksum

	_, err := pendingPreflightFor135(embedded, applied, expected.Checksum)
	requireErrorContains(t, err, "already recorded as applied")
}

func TestPendingPreflightRejects135ChecksumMismatch(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)

	_, err := pendingPreflightFor135(embedded, appliedBefore135(t, embedded), strings.Repeat("0", 64))
	requireErrorContains(t, err, "checksum mismatch")
}

func TestPendingPreflightRejectsMissing135EmbeddedMigration(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	embedded = removeMigration(embedded, expectedMigrationFile)

	_, err := pendingPreflightFor135(embedded, appliedBefore135(t, embedded), checksum135)
	requireErrorContains(t, err, "embedded migrations missing "+expectedMigrationFile)
}

func TestPendingPreflightRejectsHistoryChecksumMismatch(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	applied := appliedBefore135(t, embedded)
	applied[previousMigrationFile] = strings.Repeat("b", 64)

	_, err := pendingPreflightFor135(embedded, applied, checksum135)
	requireErrorContains(t, err, "schema_migrations checksum mismatch")
}

func TestPendingPreflightRejectsHistoryMissingFromEmbeddedMigrations(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	applied := appliedBefore135(t, embedded)
	applied["132_missing_from_embedded.sql"] = strings.Repeat("c", 64)

	_, err := pendingPreflightFor135(embedded, applied, checksum135)
	requireErrorContains(t, err, "but embedded migrations do not")
}

func TestControlledMigrationSQLRejectsUnsafePatterns(t *testing.T) {
	spec, _ := migrationSpecFromInput(expectedMigrationName)
	tests := map[string]string{
		"DROP":             "DROP TABLE users;",
		"TRUNCATE":         "TrUnCaTe TABLE users;",
		"DELETE FROM":      "DELETE\nFROM users;",
		"ALTER SYSTEM":     "ALTER SYSTEM SET work_mem = '64MB';",
		"CREATE EXTENSION": "CREATE EXTENSION dblink;",
	}
	for name, sql := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateControlledMigrationSQL(spec, sql)
			requireErrorContains(t, err, "blocked SQL pattern")
		})
	}
}

func TestExecuteRunBackupGateFailureDoesNotApply(t *testing.T) {
	embedded := testEmbeddedMigrations(t, false)
	expected, _ := findMigration(embedded, expectedMigrationFile)
	opts := validRunOptions(t)
	opts.expectedChecksum = expected.Checksum

	var applyCalled bool
	deps := depsWithSQLMock(t, appliedBefore135(t, embedded), func(context.Context, *sql.DB) error {
		applyCalled = true
		return nil
	})
	deps.validateBackup = func() error {
		return errors.New("backup unreadable")
	}

	err := executeRun(context.Background(), opts, embedded, deps)
	requireErrorContains(t, err, "backup unreadable")
	if applyCalled {
		t.Fatal("applyMigration called after backup gate failure")
	}
}

func TestOutputDoesNotExposeSensitiveStrings(t *testing.T) {
	t.Setenv(config.AutoMigrationsEnabledEnv, "false")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--status", "--target=production"}, &stdout, &stderr, depsThatFailIfCalled(t))
	if code != exitOK {
		t.Fatalf("runCLI exit = %d, stderr=%s", code, stderr.String())
	}
	out := stdout.String() + stderr.String()
	for _, sensitive := range []string{"password=", "postgres://", "token", "api_key", "secret=", "dsn", "jwt", "session", "password_hash"} {
		if strings.Contains(strings.ToLower(out), sensitive) {
			t.Fatalf("output contains sensitive marker %q: %s", sensitive, out)
		}
	}
}

func TestValidateDatabaseConfigSafeForLocalStage(t *testing.T) {
	if err := validateDatabaseConfigSafeForLocalStage(testSafeConfig()); err != nil {
		t.Fatalf("validateDatabaseConfigSafeForLocalStage() error = %v", err)
	}

	cfg := testSafeConfig()
	cfg.Database.Host = "db.production.example"
	requireErrorContains(t, validateDatabaseConfigSafeForLocalStage(cfg), "loopback")

	cfg = testSafeConfig()
	cfg.Database.DBName = "sub2api_production"
	requireErrorContains(t, validateDatabaseConfigSafeForLocalStage(cfg), "database config name mismatch")
}

func TestValidateBackupFileReadableWithRunner(t *testing.T) {
	repoRoot := t.TempDir()
	outsideDir := t.TempDir()
	backupPath := filepath.Join(outsideDir, "backup.dump")
	if err := os.WriteFile(backupPath, []byte("backup"), 0600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	var listed bool
	err := validateBackupFileReadableWithRunner(backupPath, repoRoot, func(path string) error {
		listed = true
		if path == "" {
			t.Fatal("backup path is empty")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("validateBackupFileReadableWithRunner() error = %v", err)
	}
	if !listed {
		t.Fatal("pg_restore --list runner was not called")
	}

	insideBackup := filepath.Join(repoRoot, "backup.dump")
	if err := os.WriteFile(insideBackup, []byte("backup"), 0600); err != nil {
		t.Fatalf("write repo backup: %v", err)
	}
	requireErrorContains(t, validateBackupFileReadableWithRunner(insideBackup, repoRoot, func(string) error {
		t.Fatal("list runner must not be called for repo-internal backup")
		return nil
	}), "outside the repository")

	requireErrorContains(t, validateBackupFileReadableWithRunner(backupPath, repoRoot, func(string) error {
		return errors.New("pg_restore failed")
	}), "pg_restore --list")
}

func TestLoadEmbeddedMigrationsComputesChecksums(t *testing.T) {
	embedded, err := loadEmbeddedMigrations(fstest.MapFS{
		"002_b.sql": &fstest.MapFile{Data: []byte(" SELECT 2; \n")},
		"001_a.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	})
	if err != nil {
		t.Fatalf("loadEmbeddedMigrations() error = %v", err)
	}
	if got := embedded[0].Name; got != "001_a.sql" {
		t.Fatalf("first migration = %q", got)
	}
	if len(embedded[0].Checksum) != 64 {
		t.Fatalf("checksum length = %d, want 64", len(embedded[0].Checksum))
	}
}

func TestLoadMigrationHistoryUsesReadOnlySelects(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true)).
		RowsWillBeClosed()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT filename, checksum FROM schema_migrations ORDER BY filename")).
		WillReturnRows(sqlmock.NewRows([]string{"filename", "checksum"}).
			AddRow("001_init.sql", checksum001).
			AddRow(previousMigrationFile, checksum134)).
		RowsWillBeClosed()
	mock.ExpectRollback()

	applied, err := loadMigrationHistory(context.Background(), db)
	if err != nil {
		t.Fatalf("loadMigrationHistory() error = %v", err)
	}
	if got := applied[previousMigrationFile]; got != checksum134 {
		t.Fatalf("checksum = %q, want %q", got, checksum134)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLoadMigrationHistoryRejectsMissingSchemaMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false)).
		RowsWillBeClosed()
	mock.ExpectRollback()

	_, err = loadMigrationHistory(context.Background(), db)
	requireErrorContains(t, err, "schema_migrations does not exist")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLoadMigrationHistoryReadFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true)).
		RowsWillBeClosed()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT filename, checksum FROM schema_migrations ORDER BY filename")).
		WillReturnError(errors.New("read failed"))
	mock.ExpectRollback()

	_, err = loadMigrationHistory(context.Background(), db)
	requireErrorContains(t, err, "read schema_migrations")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLoadMigrationHistoryScanFailureClosesRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true)).
		RowsWillBeClosed()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT filename, checksum FROM schema_migrations ORDER BY filename")).
		WillReturnRows(sqlmock.NewRows([]string{"filename"}).
			AddRow("001_init.sql")).
		RowsWillBeClosed()
	mock.ExpectRollback()

	_, err = loadMigrationHistory(context.Background(), db)
	requireErrorContains(t, err, "scan schema_migrations")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func runCLIWithEmbeddedForTest(args []string, stdout, stderr ioWriter, deps runDeps, embedded []migrationFile) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitUsage
	}
	if countTrue(opts.status, opts.dryRun, opts.dbStatus, opts.preflight, opts.run) != 1 {
		return exitUsage
	}
	if opts.status || opts.dryRun {
		printStatus(stdout, "status", opts, embedded)
		return exitOK
	}
	if opts.dbStatus || opts.preflight {
		if err := validateDBReadOnlyRequest(opts); err != nil {
			fmt.Fprintf(stderr, "refused: %s\n", sanitizeOutput(err.Error()))
			return exitUsage
		}
		result, err := executeDBStatus(context.Background(), opts, embedded, deps)
		if err != nil {
			fmt.Fprintf(stderr, "db status failed: %s\n", sanitizeOutput(err.Error()))
			return exitFailed
		}
		mode := "db-status"
		if opts.preflight {
			mode = "preflight"
		}
		printDBStatus(stdout, mode, opts, result)
		if result.PreflightErr != nil {
			fmt.Fprintf(stderr, "%s failed: %s\n", mode, sanitizeOutput(result.PreflightErr.Error()))
			return exitFailed
		}
		return exitOK
	}
	if err := validateRunRequest(opts, lookupEnv); err != nil {
		fmt.Fprintf(stderr, "refused: %s\n", sanitizeOutput(err.Error()))
		return exitUsage
	}
	if err := executeRun(context.Background(), opts, embedded, deps); err != nil {
		fmt.Fprintf(stderr, "migration run failed: %s\n", sanitizeOutput(err.Error()))
		return exitFailed
	}
	fmt.Fprintln(stdout, "controlled migration completed")
	return exitOK
}

type ioWriter interface {
	Write([]byte) (int, error)
}

func validRunOptions(t *testing.T) cliOptions {
	t.Helper()
	embedded := testEmbeddedMigrations(t, false)
	target, _ := findMigration(embedded, expectedMigrationFile)
	return cliOptions{
		run:              true,
		migration:        expectedMigrationName,
		target:           "staging",
		confirm:          true,
		yesIUnderstand:   true,
		expectedChecksum: target.Checksum,
	}
}

func validEnv() map[string]string {
	return map[string]string{
		envRunControlledMigration: expectedMigrationName,
		envMigrationConfirm:       "true",
	}
}

func envLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func requireValidateError(t *testing.T, opts cliOptions, env map[string]string, want string) {
	t.Helper()
	err := validateRunRequest(opts, envLookup(env))
	requireErrorContains(t, err, want)
}

func requireErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want contains %q", want)
	}
	requireContains(t, err.Error(), want)
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("string does not contain %q:\n%s", want, got)
	}
}

func testEmbeddedMigrations(t *testing.T, include136 bool) []migrationFile {
	t.Helper()
	files := []migrationFile{
		{Name: "001_init.sql", Checksum: checksum001},
		{Name: "133_before.sql", Checksum: checksum133},
		{Name: previousMigrationFile, Checksum: checksum134},
		{Name: expectedMigrationFile, Checksum: checksum135},
	}
	if include136 {
		files = append(files, migrationFile{Name: "136_next.sql", Checksum: checksum136})
	}
	return files
}

func removeMigration(files []migrationFile, name string) []migrationFile {
	filtered := make([]migrationFile, 0, len(files))
	for _, file := range files {
		if file.Name == name {
			continue
		}
		filtered = append(filtered, file)
	}
	return filtered
}

func appliedBefore135(t *testing.T, embedded []migrationFile) map[string]string {
	t.Helper()
	applied := make(map[string]string)
	for _, migration := range embedded {
		if migration.Name == expectedMigrationFile || migrationNumber(migration.Name) >= 136 {
			continue
		}
		applied[migration.Name] = migration.Checksum
	}
	return applied
}

func pendingPreflightFor135(embedded []migrationFile, applied map[string]string, checksum string) (preflightResult, error) {
	spec, _ := migrationSpecFromInput(expectedMigrationName)
	return pendingPreflight(embedded, applied, checksum, spec)
}

func testSafeConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:    "127.0.0.1",
			Port:    5432,
			User:    localStagingUser,
			DBName:  localStagingDatabase,
			SSLMode: "disable",
		},
	}
}

func depsThatFailIfCalled(t *testing.T) runDeps {
	t.Helper()
	return runDeps{
		loadConfig: func() (*config.Config, error) {
			t.Fatal("loadConfig must not be called")
			return nil, nil
		},
		openDB: func(*config.Config) (*sql.DB, error) {
			t.Fatal("openDB must not be called")
			return nil, nil
		},
		loadHistory: func(context.Context, *sql.DB) (map[string]string, error) {
			t.Fatal("loadHistory must not be called")
			return nil, nil
		},
		applyMigration: func(context.Context, *sql.DB, migrationFile) error {
			t.Fatal("applyMigration must not be called")
			return nil
		},
		validateBackup: func() error {
			t.Fatal("validateBackup must not be called")
			return nil
		},
	}
}

func depsWithSQLMock(t *testing.T, applied map[string]string, apply func(context.Context, *sql.DB) error) runDeps {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery("SELECT current_database").
		WillReturnRows(sqlmock.NewRows([]string{"database", "user", "addr"}).AddRow(localStagingDatabase, localStagingUser, "127.0.0.1"))
	return runDeps{
		loadConfig: func() (*config.Config, error) {
			return testSafeConfig(), nil
		},
		openDB: func(*config.Config) (*sql.DB, error) {
			return db, nil
		},
		loadHistory: func(context.Context, *sql.DB) (map[string]string, error) {
			return applied, nil
		},
		applyMigration: func(ctx context.Context, db *sql.DB, migration migrationFile) error {
			return apply(ctx, db)
		},
		validateBackup: func() error {
			return nil
		},
	}
}

func depsWithMigrationHistorySQLMock(t *testing.T, applied map[string]string, apply func(context.Context, *sql.DB) error) runDeps {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	rows := sqlmock.NewRows([]string{"filename", "checksum"})
	for filename, checksum := range applied {
		rows.AddRow(filename, checksum)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true)).
		RowsWillBeClosed()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT filename, checksum FROM schema_migrations ORDER BY filename")).
		WillReturnRows(rows).
		RowsWillBeClosed()
	mock.ExpectRollback()

	return runDeps{
		loadConfig: func() (*config.Config, error) {
			return testSafeConfig(), nil
		},
		openDB: func(*config.Config) (*sql.DB, error) {
			return db, nil
		},
		loadHistory: loadMigrationHistory,
		applyMigration: func(ctx context.Context, db *sql.DB, migration migrationFile) error {
			return apply(ctx, db)
		},
		validateBackup: func() error {
			return nil
		},
	}
}

func assertNoSensitiveOutput(t *testing.T, out string) {
	t.Helper()
	lower := strings.ToLower(out)
	for _, sensitive := range []string{"password=", "postgres://", "token", "api_key", "secret=", "dsn"} {
		if strings.Contains(lower, sensitive) {
			t.Fatalf("output contains sensitive marker %q: %s", sensitive, out)
		}
	}
}

func TestExecuteRunPropagatesPreflightFailureWithoutApply(t *testing.T) {
	embedded := testEmbeddedMigrations(t, true)
	expected, _ := findMigration(embedded, expectedMigrationFile)
	opts := validRunOptions(t)
	opts.expectedChecksum = expected.Checksum

	var applyCalled bool
	deps := depsWithSQLMock(t, appliedBefore135(t, embedded), func(context.Context, *sql.DB) error {
		applyCalled = true
		return errors.New("must not apply")
	})
	err := executeRun(context.Background(), opts, embedded, deps)
	requireErrorContains(t, err, "not 135_create_chat_workspace_tables.sql")
	if applyCalled {
		t.Fatal("applyMigrations called after failed preflight")
	}
}
