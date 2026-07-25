package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"

func (c *Channel) IsBedrockCCCompatEnabled(_ string) bool {
	if c == nil || c.FeaturesConfig == nil {
		return false
	}
	enabled, ok := c.FeaturesConfig[featureKeyBedrockCCCompat].(bool)
	return ok && enabled
}

func (s *GatewayService) resolveTLSFingerprintProfile(account *Account) *tlsfingerprint.Profile {
	if s == nil || s.tlsFPProfileService == nil || account == nil {
		return nil
	}
	return s.tlsFPProfileService.ResolveTLSProfile(account)
}
