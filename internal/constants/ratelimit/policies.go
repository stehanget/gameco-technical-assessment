package ratelimit

import (
	"time"

	"gameco-technical-assessment/internal/platform/middleware"
)

const (
	GlobalAnonymousPolicyName     = "global-anonymous"
	GlobalAuthenticatedPolicyName = "global-authenticated"
)

var (
	GlobalAnonymousPolicy = middleware.RateLimitPolicy{
		Name:   GlobalAnonymousPolicyName,
		Limit:  60,
		Window: time.Minute,
	}
	GlobalAuthenticatedPolicy = middleware.RateLimitPolicy{
		Name:   GlobalAuthenticatedPolicyName,
		Limit:  120,
		Window: time.Minute,
	}
)
