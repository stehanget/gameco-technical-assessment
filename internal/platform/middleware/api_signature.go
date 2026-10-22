package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gameco-technical-assessment/internal/config"
	errorconst "gameco-technical-assessment/internal/constants/error"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RequireAPISignature verifies an optional-to-use, route-scoped signature for
// trusted server-to-server callers. It must be applied only to routes that
// explicitly require this additional authentication layer.
func RequireAPISignature(cfg config.APISignatureConfig, redisClient *redis.Client) gin.HandlerFunc {
	return RequireAPISignatureWithClock(cfg, redisClient, platformclock.System{})
}

func RequireAPISignatureWithClock(cfg config.APISignatureConfig, redisClient *redis.Client, timeSource platformclock.Clock) gin.HandlerFunc {
	if timeSource == nil {
		timeSource = platformclock.System{}
	}
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		keyID := strings.TrimSpace(c.GetHeader("X-API-Key"))
		timestampValue := strings.TrimSpace(c.GetHeader("X-API-Timestamp"))
		nonce := strings.TrimSpace(c.GetHeader("X-API-Nonce"))
		suppliedSignature := strings.TrimSpace(c.GetHeader("X-API-Signature"))
		secret, knownKey := cfg.Keys[keyID]
		if !knownKey || timestampValue == "" || nonce == "" || suppliedSignature == "" {
			rejectAPISignature(c)
			return
		}

		seconds, err := strconv.ParseInt(timestampValue, 10, 64)
		if err != nil || !timestampWithinWindow(seconds, timeSource.Now(), cfg.TimestampWindow) {
			rejectAPISignature(c)
			return
		}

		body, err := readAndRestoreBody(c.Request, cfg.MaxSignedRequestBodyBytes)
		if err != nil {
			rejectAPISignature(c)
			return
		}

		suppliedSignatureBytes, err := parseAPISignature(suppliedSignature, cfg.Version)
		if err != nil || !validAPISignature(cfg, secret, timestampValue, nonce, c.Request.Method, requestPathWithQuery(c), body, suppliedSignatureBytes) {
			rejectAPISignature(c)
			return
		}

		nonceKey := apiSignatureNonceKey(cfg.NonceKeyPrefix, keyID, nonce)
		accepted, err := redisClient.SetNX(c.Request.Context(), nonceKey, "1", 2*cfg.TimestampWindow).Result()
		if err != nil {
			httpx.HandleError(c, faults.New(errorconst.AuthAPISignatureUnavailable, err))
			c.Abort()
			return
		}
		if !accepted {
			rejectAPISignature(c)
			return
		}

		c.Next()
	}
}

func timestampWithinWindow(seconds int64, now time.Time, window time.Duration) bool {
	timestamp := time.Unix(seconds, 0)
	return !timestamp.Before(now.Add(-window)) && !timestamp.After(now.Add(window))
}

func readAndRestoreBody(request *http.Request, maxBodyBytes int) ([]byte, error) {
	if request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, int64(maxBodyBytes)+1))
	if err != nil {
		return nil, err
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > maxBodyBytes {
		return nil, errors.New("signed request body too large")
	}
	return body, nil
}

func requestPathWithQuery(c *gin.Context) string {
	path := c.Request.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	if c.Request.URL.RawQuery != "" {
		return path + "?" + c.Request.URL.RawQuery
	}
	return path
}

func parseAPISignature(value, version string) ([]byte, error) {
	configuredVersion, encoded, ok := strings.Cut(value, "=")
	if !ok || configuredVersion != version || encoded == "" || strings.Contains(encoded, "=") {
		return nil, errors.New("invalid API signature format")
	}
	decoded, err := hex.DecodeString(encoded)
	if err != nil || len(decoded) != sha256.Size {
		return nil, errors.New("invalid API signature encoding")
	}
	return decoded, nil
}

func validAPISignature(cfg config.APISignatureConfig, secret, timestamp, nonce, method, path string, body, provided []byte) bool {
	bodyHash := sha256.Sum256(body)
	fields := []string{method, path, timestamp, nonce, hex.EncodeToString(bodyHash[:])}
	rotation := parseTimestampRotation(timestamp, cfg.RotationCount)
	rotated := append(append([]string{}, fields[rotation:]...), fields[:rotation]...)
	message := strings.Join(rotated, "|")

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hmac.Equal(provided, mac.Sum(nil))
}

func parseTimestampRotation(timestamp string, rotationCount int64) int {
	seconds, _ := strconv.ParseInt(timestamp, 10, 64)
	return int(seconds % rotationCount)
}

func apiSignatureNonceKey(prefix, keyID, nonce string) string {
	return prefix + keyID + ":" + nonce
}

func rejectAPISignature(c *gin.Context) {
	httpx.HandleError(c, faults.New(errorconst.AuthAPISignatureInvalid, nil))
	c.Abort()
}
