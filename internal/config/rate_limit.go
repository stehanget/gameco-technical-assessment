package config

import "time"

type RateLimitPolicyConfig struct {
	Limit  int
	Window time.Duration
}

type RateLimitConfig struct {
	Enabled bool
	Default RateLimitPolicyConfig
}

func loadRateLimitConfig() (RateLimitConfig, error) {
	enabled, err := getEnvBool("RATE_LIMIT_ENABLED", true)
	if err != nil {
		return RateLimitConfig{}, err
	}

	defaultPolicy, err := loadRateLimitPolicy("RATE_LIMIT", 120, time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	return RateLimitConfig{
		Enabled: enabled,
		Default: defaultPolicy,
	}, nil
}

func loadRateLimitPolicy(prefix string, defaultLimit int, defaultWindow time.Duration) (RateLimitPolicyConfig, error) {
	limit, err := getEnvInt(prefix+"_REQUESTS", defaultLimit)
	if err != nil {
		return RateLimitPolicyConfig{}, err
	}

	window, err := getEnvDuration(prefix+"_WINDOW", defaultWindow)
	if err != nil {
		return RateLimitPolicyConfig{}, err
	}

	return RateLimitPolicyConfig{Limit: limit, Window: window}, nil
}
