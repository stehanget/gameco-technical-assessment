package middleware

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	errorconst "gameco-technical-assessment/internal/constants/error"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const rateLimitScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {count, ttl}
`

type RateLimitPolicy struct {
	Name    string
	Limit   int
	Window  time.Duration
	KeyFunc func(*gin.Context) string
}

type RateLimitPolicySet struct {
	Anonymous     RateLimitPolicy
	Authenticated RateLimitPolicy
}

func RateLimit(client redis.UniversalClient, policies RateLimitPolicySet, jwtManager *platformauth.JWTManager) gin.HandlerFunc {
	return rateLimitHandler(client, func(c *gin.Context) RateLimitPolicy {
		if jwtManager != nil {
			if token := bearerToken(c); token != "" {
				claims, err := jwtManager.ParseAccessToken(token)
				if err == nil && claims.UserID != "" {
					policy := policies.Authenticated
					if policy.KeyFunc == nil {
						policy.KeyFunc = func(*gin.Context) string { return "user:" + claims.UserID }
					}
					return policy
				}
			}
		}
		return policies.Anonymous
	})
}

func rateLimitHandler(client redis.UniversalClient, selectPolicy func(*gin.Context) RateLimitPolicy) gin.HandlerFunc {
	return func(c *gin.Context) {
		policy := selectPolicy(c)
		if client == nil || policy.Limit <= 0 || policy.Window <= 0 {
			c.Next()
			return
		}

		identity := clientIP(c)
		if policy.KeyFunc != nil {
			identity = policy.KeyFunc(c)
		}
		if identity == "" {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.Unauthorized, nil))
			c.Abort()
			return
		}
		key := fmt.Sprintf("gameco:rate-limit:%s:%s", policy.Name, identity)
		result, err := client.Eval(
			c.Request.Context(),
			rateLimitScript,
			[]string{key},
			policy.Window.Milliseconds(),
		).Result()
		if err != nil {
			httpx.HandleError(c, faults.New(faults.KindUnavailable, errorconst.ServiceUnavailable, err))
			c.Abort()
			return
		}

		count, ttl, err := parseRateLimitResult(result)
		if err != nil {
			httpx.HandleError(c, faults.New(faults.KindUnavailable, errorconst.ServiceUnavailable, err))
			c.Abort()
			return
		}

		remaining := maxInt64(int64(policy.Limit)-count, 0)
		resetAt := time.Now().Add(time.Duration(maxInt64(ttl, 0)) * time.Millisecond)
		c.Header("X-RateLimit-Limit", strconv.Itoa(policy.Limit))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if count > int64(policy.Limit) {
			retryAfter := (maxInt64(ttl, 0) + 999) / 1000
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			httpx.HandleError(c, faults.New(faults.KindRateLimit, errorconst.RateLimitExceeded, nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func parseRateLimitResult(result interface{}) (int64, int64, error) {
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, 0, fmt.Errorf("unexpected rate limit result")
	}

	count, err := redisInt64(values[0])
	if err != nil {
		return 0, 0, err
	}
	ttl, err := redisInt64(values[1])
	if err != nil {
		return 0, 0, err
	}
	return count, ttl, nil
}

func redisInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis integer type %T", value)
	}
}

func clientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return c.Request.RemoteAddr
}

func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}
