//go:build unit

package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoBalance_CallsAffiliateRebate_OnSuccess(t *testing.T) {
	markCompletedCalls := 0
	affiliateRebateCalls := 0

	err := completePaymentFulfillmentWithAffiliateRebate(
		func() error {
			markCompletedCalls++
			return nil
		},
		func() error {
			affiliateRebateCalls++
			return nil
		},
		101,
		"affiliate rebate failed after balance fulfillment",
	)

	require.NoError(t, err)
	require.Equal(t, 1, markCompletedCalls)
	require.Equal(t, 1, affiliateRebateCalls)
}

func TestDoBalance_AffiliateRebateError_DoesNotRollback(t *testing.T) {
	markCompletedCalls := 0
	affiliateRebateCalls := 0

	err := completePaymentFulfillmentWithAffiliateRebate(
		func() error {
			markCompletedCalls++
			return nil
		},
		func() error {
			affiliateRebateCalls++
			return errors.New("affiliate unavailable")
		},
		102,
		"affiliate rebate failed after balance fulfillment",
	)

	require.NoError(t, err)
	require.Equal(t, 1, markCompletedCalls)
	require.Equal(t, 1, affiliateRebateCalls)
}

func TestDoSub_CallsAffiliateRebate_OnSuccess(t *testing.T) {
	markCompletedCalls := 0
	affiliateRebateCalls := 0

	err := completePaymentFulfillmentWithAffiliateRebate(
		func() error {
			markCompletedCalls++
			return nil
		},
		func() error {
			affiliateRebateCalls++
			return nil
		},
		103,
		"affiliate rebate failed after subscription fulfillment",
	)

	require.NoError(t, err)
	require.Equal(t, 1, markCompletedCalls)
	require.Equal(t, 1, affiliateRebateCalls)
}
