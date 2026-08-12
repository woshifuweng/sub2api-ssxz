package service

func DefaultRateLimit429CooldownSettings() *RateLimit429CooldownSettings {
	return &RateLimit429CooldownSettings{Enabled: true, CooldownSeconds: 5}
}

func DefaultOpenAIFastPolicySettings() *OpenAIFastPolicySettings {
	return &OpenAIFastPolicySettings{Rules: []OpenAIFastPolicyRule{}}
}
