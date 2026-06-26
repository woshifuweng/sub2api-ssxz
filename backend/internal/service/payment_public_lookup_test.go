package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newPaymentPublicLookupTestClient(t *testing.T) *dbent.Client {
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

func createPaymentPublicLookupUser(t *testing.T, client *dbent.Client, email string) *dbent.User {
	t.Helper()

	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed-password").
		Save(context.Background())
	require.NoError(t, err)
	return user
}

func createPaymentPublicLookupOrder(t *testing.T, client *dbent.Client, user *dbent.User, mutate func(*dbent.PaymentOrderCreate)) *dbent.PaymentOrder {
	t.Helper()

	builder := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName("Payment Tester").
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo("sub2_public_lookup").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test")
	if mutate != nil {
		mutate(builder)
	}

	order, err := builder.Save(context.Background())
	require.NoError(t, err)
	return order
}

func TestVerifyOrderPublicReturnsPersistedStateWithoutUpstreamQuery(t *testing.T) {
	ctx := context.Background()
	client := newPaymentPublicLookupTestClient(t)
	user := createPaymentPublicLookupUser(t, client, "public-verify@example.test")
	order := createPaymentPublicLookupOrder(t, client, user, nil)

	provider := &countingPaymentProvider{
		providerKey: payment.TypeAlipay,
		types:       []payment.PaymentType{payment.TypeAlipay},
		queryResp:   &payment.QueryOrderResponse{Status: payment.ProviderStatusPending},
	}
	registry := payment.NewRegistry()
	registry.Register(provider)
	svc := &PaymentService{
		entClient: client,
		registry:  registry,
	}

	publicOrder, err := svc.VerifyOrderPublic(ctx, order.OutTradeNo)
	require.NoError(t, err)
	require.Equal(t, order.ID, publicOrder.ID)
	require.Equal(t, OrderStatusPending, publicOrder.Status)
	require.Equal(t, 0, provider.queryCalls, "anonymous public verify must not query upstream payment providers")

	_, err = svc.VerifyOrderByOutTradeNo(ctx, order.OutTradeNo, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, provider.queryCalls, "authenticated verify remains the upstream reconciliation path")
}

func TestVerifyOrderByOutTradeNoRejectsOtherUsersBeforeUpstreamQuery(t *testing.T) {
	ctx := context.Background()
	client := newPaymentPublicLookupTestClient(t)
	owner := createPaymentPublicLookupUser(t, client, "owner@example.test")
	other := createPaymentPublicLookupUser(t, client, "other@example.test")
	order := createPaymentPublicLookupOrder(t, client, owner, nil)

	provider := &countingPaymentProvider{
		providerKey: payment.TypeAlipay,
		types:       []payment.PaymentType{payment.TypeAlipay},
		queryResp:   &payment.QueryOrderResponse{Status: payment.ProviderStatusPending},
	}
	registry := payment.NewRegistry()
	registry.Register(provider)
	svc := &PaymentService{
		entClient: client,
		registry:  registry,
	}

	_, err := svc.VerifyOrderByOutTradeNo(ctx, order.OutTradeNo, other.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no permission")
	require.Equal(t, 0, provider.queryCalls, "wrong-user verify must fail before upstream lookup")
}

func TestGetPublicOrderByResumeTokenRejectsInvalidSignedTokens(t *testing.T) {
	ctx := context.Background()
	client := newPaymentPublicLookupTestClient(t)
	user := createPaymentPublicLookupUser(t, client, "resume-token@example.test")
	order := createPaymentPublicLookupOrder(t, client, user, func(builder *dbent.PaymentOrderCreate) {
		builder.SetProviderKey(payment.TypeAlipay)
	})
	resume := NewPaymentResumeService([]byte("payment-resume-test-signing-key-32-bytes"))
	svc := &PaymentService{
		entClient:     client,
		resumeService: resume,
	}

	validToken, err := resume.CreateToken(ResumeTokenClaims{
		OrderID:     order.ID,
		UserID:      user.ID,
		ProviderKey: payment.TypeAlipay,
		PaymentType: payment.TypeAlipay,
	})
	require.NoError(t, err)

	got, err := svc.GetPublicOrderByResumeToken(ctx, validToken)
	require.NoError(t, err)
	require.Equal(t, order.ID, got.ID)

	_, err = svc.GetPublicOrderByResumeToken(ctx, validToken+"tampered")
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_RESUME_TOKEN")

	expiredToken, err := resume.CreateToken(ResumeTokenClaims{
		OrderID:   order.ID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	require.NoError(t, err)
	_, err = svc.GetPublicOrderByResumeToken(ctx, expiredToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")

	otherUserToken, err := resume.CreateToken(ResumeTokenClaims{
		OrderID: order.ID,
		UserID:  user.ID + 1,
	})
	require.NoError(t, err)
	_, err = svc.GetPublicOrderByResumeToken(ctx, otherUserToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")

	wrongProviderToken, err := resume.CreateToken(ResumeTokenClaims{
		OrderID:     order.ID,
		UserID:      user.ID,
		ProviderKey: payment.TypeWxpay,
	})
	require.NoError(t, err)
	_, err = svc.GetPublicOrderByResumeToken(ctx, wrongProviderToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")

	wrongPaymentTypeToken, err := resume.CreateToken(ResumeTokenClaims{
		OrderID:     order.ID,
		UserID:      user.ID,
		PaymentType: payment.TypeWxpay,
	})
	require.NoError(t, err)
	_, err = svc.GetPublicOrderByResumeToken(ctx, wrongPaymentTypeToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")
}

type countingPaymentProvider struct {
	providerKey string
	types       []payment.PaymentType
	queryResp   *payment.QueryOrderResponse
	queryCalls  int
}

func (p *countingPaymentProvider) Name() string {
	return "counting payment provider"
}

func (p *countingPaymentProvider) ProviderKey() string {
	return p.providerKey
}

func (p *countingPaymentProvider) SupportedTypes() []payment.PaymentType {
	return p.types
}

func (p *countingPaymentProvider) CreatePayment(context.Context, payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	return nil, fmt.Errorf("unexpected CreatePayment call")
}

func (p *countingPaymentProvider) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	p.queryCalls++
	return p.queryResp, nil
}

func (p *countingPaymentProvider) VerifyNotification(context.Context, string, map[string]string) (*payment.PaymentNotification, error) {
	return nil, fmt.Errorf("unexpected VerifyNotification call")
}

func (p *countingPaymentProvider) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("unexpected Refund call")
}
