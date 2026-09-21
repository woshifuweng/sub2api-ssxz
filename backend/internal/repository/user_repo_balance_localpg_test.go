//go:build localpg && !integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Explicit opt-in, independent of the Docker integration TestMain. Never accepts
// a DSN: initdb creates a fresh cluster inside t.TempDir, bound only to loopback.
func isolatedLocalPostgres(t *testing.T) *sql.DB {
	t.Helper()
	bin := os.Getenv("SUB2API_LOCAL_PG_BIN")
	require.NotEmpty(t, bin, "set SUB2API_LOCAL_PG_BIN to the PostgreSQL bin directory")
	dir := t.TempDir()
	data := filepath.Join(dir, "cluster")
	run := func(name string, args ...string) {
		t.Helper()
		// A spawned Windows server can inherit pipe handles after pg_ctl exits.
		// Use a file so waiting for pg_ctl does not wait for the server's EOF.
		logPath := filepath.Join(dir, name+".log")
		out, err := os.Create(logPath)
		require.NoError(t, err)
		cmd := exec.Command(filepath.Join(bin, name), args...)
		cmd.Stdout, cmd.Stderr = out, out
		err = cmd.Run()
		require.NoError(t, out.Close())
		if err != nil {
			output, _ := os.ReadFile(logPath)
			t.Fatalf("%s: %v: %s", name, err, output)
		}
	}
	run("initdb", "-D", data, "-U", "balance_test", "--auth=trust", "--no-locale", "--encoding=UTF8")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	options := fmt.Sprintf("-h 127.0.0.1 -p %d -c max_connections=10 -c shared_buffers=16MB", port)
	// Register cleanup before starting so a partially successful start is stopped too.
	t.Cleanup(func() {
		out, err := exec.Command(filepath.Join(bin, "pg_ctl"), "-D", data, "-m", "immediate", "-w", "-t", "20", "stop").CombinedOutput()
		if err != nil {
			t.Errorf("stop isolated PostgreSQL: %v: %s", err, out)
		}
	})
	run("pg_ctl", "-D", data, "-l", filepath.Join(dir, "postgres.log"), "-o", options, "-w", "-t", "20", "start")
	dsn := fmt.Sprintf("host=127.0.0.1 port=%d user=balance_test dbname=postgres sslmode=disable", port)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestAdjustBalance_LocalPostgres(t *testing.T) {
	db := isolatedLocalPostgres(t)
	ctx := context.Background()
	// Minimal schema for this repository method; no application migrations or
	// existing databases are touched. NUMERIC matches the production money columns.
	_, err := db.ExecContext(ctx, `CREATE TABLE users (
		id BIGINT PRIMARY KEY, balance NUMERIC(20,8) NOT NULL,
		total_recharged NUMERIC(20,8) NOT NULL DEFAULT 30,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ)`)
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := newUserRepositoryWithSQL(client, db)
	reset := func(balance float64) {
		t.Helper()
		_, err := db.ExecContext(ctx, `INSERT INTO users (id,balance) VALUES (1,$1)
			ON CONFLICT (id) DO UPDATE SET balance=$1, total_recharged=30, deleted_at=NULL`, balance)
		require.NoError(t, err)
	}
	check := func(want float64) {
		t.Helper()
		var balance, recharged float64
		require.NoError(t, db.QueryRowContext(ctx, `SELECT balance,total_recharged FROM users WHERE id=1`).Scan(&balance, &recharged))
		require.Equal(t, want, balance)
		require.Equal(t, 30.0, recharged, "generic adjustment must not classify admin/free credit as recharge")
	}
	for _, tc := range []struct {
		name             string
		old, delta, want float64
		rejected         bool
	}{
		{"credit_reduces_debt", -10, 5, -5, false},
		{"credit_clears_debt", -10, 10, 0, false},
		{"credit_exceeds_debt", -10, 15, 5, false},
		{"admin_or_free_credit", 10, 5, 15, false},
		{"zero_preserves_debt", -10, 0, -10, false},
		{"overdraw_rejected", 3, -4, 3, true},
		{"debit_from_debt_rejected", -10, -1, -10, true},
		{"debit_to_zero", 3, -3, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset(tc.old)
			change, err := repo.AdjustBalance(ctx, 1, tc.delta)
			if tc.rejected {
				require.ErrorIs(t, err, service.ErrBalanceNegative)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, service.BalanceChange{Old: tc.old, New: tc.old + tc.delta}, change)
			check(tc.want)
		})
	}
	t.Run("missing_or_deleted_user", func(t *testing.T) {
		_, err := repo.AdjustBalance(ctx, 999, 5)
		require.ErrorIs(t, err, service.ErrUserNotFound)
		reset(-10)
		_, err = db.ExecContext(ctx, `UPDATE users SET deleted_at=NOW() WHERE id=1`)
		require.NoError(t, err)
		_, err = repo.AdjustBalance(ctx, 1, 5)
		require.ErrorIs(t, err, service.ErrUserNotFound)
		check(-10)
	})
	t.Run("outer_transaction_rollback", func(t *testing.T) {
		reset(-10)
		tx, err := client.Tx(ctx)
		require.NoError(t, err)
		defer tx.Rollback()
		change, err := repo.AdjustBalance(dbent.NewTxContext(ctx, tx), 1, 5)
		require.NoError(t, err)
		require.Equal(t, service.BalanceChange{Old: -10, New: -5}, change)
		require.NoError(t, tx.Rollback())
		check(-10)
	})
	t.Run("concurrent_debits_keep_atomic_guard", func(t *testing.T) {
		reset(5)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := repo.AdjustBalance(ctx, 1, -4)
				results <- err
			}()
		}
		wg.Wait()
		close(results)
		accepted := 0
		for err := range results {
			if err == nil {
				accepted++
			} else {
				require.ErrorIs(t, err, service.ErrBalanceNegative)
			}
		}
		require.Equal(t, 1, accepted)
		check(1)
	})
}
