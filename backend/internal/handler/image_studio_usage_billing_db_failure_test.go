package handler

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestOpenAIImagesGateway_UpstreamFailureDoesNotPersistUsageOrBilling(t *testing.T) {
	tests := []struct {
		name           string
		upstream       service.HTTPUpstream
		wantStatus     int
		wantBodyNeedle string
	}{
		{
			name: "upstream_4xx",
			upstream: &imageStudioGatewayUpstream{
				status: http.StatusBadRequest,
				body:   `{"error":{"message":"invalid image request"}}`,
			},
			wantStatus:     http.StatusBadGateway,
			wantBodyNeedle: "Upstream request failed",
		},
		{
			name:           "transport_timeout",
			upstream:       &imageStudioGatewayErrorUpstream{err: imageStudioGatewayTimeoutError{}},
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyNeedle: "Service temporarily unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			db := newImageStudioUsageBillingSQLite(t)
			handler, apiKey, user := newImageStudioGatewayHandlerWithSQLUsageBilling(t, db, tt.upstream)

			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader([]byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
			ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
			ctx.Set(string(middleware.ContextKeyUserRole), user.Role)

			handler.ImagesGateway(gatewayctx.FromGin(ctx))

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.wantBodyNeedle)
			require.Zero(t, countImageStudioSQLRows(t, db, "usage_logs"))
			require.Zero(t, countImageStudioSQLRows(t, db, "usage_billing_dedup"))

			var balance float64
			require.NoError(t, db.QueryRowContext(context.Background(), "SELECT balance FROM users WHERE id = ?", user.ID).Scan(&balance))
			require.Equal(t, 10.0, balance)
		})
	}
}

func newImageStudioGatewayHandlerWithSQLUsageBilling(
	t *testing.T,
	db *sql.DB,
	upstream service.HTTPUpstream,
) (*OpenAIGatewayHandler, *service.APIKey, *service.User) {
	t.Helper()

	groupID := int64(301)
	user := &service.User{
		ID:          701,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     10,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:       groupID,
		Name:     "image-test",
		Platform: service.PlatformOpenAI,
		Status:   service.StatusActive,
	}
	apiKey := &service.APIKey{
		ID:      801,
		UserID:  user.ID,
		Key:     "test-image-studio-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}
	account := service.Account{
		ID:          901,
		Name:        "image-upstream",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 0,
		Credentials: map[string]any{
			"api_key":  "test-upstream-key",
			"base_url": "https://api.example.test",
		},
	}

	cfg := &config.Config{}
	userRepo := &imageStudioGatewayUserRepo{user: user}
	billingCacheService := service.NewBillingCacheService(nil, userRepo, nil, nil, cfg)
	gatewayService := service.NewOpenAIGatewayService(
		&imageStudioGatewayAccountRepo{accounts: []service.Account{account}},
		nil,
		&imageStudioSQLUsageRepo{db: db},
		&imageStudioSQLBillingRepo{db: db},
		userRepo,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		service.NewBillingService(cfg, nil),
		nil,
		nil,
		billingCacheService,
		upstream,
		nil,
		nil,
	)
	t.Cleanup(gatewayService.CloseOpenAIWSPool)

	return NewOpenAIGatewayHandler(
		gatewayService,
		service.NewConcurrencyService(nil),
		billingCacheService,
		&service.APIKeyService{},
		nil,
		nil,
		cfg,
	), apiKey, user
}

func newImageStudioUsageBillingSQLite(t *testing.T) *sql.DB {
	t.Helper()

	dsn := "file:" + strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			balance REAL NOT NULL
		);
		CREATE TABLE usage_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL DEFAULT '',
			api_key_id INTEGER NOT NULL DEFAULT 0,
			user_id INTEGER NOT NULL DEFAULT 0,
			account_id INTEGER NOT NULL DEFAULT 0,
			model TEXT NOT NULL DEFAULT '',
			total_cost REAL NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE usage_billing_dedup (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL DEFAULT '',
			api_key_id INTEGER NOT NULL DEFAULT 0,
			request_fingerprint TEXT NOT NULL DEFAULT '',
			balance_cost REAL NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO users (id, balance) VALUES (701, 10);
	`)
	require.NoError(t, err)

	return db
}

func countImageStudioSQLRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	var err error
	switch table {
	case "usage_logs":
		err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM usage_logs").Scan(&count)
	case "usage_billing_dedup":
		err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM usage_billing_dedup").Scan(&count)
	default:
		t.Fatalf("unknown test table %q", table)
	}
	require.NoError(t, err)
	return count
}

type imageStudioSQLUsageRepo struct {
	service.UsageLogRepository

	db *sql.DB
}

func (r *imageStudioSQLUsageRepo) Create(ctx context.Context, log *service.UsageLog) (bool, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_logs (
			request_id, api_key_id, user_id, account_id, model, total_cost
		) VALUES (?, ?, ?, ?, ?, ?)
	`, log.RequestID, log.APIKeyID, log.UserID, log.AccountID, log.Model, log.TotalCost)
	return err == nil, err
}

type imageStudioSQLBillingRepo struct {
	service.UsageBillingRepository

	db *sql.DB
}

func (r *imageStudioSQLBillingRepo) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	cmd.Normalize()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_billing_dedup (
			request_id, api_key_id, request_fingerprint, balance_cost
		) VALUES (?, ?, ?, ?)
	`, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint, cmd.BalanceCost)
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, "UPDATE users SET balance = balance - ? WHERE id = ?", cmd.BalanceCost, cmd.UserID)
	if err != nil {
		return nil, err
	}
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

type imageStudioGatewayErrorUpstream struct {
	err error
}

func (u *imageStudioGatewayErrorUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, u.err
}

func (u *imageStudioGatewayErrorUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ bool) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

type imageStudioGatewayTimeoutError struct{}

func (imageStudioGatewayTimeoutError) Error() string   { return "context deadline exceeded" }
func (imageStudioGatewayTimeoutError) Timeout() bool   { return true }
func (imageStudioGatewayTimeoutError) Temporary() bool { return true }

var _ service.UsageLogRepository = (*imageStudioSQLUsageRepo)(nil)
var _ service.UsageBillingRepository = (*imageStudioSQLBillingRepo)(nil)
var _ service.HTTPUpstream = (*imageStudioGatewayErrorUpstream)(nil)
var _ error = imageStudioGatewayTimeoutError{}
var _ interface {
	Timeout() bool
	Temporary() bool
} = imageStudioGatewayTimeoutError{}
