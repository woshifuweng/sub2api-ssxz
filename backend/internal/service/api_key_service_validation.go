package service

func validateAPIKeyCreateLimits(req CreateAPIKeyRequest) error {
	if req.Quota < 0 {
		return ErrAPIKeyQuotaInvalid
	}
	return validateAPIKeyRateLimits(req.RateLimit5h, req.RateLimit1d, req.RateLimit7d)
}

func validateAPIKeyUpdateLimits(req UpdateAPIKeyRequest) error {
	if req.Quota != nil && *req.Quota < 0 {
		return ErrAPIKeyQuotaInvalid
	}
	if req.RateLimit5h != nil && *req.RateLimit5h < 0 {
		return ErrAPIKeyRateLimitInvalid
	}
	if req.RateLimit1d != nil && *req.RateLimit1d < 0 {
		return ErrAPIKeyRateLimitInvalid
	}
	if req.RateLimit7d != nil && *req.RateLimit7d < 0 {
		return ErrAPIKeyRateLimitInvalid
	}
	return nil
}

func validateAPIKeyRateLimits(rateLimit5h, rateLimit1d, rateLimit7d float64) error {
	if rateLimit5h < 0 || rateLimit1d < 0 || rateLimit7d < 0 {
		return ErrAPIKeyRateLimitInvalid
	}
	return nil
}
