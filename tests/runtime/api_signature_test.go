//go:build runtime

package runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"

	"github.com/redis/go-redis/v9"
)

func TestAPISignatureRejectsTamperingAndReplay(t *testing.T) {
	baseURL := runtimeBaseURL(t)
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		redisDB = 0
	}
	redisClient := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR"), Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB})
	if err := redisClient.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("reset runtime Redis state: %v", err)
	}
	t.Cleanup(func() {
		_ = redisClient.FlushDB(context.Background()).Err()
		_ = redisClient.Close()
	})
	path := "/api/v1/auth/register"
	body, err := json.Marshal(map[string]string{"email": "runtime-signed@example.com", "password": "correct-horse-battery-staple"})
	if err != nil {
		t.Fatalf("encode signed body: %v", err)
	}
	keyID, secret := runtimeAPISigningCredentials(t)
	headers := testsupport.APISignatureHeaders(http.MethodPost, path, body, time.Now().Unix(), "runtime-fixed-nonce", keyID, secret)

	status, responseBody := requestJSONWithHeaders(t, http.MethodPost, baseURL+path, bytes.NewReader(body), "", headers)
	if status != http.StatusCreated {
		t.Fatalf("valid signed register: expected 201, got %d: %s", status, responseBody)
	}

	tamperedBody, err := json.Marshal(map[string]string{"email": "runtime-signed@example.com", "password": "tampered-password"})
	if err != nil {
		t.Fatalf("encode tampered body: %v", err)
	}
	status, responseBody = requestJSONWithHeaders(t, http.MethodPost, baseURL+path, bytes.NewReader(tamperedBody), "", headers)
	if status != http.StatusUnauthorized {
		t.Fatalf("tampered signed register: expected 401, got %d: %s", status, responseBody)
	}

	status, responseBody = requestJSONWithHeaders(t, http.MethodPost, baseURL+path, bytes.NewReader(body), "", headers)
	if status != http.StatusUnauthorized {
		t.Fatalf("replayed signed register: expected 401, got %d: %s", status, responseBody)
	}
}
