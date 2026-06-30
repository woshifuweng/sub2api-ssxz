package service

import (
	"context"
	"strconv"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/stretchr/testify/require"
)

func TestNormalizePaymentAdminOperator(t *testing.T) {
	require.Equal(t, "admin", normalizePaymentAdminOperator(""))
	require.Equal(t, "admin", normalizePaymentAdminOperator("   "))
	require.Equal(t, "admin:42", normalizePaymentAdminOperator(" admin:42 "))
}

func TestRefundPlanAuditOperator(t *testing.T) {
	require.Equal(t, "admin", (&RefundPlan{}).auditOperator())
	require.Equal(t, "admin:42", (&RefundPlan{AdminOperator: "admin:42"}).auditOperator())
}

func TestAdminCancelOrderAuditIncludesOperator(t *testing.T) {
	ctx := context.Background()
	client := newPaymentPublicLookupTestClient(t)
	user := createPaymentPublicLookupUser(t, client, "admin-cancel-audit@example.test")
	order := createPaymentPublicLookupOrder(t, client, user, func(builder *dbent.PaymentOrderCreate) {
		builder.SetPaymentType("")
	})
	svc := &PaymentService{entClient: client}

	result, err := svc.AdminCancelOrder(ctx, order.ID, "admin:42")
	require.NoError(t, err)
	require.Equal(t, checkPaidResultCancelled, result)

	log, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("ORDER_CANCELLED"),
		).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "admin:42", log.Operator)
}
