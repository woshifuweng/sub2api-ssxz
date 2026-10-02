//go:build unit

package repository

import (
	"context"
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCreditPaidRecharge_RejectsInvalidAmountBeforeDatabase(t *testing.T) {
	// No client: invalid amounts must be rejected before any SQL or money write.
	repo := &userRepository{}
	for name, amount := range map[string]float64{
		"zero": 0, "negative": -1, "nan": math.NaN(),
		"positive_infinity": math.Inf(1), "negative_infinity": math.Inf(-1),
	} {
		t.Run(name, func(t *testing.T) {
			change, err := repo.CreditPaidRecharge(context.Background(), 1, amount, amount)
			require.ErrorIs(t, err, service.ErrInvalidPaidRechargeAmount)
			require.Equal(t, service.BalanceChange{}, change)
		})
	}
}

func TestCreditPaidRecharge_RejectsInvalidPaidSplitBeforeDatabase(t *testing.T) {
	repo := &userRepository{}
	for name, paid := range map[string]float64{
		"negative": -1, "above_credit": 6, "nan": math.NaN(),
		"positive_infinity": math.Inf(1), "negative_infinity": math.Inf(-1),
	} {
		t.Run(name, func(t *testing.T) {
			change, err := repo.CreditPaidRecharge(context.Background(), 1, 5, paid)
			require.ErrorIs(t, err, service.ErrInvalidPaidRechargeAmount)
			require.Equal(t, service.BalanceChange{}, change)
		})
	}
}
