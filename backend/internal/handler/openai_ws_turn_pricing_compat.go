package handler

import "time"

type openAIWSTurnPricing struct {
	pricingAt time.Time
}

func (p openAIWSTurnPricing) current() time.Time {
	return p.pricingAt
}

func (p *openAIWSTurnPricing) currentOr(fallback time.Time) time.Time {
	if p == nil || p.pricingAt.IsZero() {
		return fallback
	}
	return p.pricingAt
}

func (p *openAIWSTurnPricing) freeze(at time.Time) {
	if p != nil {
		p.pricingAt = at
	}
}
