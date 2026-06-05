package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
)

const (
	envRunControlledMigration = "SUB2API_RUN_CONTROLLED_MIGRATION"
	envMigrationConfirm       = "SUB2API_MIGRATION_CONFIRM"
	envMigrationBackupPath    = "SUB2API_MIGRATION_BACKUP_PATH"

	migration135Name = "135_create_chat_workspace_tables"
	migration136Name = "136_runtime_monitor_and_identity_tables"

	localStagingDatabase = "sub2api_staging"
	localStagingUser     = "sub2api_staging_app"

	exitOK                       = 0
	exitUsage                    = 2
	exitFailed                   = 1
	runTimeout                   = 10 * time.Minute
	databaseKind                 = "postgres"
	defaultTarget                = "local"
	productionTarget             = "production"
	migrationsAdvisoryLockID     = 694208311321144027
	migrationsLockRetryInterval  = 500 * time.Millisecond
	expectedMigration136ByteSize = 10591
)

var migrationFilenamePattern = regexp.MustCompile(`\b([0-9]+)_[A-Za-z0-9_.-]+\.sql\b`)
var createTablePattern = regexp.MustCompile(`(?im)^\s*CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

const schemaMigrationsTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename   TEXT PRIMARY KEY,
	checksum   TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

var allowedTargets = map[string]struct{}{
	"local":   {},
	"staging": {},
}

type migrationSpec struct {
	Name                       string
	File                       string
	PreviousFile               string
	AllowedTargets             map[string]struct{}
	RequireLocalStaging        bool
	RequireBackup              bool
	ExpectedLength             int
	ApprovedTables             []string
	RequireApprovedTablesEmpty bool
}

var controlledMigrationSpecs = []migrationSpec{
	{
		Name:         migration135Name,
		File:         migration135Name + ".sql",
		PreviousFile: "134_add_api_key_allowed_models.sql",
		AllowedTargets: map[string]struct{}{
			"local":   {},
			"staging": {},
		},
		RequireLocalStaging: true,
		RequireBackup:       true,
	},
	{
		Name:                migration136Name,
		File:                migration136Name + ".sql",
		PreviousFile:        migration135Name + ".sql",
		AllowedTargets:      map[string]struct{}{"local": {}, "staging": {}},
		RequireLocalStaging: true,
		RequireBackup:       true,
		ExpectedLength:      expectedMigration136ByteSize,
		ApprovedTables: []string{
			"channel_monitors",
			"channel_monitor_request_templates",
			"channel_monitor_histories",
			"channel_monitor_daily_rollups",
			"channel_monitor_aggregation_watermark",
			"payment_provider_instances",
			"auth_identities",
			"auth_identity_channels",
			"pending_auth_sessions",
			"identity_adoption_decisions",
			"subscription_plans",
		},
		RequireApprovedTablesEmpty: true,
	},
}

type cliOptions struct {
	status           bool
	dryRun           bool
	dbStatus         bool
	preflight        bool
	run              bool
	migration        string
	target           string
	confirm          bool
	yesIUnderstand   bool
	expectedChecksum string
}

type migrationFile struct {
	Name     string
	Checksum string
	Content  string
	Length   int
}

type preflightResult struct {
	Pending []string
}

type runDeps struct {
	loadConfig     func() (*config.Config, error)
	openDB         func(*config.Config) (*sql.DB, error)
	loadHistory    func(context.Context, *sql.DB) (map[string]string, error)
	applyMigration func(context.Context, *sql.DB, migrationFile) error
	validateBackup func() error
}

type dbStatusResult struct {
	Applied                map[string]string
	Preflight              preflightResult
	PreflightErr           error
	SchemaMigrationsRead   bool
	Target                 migrationSpec
	EmbeddedContainsTarget bool
	EmbeddedTargetChecksum string
	HistoryContainsTarget  bool
	HistoryTargetChecksum  string
	PreviousApplied        bool
	HistoryMatchesEmbedded bool
	PendingOtherThanTarget bool
	PendingAtOrAfterTarget bool
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, realRunDeps()))
}

func runCLI(args []string, stdout, stderr io.Writer, deps runDeps) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		printUsage(stderr)
		return exitUsage
	}

	embedded, err := loadEmbeddedMigrations(migrations.FS)
	if err != nil {
		fmt.Fprintf(stderr, "error: load embedded migrations: %v\n", err)
		return exitFailed
	}

	modeCount := countTrue(opts.status, opts.dryRun, opts.dbStatus, opts.preflight, opts.run)
	if modeCount != 1 {
		fmt.Fprintln(stderr, "error: choose exactly one mode: --status, --dry-run, --db-status, --preflight, or --run")
		printUsage(stderr)
		return exitUsage
	}

	if opts.status || opts.dryRun {
		mode := "status"
		if opts.dryRun {
			mode = "dry-run"
		}
		printStatus(stdout, mode, opts, embedded)
		return exitOK
	}

	if opts.dbStatus || opts.preflight {
		mode := "db-status"
		if opts.preflight {
			mode = "preflight"
		}
		if err := validateDBReadOnlyRequest(opts); err != nil {
			fmt.Fprintf(stderr, "mode: %s\n", mode)
			fmt.Fprintln(stderr, "will execute migration: no")
			fmt.Fprintf(stderr, "refused: %s\n", sanitizeOutput(err.Error()))
			return exitUsage
		}
		result, err := executeDBStatus(context.Background(), opts, embedded, deps)
		if err != nil {
			fmt.Fprintf(stderr, "db status failed: %s\n", sanitizeOutput(err.Error()))
			return exitFailed
		}
		printDBStatus(stdout, mode, opts, result)
		if result.PreflightErr != nil {
			fmt.Fprintf(stderr, "%s failed: %s\n", mode, sanitizeOutput(result.PreflightErr.Error()))
			return exitFailed
		}
		return exitOK
	}

	if err := validateRunRequest(opts, lookupEnv); err != nil {
		fmt.Fprintln(stderr, "mode: run")
		fmt.Fprintln(stderr, "will execute migration: no")
		fmt.Fprintf(stderr, "refused: %s\n", sanitizeOutput(err.Error()))
		printRunRequirements(stderr)
		return exitUsage
	}

	if err := executeRun(context.Background(), opts, embedded, deps); err != nil {
		fmt.Fprintf(stderr, "migration run failed: %s\n", sanitizeOutput(err.Error()))
		return exitFailed
	}
	fmt.Fprintln(stdout, "mode: run")
	fmt.Fprintln(stdout, "will execute migration: yes")
	fmt.Fprintln(stdout, "controlled migration completed")
	return exitOK
}

func parseOptions(args []string, output io.Writer) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("sub2api-migrate", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&opts.status, "status", false, "show embedded migration status without executing SQL")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show dry-run status without executing SQL")
	fs.BoolVar(&opts.dbStatus, "db-status", false, "read target database migration status without executing SQL")
	fs.BoolVar(&opts.preflight, "preflight", false, "read target database and check whether controlled run prerequisites pass")
	fs.BoolVar(&opts.run, "run", false, "run controlled migrations after explicit confirmation")
	fs.StringVar(&opts.migration, "migration", "", "expected migration name")
	fs.StringVar(&opts.target, "target", defaultTarget, "target label: local or staging; production is refused")
	fs.BoolVar(&opts.confirm, "confirm", false, "confirm controlled migration execution")
	fs.BoolVar(&opts.yesIUnderstand, "yes-i-understand", false, "acknowledge this can modify the target database")
	fs.StringVar(&opts.expectedChecksum, "expected-checksum", "", "frozen checksum for the target migration")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() > 0 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	opts.target = normalizeTarget(opts.target)
	opts.migration = strings.TrimSpace(opts.migration)
	opts.expectedChecksum = strings.TrimSpace(opts.expectedChecksum)
	return opts, nil
}

func printStatus(w io.Writer, mode string, opts cliOptions, embedded []migrationFile) {
	autoMigrations, autoErr := config.AutoMigrationsEnabled()
	chatWorkspace := config.ChatWorkspaceEnabled()
	target := normalizeTarget(opts.target)

	spec, ok := migrationSpecFromInput(opts.migration)
	if !ok {
		spec = controlledMigrationSpecs[0]
	}
	targetMigration, containsTarget := findMigration(embedded, spec.File)
	pending := []string{spec.File}
	fmt.Fprintf(w, "mode: %s\n", mode)
	fmt.Fprintln(w, "will execute migration: no")
	fmt.Fprintf(w, "target: %s\n", target)
	fmt.Fprintf(w, "database: %s\n", databaseKind)
	fmt.Fprintln(w, "database connection: not opened")
	fmt.Fprintf(w, "allowlisted migrations: %s\n", allowlistedMigrationList())
	fmt.Fprintf(w, "selected migration: %s\n", displayMigrationName(spec.File))
	fmt.Fprintf(w, "embedded contains %s: %t\n", displayMigrationName(spec.File), containsTarget)
	if containsTarget {
		fmt.Fprintf(w, "%s checksum: %s\n", displayMigrationName(spec.File), targetMigration.Checksum)
	}
	fmt.Fprintln(w, "schema_migrations status: not read in this mode")
	fmt.Fprintf(w, "%s applied: unknown (requires target database preflight)\n", displayMigrationName(spec.PreviousFile))
	fmt.Fprintf(w, "%s not applied: unknown (requires target database preflight)\n", displayMigrationName(spec.File))
	fmt.Fprintf(w, "pending migrations: %s (expected pending set for controlled run only; real state requires preflight)\n", displayMigrationList(pending))
	fmt.Fprintln(w, "pending other than selected migration: unknown (requires target database preflight)")
	fmt.Fprintf(w, "pending %s or higher: unknown (requires target database preflight)\n", displayMigrationName(spec.File))
	if autoErr != nil {
		fmt.Fprintf(w, "%s: invalid (%v)\n", config.AutoMigrationsEnabledEnv, autoErr)
	} else {
		fmt.Fprintf(w, "%s: %t\n", config.AutoMigrationsEnabledEnv, autoMigrations)
	}
	fmt.Fprintf(w, "%s: %t\n", config.ChatWorkspaceEnabledEnv, chatWorkspace)
}

func validateDBReadOnlyRequest(opts cliOptions) error {
	if strings.TrimSpace(opts.migration) == "" {
		return errors.New("missing --migration")
	}
	if _, err := validateTarget(opts.target); err != nil {
		return err
	}
	if _, err := controlledMigrationSpecFromInput(opts.migration); err != nil {
		return err
	}
	if err := validateChecksumFormat(opts.expectedChecksum); err != nil {
		return err
	}
	return nil
}

func executeDBStatus(ctx context.Context, opts cliOptions, embedded []migrationFile, deps runDeps) (dbStatusResult, error) {
	if deps.loadConfig == nil || deps.openDB == nil || deps.loadHistory == nil {
		return dbStatusResult{}, errors.New("db status dependencies are not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	cfg, err := deps.loadConfig()
	if err != nil {
		return dbStatusResult{}, fmt.Errorf("load config: %w", err)
	}
	if err := validateDatabaseConfigSafeForLocalStage(cfg); err != nil {
		return dbStatusResult{}, err
	}
	db, err := deps.openDB(cfg)
	if err != nil {
		return dbStatusResult{}, fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	applied, err := deps.loadHistory(ctx, db)
	if err != nil {
		return dbStatusResult{}, fmt.Errorf("load migration history: %w", err)
	}
	spec, err := controlledMigrationSpecFromInput(opts.migration)
	if err != nil {
		return dbStatusResult{}, err
	}
	return buildDBStatusResult(embedded, applied, opts.expectedChecksum, spec), nil
}

func buildDBStatusResult(embedded []migrationFile, applied map[string]string, expectedChecksum string, spec migrationSpec) dbStatusResult {
	result := dbStatusResult{
		Applied:              applied,
		SchemaMigrationsRead: true,
		Target:               spec,
	}
	if targetMigration, ok := findMigration(embedded, spec.File); ok {
		result.EmbeddedContainsTarget = true
		result.EmbeddedTargetChecksum = targetMigration.Checksum
	}
	if checksum, ok := applied[spec.File]; ok {
		result.HistoryContainsTarget = true
		result.HistoryTargetChecksum = checksum
	}
	result.PreviousApplied = applied[spec.PreviousFile] != ""

	preflight, err := pendingPreflight(embedded, applied, expectedChecksum, spec)
	result.Preflight = preflight
	result.PreflightErr = err

	embeddedByName := make(map[string]migrationFile, len(embedded))
	for _, migration := range embedded {
		embeddedByName[migration.Name] = migration
		if _, ok := applied[migration.Name]; !ok {
			if migration.Name != spec.File {
				result.PendingOtherThanTarget = true
			}
			if migrationNumber(migration.Name) >= migrationNumber(spec.File) {
				result.PendingAtOrAfterTarget = true
			}
		}
	}
	result.HistoryMatchesEmbedded = true
	for name, checksum := range applied {
		migration, ok := embeddedByName[name]
		if !ok || migration.Checksum != checksum {
			result.HistoryMatchesEmbedded = false
			break
		}
	}
	return result
}

func printDBStatus(w io.Writer, mode string, opts cliOptions, result dbStatusResult) {
	autoMigrations, autoErr := config.AutoMigrationsEnabled()
	chatWorkspace := config.ChatWorkspaceEnabled()
	fmt.Fprintf(w, "mode: %s\n", mode)
	fmt.Fprintln(w, "will execute migration: no")
	fmt.Fprintln(w, "read only: yes")
	fmt.Fprintln(w, "select only: yes")
	fmt.Fprintf(w, "target: %s\n", opts.target)
	fmt.Fprintf(w, "database: %s\n", databaseKind)
	fmt.Fprintln(w, "database connection: opened; connection details hidden")
	fmt.Fprintf(w, "selected migration: %s\n", displayMigrationName(result.Target.File))
	fmt.Fprintf(w, "embedded contains %s: %t\n", displayMigrationName(result.Target.File), result.EmbeddedContainsTarget)
	if result.EmbeddedContainsTarget {
		fmt.Fprintf(w, "%s embedded checksum: %s\n", displayMigrationName(result.Target.File), result.EmbeddedTargetChecksum)
	}
	fmt.Fprintf(w, "schema_migrations readable: %t\n", result.SchemaMigrationsRead)
	fmt.Fprintf(w, "%s applied: %t\n", displayMigrationName(result.Target.PreviousFile), result.PreviousApplied)
	fmt.Fprintf(w, "%s applied: %t\n", displayMigrationName(result.Target.File), result.HistoryContainsTarget)
	if result.HistoryContainsTarget {
		fmt.Fprintf(w, "%s history checksum: %s\n", displayMigrationName(result.Target.File), result.HistoryTargetChecksum)
	}
	fmt.Fprintf(w, "pending migrations: %s\n", displayMigrationList(result.Preflight.Pending))
	fmt.Fprintf(w, "pending other than selected migration: %t\n", result.PendingOtherThanTarget)
	fmt.Fprintf(w, "pending %s or higher: %t\n", displayMigrationName(result.Target.File), result.PendingAtOrAfterTarget)
	fmt.Fprintf(w, "migration history matches embedded migrations: %t\n", result.HistoryMatchesEmbedded)
	if result.PreflightErr == nil {
		fmt.Fprintln(w, "preflight result: pass")
	} else {
		fmt.Fprintf(w, "preflight result: fail (%s)\n", sanitizeOutput(result.PreflightErr.Error()))
	}
	if autoErr != nil {
		fmt.Fprintf(w, "%s: invalid (%v)\n", config.AutoMigrationsEnabledEnv, autoErr)
	} else {
		fmt.Fprintf(w, "%s: %t\n", config.AutoMigrationsEnabledEnv, autoMigrations)
	}
	fmt.Fprintf(w, "%s: %t\n", config.ChatWorkspaceEnabledEnv, chatWorkspace)
	fmt.Fprintln(w, "sensitive information hidden: yes")
}

func validateRunRequest(opts cliOptions, getenv func(string) (string, bool)) error {
	var missing []string
	if !opts.run {
		missing = append(missing, "--run")
	}
	if !opts.confirm {
		missing = append(missing, "--confirm")
	}
	if !opts.yesIUnderstand {
		missing = append(missing, "--yes-i-understand")
	}
	if strings.TrimSpace(opts.migration) == "" {
		missing = append(missing, "--migration")
	}
	if strings.TrimSpace(opts.expectedChecksum) == "" {
		missing = append(missing, "--expected-checksum")
	}
	if envMigration, ok := getenv(envRunControlledMigration); !ok || strings.TrimSpace(envMigration) == "" {
		missing = append(missing, envRunControlledMigration)
	}
	if envConfirm, ok := getenv(envMigrationConfirm); !ok || strings.TrimSpace(envConfirm) == "" {
		missing = append(missing, envMigrationConfirm)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing confirmations: %s", strings.Join(missing, ", "))
	}
	if err := validateChecksumFormat(opts.expectedChecksum); err != nil {
		return err
	}

	spec, err := controlledMigrationSpecFromInput(opts.migration)
	if err != nil {
		return err
	}
	target, err := validateTarget(opts.target)
	if err != nil {
		return err
	}
	if _, ok := spec.AllowedTargets[target]; !ok {
		return fmt.Errorf("invalid target %q for %s: allowed targets are %s", target, displayMigrationName(spec.File), allowedTargetListFor(spec))
	}
	envMigration, _ := getenv(envRunControlledMigration)
	if !migrationInputMatchesSpec(envMigration, spec) {
		return fmt.Errorf("%s mismatch: got %q, want %q", envRunControlledMigration, envMigration, spec.Name)
	}
	envConfirm, _ := getenv(envMigrationConfirm)
	confirmed, err := parseBoolStrict(envConfirm)
	if err != nil {
		return fmt.Errorf("invalid %s value %q: use 1/true/yes/on", envMigrationConfirm, envConfirm)
	}
	if !confirmed {
		return fmt.Errorf("%s must be true/1/yes/on", envMigrationConfirm)
	}
	return nil
}

func executeRun(ctx context.Context, opts cliOptions, embedded []migrationFile, deps runDeps) error {
	if deps.loadConfig == nil || deps.openDB == nil || deps.loadHistory == nil || deps.applyMigration == nil || deps.validateBackup == nil {
		return errors.New("migration run dependencies are not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	cfg, err := deps.loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := validateDatabaseConfigSafeForLocalStage(cfg); err != nil {
		return err
	}
	db, err := deps.openDB(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	applied, err := deps.loadHistory(ctx, db)
	if err != nil {
		return fmt.Errorf("load migration history: %w", err)
	}
	spec, err := controlledMigrationSpecFromInput(opts.migration)
	if err != nil {
		return err
	}
	result, err := pendingPreflight(embedded, applied, opts.expectedChecksum, spec)
	if err != nil {
		return fmt.Errorf("pending migration preflight: %w", err)
	}
	if len(result.Pending) != 1 || result.Pending[0] != spec.File {
		return fmt.Errorf("pending migration preflight: refusing to run with pending set %v", result.Pending)
	}
	targetMigration, ok := findMigration(embedded, spec.File)
	if !ok {
		return fmt.Errorf("embedded migrations missing %s", spec.File)
	}
	if err := validateControlledMigrationGate(ctx, db, spec, targetMigration, cfg.Database.Host, deps.validateBackup); err != nil {
		return fmt.Errorf("controlled migration gate: %w", err)
	}
	if err := deps.applyMigration(ctx, db, targetMigration); err != nil {
		return fmt.Errorf("apply migration %s: %w", spec.File, err)
	}
	return nil
}

func pendingPreflight(embedded []migrationFile, applied map[string]string, frozenChecksum string, spec migrationSpec) (preflightResult, error) {
	if strings.TrimSpace(frozenChecksum) == "" {
		return preflightResult{}, fmt.Errorf("missing frozen checksum for %s", displayMigrationName(spec.File))
	}
	if err := validateChecksumFormat(frozenChecksum); err != nil {
		return preflightResult{}, err
	}

	embeddedByName := make(map[string]migrationFile, len(embedded))
	for _, migration := range embedded {
		embeddedByName[migration.Name] = migration
	}
	for name := range applied {
		embeddedMigration, ok := embeddedByName[name]
		if !ok {
			return preflightResult{}, fmt.Errorf("schema_migrations contains %s, but embedded migrations do not", name)
		}
		if embeddedMigration.Checksum != applied[name] {
			return preflightResult{}, fmt.Errorf("schema_migrations checksum mismatch for %s", name)
		}
	}

	prev, ok := embeddedByName[spec.PreviousFile]
	if !ok {
		return preflightResult{}, fmt.Errorf("embedded migrations missing %s", spec.PreviousFile)
	}
	prevChecksum, ok := applied[spec.PreviousFile]
	if !ok {
		return preflightResult{}, fmt.Errorf("%s is not recorded as applied", spec.PreviousFile)
	}
	if prevChecksum != prev.Checksum {
		return preflightResult{}, fmt.Errorf("%s checksum mismatch", spec.PreviousFile)
	}

	targetMigration, ok := embeddedByName[spec.File]
	if !ok {
		return preflightResult{}, fmt.Errorf("embedded migrations missing %s", spec.File)
	}
	if targetMigration.Checksum != frozenChecksum {
		return preflightResult{}, fmt.Errorf("%s checksum mismatch: embedded=%s frozen=%s", spec.File, targetMigration.Checksum, frozenChecksum)
	}
	if _, ok := applied[spec.File]; ok {
		return preflightResult{}, fmt.Errorf("%s is already recorded as applied", spec.File)
	}

	pending := make([]string, 0)
	for _, migration := range embedded {
		if _, ok := applied[migration.Name]; ok {
			continue
		}
		pending = append(pending, migration.Name)
	}
	sort.Strings(pending)
	for _, name := range pending {
		if name != spec.File {
			return preflightResult{}, fmt.Errorf("pending migration %s is not %s; refusing to run all pending migrations", name, spec.File)
		}
	}
	if len(pending) != 1 {
		return preflightResult{}, fmt.Errorf("expected only %s pending, got %v", spec.File, pending)
	}
	return preflightResult{Pending: pending}, nil
}

func validateChecksumFormat(checksum string) error {
	if strings.TrimSpace(checksum) != checksum {
		return errors.New("expected checksum must not contain leading or trailing whitespace")
	}
	if len(checksum) != sha256.Size*2 {
		return fmt.Errorf("expected checksum must be a full %d-character sha256 hex string", sha256.Size*2)
	}
	for _, r := range checksum {
		if r < '0' || r > '9' && r < 'a' || r > 'f' {
			return errors.New("expected checksum must use lowercase sha256 hex characters")
		}
	}
	return nil
}

func loadMigrationHistory(ctx context.Context, db *sql.DB) (map[string]string, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin read-only migration history transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = $1
)
`, "schema_migrations").Scan(&exists); err != nil {
		return nil, fmt.Errorf("check schema_migrations: %w", err)
	}
	if !exists {
		return nil, errors.New("schema_migrations does not exist")
	}

	rows, err := tx.QueryContext(ctx, "SELECT filename, checksum FROM schema_migrations ORDER BY filename")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[string]string)
	for rows.Next() {
		var filename, checksum string
		if err := rows.Scan(&filename, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[filename] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

func loadEmbeddedMigrations(fsys fs.FS) ([]migrationFile, error) {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	migrations := make([]migrationFile, 0, len(files))
	for _, name := range files {
		contentBytes, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		content := strings.TrimSpace(string(contentBytes))
		sum := sha256.Sum256([]byte(content))
		migrations = append(migrations, migrationFile{
			Name:     name,
			Checksum: hex.EncodeToString(sum[:]),
			Content:  content,
			Length:   len(contentBytes),
		})
	}
	return migrations, nil
}

func realRunDeps() runDeps {
	return runDeps{
		loadConfig: config.Load,
		openDB: func(cfg *config.Config) (*sql.DB, error) {
			return sql.Open(databaseKind, cfg.Database.DSNWithTimezone(cfg.Timezone))
		},
		loadHistory:    loadMigrationHistory,
		applyMigration: applySelectedMigration,
		validateBackup: validateBackupFileReadable,
	}
}

func normalizeTarget(raw string) string {
	target := strings.ToLower(strings.TrimSpace(raw))
	if target == "" {
		return defaultTarget
	}
	return target
}

func validateTarget(raw string) (string, error) {
	target := normalizeTarget(raw)
	if target == productionTarget {
		return "", errors.New("target=production is not allowed; this migration CLI is local/staging only")
	}
	if _, ok := allowedTargets[target]; !ok {
		return "", fmt.Errorf("invalid target %q: allowed targets are %s", target, allowedTargetList())
	}
	return target, nil
}

func migrationSpecFromInput(raw string) (migrationSpec, bool) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return migrationSpec{}, false
	}
	if isUnsafeMigrationInput(normalized) {
		return migrationSpec{}, false
	}
	for _, spec := range controlledMigrationSpecs {
		if normalized == spec.Name || normalized == spec.File {
			return spec, true
		}
	}
	return migrationSpec{}, false
}

func controlledMigrationSpecFromInput(raw string) (migrationSpec, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return migrationSpec{}, errors.New("missing --migration")
	}
	if isUnsafeMigrationInput(normalized) {
		return migrationSpec{}, errors.New("migration selection must be an exact allowlisted name, not a path")
	}
	spec, ok := migrationSpecFromInput(normalized)
	if !ok {
		return migrationSpec{}, fmt.Errorf("unknown controlled migration %q; allowlisted migrations are %s", normalized, allowlistedMigrationList())
	}
	return spec, nil
}

func isUnsafeMigrationInput(raw string) bool {
	return strings.Contains(raw, "..") || strings.ContainsAny(raw, `/\`)
}

func migrationInputMatchesSpec(raw string, spec migrationSpec) bool {
	normalized := strings.TrimSpace(raw)
	return normalized == spec.Name
}

func allowlistedMigrationList() string {
	names := make([]string, 0, len(controlledMigrationSpecs))
	for _, spec := range controlledMigrationSpecs {
		names = append(names, spec.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func allowedTargetListFor(spec migrationSpec) string {
	targets := make([]string, 0, len(spec.AllowedTargets))
	for target := range spec.AllowedTargets {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return strings.Join(targets, ", ")
}

func validateControlledMigrationGate(ctx context.Context, db *sql.DB, spec migrationSpec, migration migrationFile, dbHost string, validateBackup func() error) error {
	if spec.ExpectedLength > 0 && migration.Length != spec.ExpectedLength {
		return fmt.Errorf("%s length mismatch: got %d, want %d", displayMigrationName(spec.File), migration.Length, spec.ExpectedLength)
	}
	if err := validateControlledMigrationSQL(spec, migration.Content); err != nil {
		return err
	}
	if spec.RequireBackup {
		if validateBackup == nil {
			return errors.New("backup gate is not configured")
		}
		if err := validateBackup(); err != nil {
			return err
		}
	}
	if spec.RequireLocalStaging {
		if err := validateLocalStagingIdentity(ctx, db, dbHost); err != nil {
			return err
		}
	}
	if spec.RequireApprovedTablesEmpty {
		if err := validateTablesAbsent(ctx, db, spec.ApprovedTables); err != nil {
			return err
		}
	}
	return nil
}

func validateControlledMigrationSQL(spec migrationSpec, content string) error {
	if len(spec.ApprovedTables) > 0 {
		createdTables := map[string]struct{}{}
		for _, match := range createTablePattern.FindAllStringSubmatch(content, -1) {
			createdTables[match[1]] = struct{}{}
		}
		for _, table := range spec.ApprovedTables {
			if _, ok := createdTables[table]; !ok {
				return fmt.Errorf("%s does not create approved table %s", displayMigrationName(spec.File), table)
			}
		}
		for table := range createdTables {
			if !containsString(spec.ApprovedTables, table) {
				return fmt.Errorf("%s creates non-allowlisted table %s", displayMigrationName(spec.File), table)
			}
		}
	}
	dangerous := map[string]string{
		"DROP":             `(?i)\bDROP\b`,
		"TRUNCATE":         `(?i)\bTRUNCATE\b`,
		"DELETE FROM":      `(?i)\bDELETE\s+FROM\b`,
		"dangerous UPDATE": `(?i)\bUPDATE\s+(users|settings|usage|payment)`,
		"dangerous INSERT": `(?i)\bINSERT\s+INTO\s+(users|settings|usage|payment|schema_migrations)`,
		"ALTER SYSTEM":     `(?i)\bALTER\s+SYSTEM\b`,
		"CREATE EXTENSION": `(?i)\bCREATE\s+EXTENSION\b`,
	}
	for label, pattern := range dangerous {
		if regexp.MustCompile(pattern).MatchString(content) {
			return fmt.Errorf("%s contains blocked SQL pattern: %s", displayMigrationName(spec.File), label)
		}
	}
	return nil
}

func validateBackupFileReadable() error {
	backupPath := strings.TrimSpace(os.Getenv(envMigrationBackupPath))
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	return validateBackupFileReadableWithRunner(backupPath, repoRoot, func(path string) error {
		cmd := exec.Command("pg_restore", "--list", path)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		return cmd.Run()
	})
}

func validateBackupFileReadableWithRunner(backupPath, repoRoot string, listBackup func(string) error) error {
	if backupPath == "" {
		return fmt.Errorf("%s is required for this migration", envMigrationBackupPath)
	}
	absBackup, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}
	if isPathInside(absRepo, absBackup) {
		return errors.New("backup file must be outside the repository")
	}
	info, err := os.Stat(absBackup)
	if err != nil {
		return fmt.Errorf("backup file is not readable: %w", err)
	}
	if info.IsDir() {
		return errors.New("backup path points to a directory")
	}
	if info.Size() <= 0 {
		return errors.New("backup file is empty")
	}
	if listBackup == nil {
		return errors.New("backup list verifier is not configured")
	}
	if err := listBackup(absBackup); err != nil {
		return fmt.Errorf("backup file is not readable by pg_restore --list: %w", err)
	}
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("read working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository root not found")
		}
		dir = parent
	}
}

func isPathInside(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func validateDatabaseConfigSafeForLocalStage(cfg *config.Config) error {
	if cfg == nil {
		return errors.New("database config is not configured")
	}
	db := cfg.Database
	if !isLoopbackDBHost(db.Host) {
		return errors.New("database config host must be loopback for this local/staging migration CLI")
	}
	if db.DBName != localStagingDatabase {
		return fmt.Errorf("database config name mismatch: got %q, want %q", sanitizeOutput(db.DBName), localStagingDatabase)
	}
	if db.User != localStagingUser {
		return fmt.Errorf("database config user mismatch: got %q, want %q", sanitizeOutput(db.User), localStagingUser)
	}
	if looksProductionDatabaseValue(db.Host) || looksProductionDatabaseValue(db.DBName) || looksProductionDatabaseValue(db.User) {
		return errors.New("database config appears production-like; refusing controlled migration")
	}
	return nil
}

func looksProductionDatabaseValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "prod") || strings.Contains(normalized, "production")
}

func validateLocalStagingIdentity(ctx context.Context, db *sql.DB, dbHost string) error {
	if !isLoopbackDBHost(dbHost) {
		return fmt.Errorf("database config host is not loopback")
	}
	var databaseName, userName, serverAddr string
	if err := db.QueryRowContext(ctx, `
SELECT current_database(), current_user, COALESCE(inet_server_addr()::text, '')
`).Scan(&databaseName, &userName, &serverAddr); err != nil {
		return fmt.Errorf("read database identity: %w", err)
	}
	if databaseName != localStagingDatabase {
		return fmt.Errorf("database mismatch: got %q, want %q", databaseName, localStagingDatabase)
	}
	if userName != localStagingUser {
		return fmt.Errorf("database user mismatch: got %q, want %q", userName, localStagingUser)
	}
	return nil
}

func isLoopbackDBHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	normalized = strings.TrimPrefix(strings.TrimSuffix(normalized, "]"), "[")
	switch normalized {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func validateTablesAbsent(ctx context.Context, db *sql.DB, tables []string) error {
	for _, table := range tables {
		var exists bool
		if err := db.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = $1
)
`, table).Scan(&exists); err != nil {
			return fmt.Errorf("check table %s: %w", table, err)
		}
		if exists {
			return fmt.Errorf("table %s already exists", table)
		}
	}
	return nil
}

func applySelectedMigration(ctx context.Context, db *sql.DB, migration migrationFile) error {
	if db == nil {
		return errors.New("nil sql db")
	}
	if strings.TrimSpace(migration.Content) == "" {
		return fmt.Errorf("migration %s is empty", migration.Name)
	}
	if err := pgAdvisoryLock(ctx, db); err != nil {
		return err
	}
	defer func() { _ = pgAdvisoryUnlock(context.Background(), db) }()

	if _, err := db.ExecContext(ctx, schemaMigrationsTableDDL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var existing string
	rowErr := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", migration.Name).Scan(&existing)
	if rowErr == nil {
		if existing != migration.Checksum {
			return fmt.Errorf("%s checksum mismatch", migration.Name)
		}
		return fmt.Errorf("%s is already recorded as applied", migration.Name)
	}
	if !errors.Is(rowErr, sql.ErrNoRows) {
		return fmt.Errorf("check migration %s: %w", migration.Name, rowErr)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Name, err)
	}
	if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", migration.Name, err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)", migration.Name, migration.Checksum); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("commit migration %s: %w", migration.Name, err)
	}
	return nil
}

func pgAdvisoryLock(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(migrationsLockRetryInterval)
	defer ticker.Stop()
	for {
		var locked bool
		if err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", migrationsAdvisoryLockID).Scan(&locked); err != nil {
			return fmt.Errorf("acquire migrations lock: %w", err)
		}
		if locked {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("acquire migrations lock: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func pgAdvisoryUnlock(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrationsAdvisoryLockID)
	if err != nil {
		return fmt.Errorf("release migrations lock: %w", err)
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func findMigration(migrations []migrationFile, name string) (migrationFile, bool) {
	for _, migration := range migrations {
		if migration.Name == name {
			return migration, true
		}
	}
	return migrationFile{}, false
}

func migrationNumber(name string) int {
	prefix := name
	if idx := strings.Index(prefix, "_"); idx >= 0 {
		prefix = prefix[:idx]
	}
	var n int
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func parseBoolStrict(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errors.New("invalid bool")
	}
}

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func allowedTargetList() string {
	targets := make([]string, 0, len(allowedTargets))
	for target := range allowedTargets {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return strings.Join(targets, ", ")
}

func countTrue(values ...bool) int {
	var count int
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func displayMigrationName(name string) string {
	if idx := strings.Index(name, "_"); idx > 0 {
		prefix := name[:idx]
		for _, r := range prefix {
			if r < '0' || r > '9' {
				return sanitizeOutput(name)
			}
		}
		return "migration " + prefix
	}
	return sanitizeOutput(name)
}

func displayMigrationList(names []string) string {
	display := make([]string, 0, len(names))
	for _, name := range names {
		display = append(display, displayMigrationName(name))
	}
	return strings.Join(display, ", ")
}

func sanitizeOutput(text string) string {
	text = migrationFilenamePattern.ReplaceAllString(text, "migration $1")
	for _, marker := range []string{
		"api_key",
		"apikey",
		"api key",
		"bearer",
		"dsn",
		"jwt",
		"password",
		"password_hash",
		"private key",
		"session",
		"token",
		"secret",
		"credential",
		"postgres://",
	} {
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(marker))
		text = re.ReplaceAllString(text, "[redacted]")
	}
	return text
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  sub2api-migrate --status [--target=local|staging]")
	fmt.Fprintln(w, "  sub2api-migrate --dry-run [--target=local|staging]")
	fmt.Fprintln(w, "  sub2api-migrate --db-status --migration=<allowlisted> --target=local|staging --expected-checksum=<sha256>")
	fmt.Fprintln(w, "  sub2api-migrate --preflight --migration=<allowlisted> --target=local|staging --expected-checksum=<sha256>")
	fmt.Fprintln(w, "  sub2api-migrate --run --migration=<allowlisted> --target=local|staging --confirm --yes-i-understand --expected-checksum=<sha256>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "safety:")
	fmt.Fprintln(w, "  local/staging only; production is refused")
	fmt.Fprintln(w, "  default mode never writes DB; --run requires every gate listed below")
}

func printRunRequirements(w io.Writer) {
	fmt.Fprintln(w, "required run confirmations:")
	fmt.Fprintln(w, "  --run")
	fmt.Fprintf(w, "  --migration=<one of: %s>\n", allowlistedMigrationList())
	fmt.Fprintln(w, "  --target=local|staging")
	fmt.Fprintln(w, "  --confirm")
	fmt.Fprintln(w, "  --yes-i-understand")
	fmt.Fprintln(w, "  --expected-checksum=<frozen migration sha256>")
	fmt.Fprintf(w, "  %s=<selected migration>\n", envRunControlledMigration)
	fmt.Fprintf(w, "  %s=true\n", envMigrationConfirm)
	fmt.Fprintf(w, "  %s=<repo-outside backup readable by pg_restore --list>\n", envMigrationBackupPath)
	fmt.Fprintln(w, "  production targets/configs are not supported by this CLI")
}
