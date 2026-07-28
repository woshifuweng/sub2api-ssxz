package service

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

const redeemCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

const redeemCodeLength = 16

type RedeemCode struct {
	ID        int64
	Code      string
	Type      string
	Value     float64
	Status    string
	UsedBy    *int64
	UsedAt    *time.Time
	Notes     string
	CreatedAt time.Time
	ExpiresAt *time.Time

	GroupID      *int64
	ValidityDays int

	User  *User
	Group *Group
}

func (r *RedeemCode) IsUsed() bool {
	return r.Status == StatusUsed
}

func (r *RedeemCode) IsExpired() bool {
	return r.IsExpiredAt(time.Now())
}

func (r *RedeemCode) IsExpiredAt(now time.Time) bool {
	if r == nil {
		return false
	}
	if r.Status == StatusExpired {
		return true
	}
	return r.Status == StatusUnused && r.ExpiresAt != nil && !r.ExpiresAt.After(now)
}

func (r *RedeemCode) CanUse() bool {
	return r.Status == StatusUnused && !r.IsExpired()
}

func GenerateRedeemCode() (string, error) {
	code := make([]byte, redeemCodeLength)
	max := big.NewInt(int64(len(redeemCodeAlphabet)))
	for i := range code {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = redeemCodeAlphabet[index.Int64()]
	}
	return string(code), nil
}

// BalanceCreditAmount applies the fixed promotional rule for balance codes.
func BalanceCreditAmount(amount float64) float64 {
	if math.Abs(amount-100) < 1e-9 {
		return 110
	}
	return amount
}
