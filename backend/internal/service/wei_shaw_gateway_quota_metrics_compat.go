package service

import "sync/atomic"

var (
	userPlatformQuotaDBIncrErrorTotal           atomic.Int64
	userPlatformQuotaDBIncrLegacyErrorTotal     atomic.Int64
	userPlatformQuotaSentinelSetCacheErrorTotal atomic.Int64
)

func GatewayUserPlatformQuotaIncrStats() (mainPathErr, legacyPathErr, sentinelSetErr int64) {
	return userPlatformQuotaDBIncrErrorTotal.Load(),
		userPlatformQuotaDBIncrLegacyErrorTotal.Load(),
		userPlatformQuotaSentinelSetCacheErrorTotal.Load()
}
