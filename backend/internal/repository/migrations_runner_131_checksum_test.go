package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMigrationChecksumCompatible_131AffiliateRebateHardening(t *testing.T) {
	ok := isMigrationChecksumCompatible(
		"131_affiliate_rebate_hardening.sql",
		"706c8102d9600a10f2e2a23156a8cd8b414a241591fd65ab3e26425b2a54fe29",
		"ddafd571d927b02b1288f98928641c19c1c91645cb9c9f2b1ed9c7cefb26476",
	)
	require.True(t, ok)
}
