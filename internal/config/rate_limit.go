package config

import "time"

type RateLimitPolicyConfig struct {
	Limit  int
	Window time.Duration
}

type RateLimitConfig struct {
	Anonymous     RateLimitPolicyConfig
	Authenticated RateLimitPolicyConfig
}

func loadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		// Anonymous traffic is more abuse-prone because it has no account-level identity.
		// Keep the global IP budget conservative; sensitive public routes define tighter policies.
		Anonymous: RateLimitPolicyConfig{
			Limit:  60,
			Window: time.Minute,
		},
		// Authenticated users share one budget across their devices and access tokens.
		// This leaves room for normal SPA/mobile activity without using a session bucket.
		Authenticated: RateLimitPolicyConfig{
			Limit:  300,
			Window: time.Minute,
		},
	}
}
