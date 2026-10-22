package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gameco-technical-assessment/internal/config"
	platformclock "gameco-technical-assessment/internal/platform/clock"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRequireAPISignatureAcceptsAllTimestampRotations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	keyID := "integration-prod-01"
	secret := "integration-secret"
	now := time.Unix(1_792_584_000, 0)
	body := []byte(`{"amount":1000}`)

	testConfig := apiSignatureTestConfig(keyID, secret)
	rotationCount := testConfig.RotationCount
	for rotation := int64(0); rotation < rotationCount; rotation++ {
		t.Run(strconv.FormatInt(rotation, 10), func(t *testing.T) {
			timestamp := now.Unix() - (now.Unix() % rotationCount) + rotation
			nonce := fmt.Sprintf("nonce-%d", rotation)
			router := gin.New()
			router.POST("/api/v1/checkout", RequireAPISignatureWithClock(testConfig, redisClient, platformclock.Fixed{Current: now}), func(c *gin.Context) {
				c.String(200, string(body))
			})

			request := httptest.NewRequest("POST", "/api/v1/checkout?preview=false", strings.NewReader(string(body)))
			request.Header.Set("X-API-Key", keyID)
			request.Header.Set("X-API-Timestamp", strconv.FormatInt(timestamp, 10))
			request.Header.Set("X-API-Nonce", nonce)
			request.Header.Set("X-API-Signature", "v1="+signAPITest(secret, request.Method, request.URL.EscapedPath()+"?"+request.URL.RawQuery, timestamp, nonce, body, rotationCount))

			record := httptest.NewRecorder()
			router.ServeHTTP(record, request)
			if record.Code != 200 {
				t.Fatalf("status = %d, body = %s", record.Code, record.Body.String())
			}
			if record.Body.String() != string(body) {
				t.Fatalf("body = %q, want %q", record.Body.String(), body)
			}
		})
	}
}

func TestRequireAPISignatureRejectsTamperingAndNonceReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	keyID := "integration-prod-01"
	secret := "integration-secret"
	now := time.Unix(1_792_584_000, 0)
	body := []byte(`{"amount":1000}`)

	newRequest := func(signatureBody []byte, signatureNonce string) *httptest.ResponseRecorder {
		timestamp := now.Unix()
		router := gin.New()
		router.POST("/api/v1/checkout", RequireAPISignatureWithClock(apiSignatureTestConfig(keyID, secret), redisClient, platformclock.Fixed{Current: now}), func(c *gin.Context) {
			c.Status(204)
		})
		request := httptest.NewRequest("POST", "/api/v1/checkout", strings.NewReader(string(body)))
		request.Header.Set("X-API-Key", keyID)
		request.Header.Set("X-API-Timestamp", strconv.FormatInt(timestamp, 10))
		request.Header.Set("X-API-Nonce", signatureNonce)
		request.Header.Set("X-API-Signature", "v1="+signAPITest(secret, request.Method, request.URL.EscapedPath(), timestamp, signatureNonce, signatureBody, apiSignatureTestConfig(keyID, secret).RotationCount))
		record := httptest.NewRecorder()
		router.ServeHTTP(record, request)
		return record
	}

	if record := newRequest([]byte(`{"amount":999}`), "nonce-tampered"); record.Code != 401 {
		t.Fatalf("tampered body status = %d, want 401", record.Code)
	}
	if record := newRequest(body, "nonce-replay"); record.Code != 204 {
		t.Fatalf("first request status = %d, want 204", record.Code)
	}
	if record := newRequest(body, "nonce-replay"); record.Code != 401 {
		t.Fatalf("replayed nonce status = %d, want 401", record.Code)
	}
}

func apiSignatureTestConfig(keyID, secret string) config.APISignatureConfig {
	rotationCount := int64(config.DefaultAPISignatureRotationCount)
	if configured := strings.TrimSpace(os.Getenv("APP_API_SIGNATURE_ROTATION_COUNT")); configured != "" {
		parsed, err := strconv.ParseInt(configured, 10, 64)
		if err != nil || parsed < 1 || parsed > config.MaxAPISignatureRotationCount {
			panic("invalid APP_API_SIGNATURE_ROTATION_COUNT in middleware test environment")
		}
		rotationCount = parsed
	}
	return config.APISignatureConfig{
		Enabled:                   true,
		Version:                   "v1",
		RotationCount:             rotationCount,
		TimestampWindow:           5 * time.Minute,
		NonceKeyPrefix:            "api-signature:nonce:",
		MaxSignedRequestBodyBytes: 1 << 20,
		Keys:                      map[string]string{keyID: secret},
	}
}

func signAPITest(secret, method, path string, timestamp int64, nonce string, body []byte, rotationCount int64) string {
	bodyHash := sha256.Sum256(body)
	timestampValue := strconv.FormatInt(timestamp, 10)
	fields := []string{method, path, timestampValue, nonce, hex.EncodeToString(bodyHash[:])}
	rotation := int(timestamp % rotationCount)
	fields = append(append([]string{}, fields[rotation:]...), fields[:rotation]...)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.Join(fields, "|")))
	return hex.EncodeToString(mac.Sum(nil))
}
