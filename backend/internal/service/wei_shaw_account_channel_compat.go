package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"

func (s *GatewayService) resolveTLSFingerprintProfile(account *Account) *tlsfingerprint.Profile {
	if s == nil || s.tlsFPProfileService == nil || account == nil {
		return nil
	}
	return s.tlsFPProfileService.ResolveTLSProfile(account)
}
