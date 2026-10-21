package checkout

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestVerifyWebhookSignature(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	timestamp := now.Unix()
	body := []byte(`{"event_id":"provider-event-1"}`)
	secret := "test-payment-webhook-secret"

	hash := hmac.New(sha256.New, []byte(secret))
	_, _ = hash.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	_, _ = hash.Write(body)
	signature := hex.EncodeToString(hash.Sum(nil))

	if err := verifyWebhookSignature(secret, fmt.Sprintf("%d", timestamp), signature, body, now); err != nil {
		t.Fatalf("valid signature: %v", err)
	}
	if err := verifyWebhookSignature(secret, fmt.Sprintf("%d", timestamp), signature+"00", body, now); err != errWebhookSignature {
		t.Fatalf("tampered signature: expected signature error, got %v", err)
	}
	staleTimestamp := timestamp - int64(webhookTimestampTolerance.Seconds()) - 1
	if err := verifyWebhookSignature(secret, fmt.Sprintf("%d", staleTimestamp), signature, body, now); err != errWebhookTimestamp {
		t.Fatalf("stale timestamp: expected timestamp error, got %v", err)
	}
}
