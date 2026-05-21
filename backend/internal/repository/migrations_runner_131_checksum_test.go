package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMigrationChecksumCompatible_131AffiliateRebateHardening(t *testing.T) {
	cases := []struct {
		name         string
		dbChecksum   string
		fileChecksum string
	}{
		{
			name:         "131_affiliate_rebate_hardening.sql",
			dbChecksum:   "706c8102d96d0a10f2e2a23156a8cd8b414a241591fd65ab3e26425b2a54fe29",
			fileChecksum: "ddafd571d1927b02b1288f98928641c19c1c91645cb9c9f2b1ed9c7cefb26476",
		},
		{
			name:         "132_affiliate_custom_settings.sql",
			dbChecksum:   "49eb9990877096cdffa640cac8e1df2982952964166a0f8a02b327012c1e3e64",
			fileChecksum: "5c8e255d345f592daaf3bc985c18afadbe70b9c65d1adcc134b3629ac2078f9a",
		},
		{
			name:         "133_affiliate_rebate_freeze.sql",
			dbChecksum:   "bd6c9733923cf7219b1b758353e315039594180b31fedf897473d479f5ac1c2e",
			fileChecksum: "360fd3b01dd2c4fedcd60db92732c0c38d2095eb216f970fe0df4292f77b660c",
		},
	}

	for _, tc := range cases {
		require.True(t, isMigrationChecksumCompatible(tc.name, tc.dbChecksum, tc.fileChecksum), tc.name)
	}
}
