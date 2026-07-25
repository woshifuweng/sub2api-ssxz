package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newUsageServiceCreateTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", name)

	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type usageServiceCreateUsageRepo struct {
	UsageLogRepository
	createCalls int
}

func (r *usageServiceCreateUsageRepo) Create(ctx context.Context, log *UsageLog) (bool, error) {
	r.createCalls++
	log.ID = 123
	return true, nil
}

type usageServiceCreateUserRepo struct {
	UserRepository
	user        *User
	deductErr   error
	deductCalls []float64
	updateCalls []float64
}

func (r *usageServiceCreateUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	return r.user, nil
}

func (r *usageServiceCreateUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	r.updateCalls = append(r.updateCalls, amount)
	return nil
}

func (r *usageServiceCreateUserRepo) DeductBalance(ctx context.Context, id int64, amount float64) error {
	r.deductCalls = append(r.deductCalls, amount)
	return r.deductErr
}

type usageServiceCreateAuthCacheInvalidator struct {
	invalidatedUserIDs []int64
}

func (i *usageServiceCreateAuthCacheInvalidator) InvalidateAuthCacheByKey(ctx context.Context, key string) {
}

func (i *usageServiceCreateAuthCacheInvalidator) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	i.invalidatedUserIDs = append(i.invalidatedUserIDs, userID)
}

func (i *usageServiceCreateAuthCacheInvalidator) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
}

func TestUsageServiceCreate_RejectsBalanceOverdraft(t *testing.T) {
	ctx := context.Background()
	client := newUsageServiceCreateTestClient(t)
	usageRepo := &usageServiceCreateUsageRepo{}
	userRepo := &usageServiceCreateUserRepo{
		user:      &User{ID: 1, Balance: 1},
		deductErr: ErrInsufficientBalance,
	}
	invalidator := &usageServiceCreateAuthCacheInvalidator{}
	svc := NewUsageService(usageRepo, userRepo, client, invalidator)

	log, err := svc.Create(ctx, CreateUsageLogRequest{
		UserID:     1,
		APIKeyID:   2,
		AccountID:  3,
		RequestID:  "usage-service-overdraft",
		Model:      "claude-3",
		TotalCost:  3,
		ActualCost: 3,
	})

	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Nil(t, log)
	require.Equal(t, 1, usageRepo.createCalls)
	require.Equal(t, []float64{3}, userRepo.deductCalls)
	require.Empty(t, userRepo.updateCalls)
	require.Empty(t, invalidator.invalidatedUserIDs)
}
