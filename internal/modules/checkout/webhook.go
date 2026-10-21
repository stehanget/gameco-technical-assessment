package checkout

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	webhookTimestampTolerance  = 5 * time.Minute
	simulatedPaymentProvider   = "simulated"
	maxPaymentWebhookBodyBytes = 64 << 10
)

type PaymentWebhookRequest struct {
	EventID          string `json:"event_id"`
	EventType        string `json:"event_type"`
	PaymentReference string `json:"payment_reference"`
	Amount           int64  `json:"amount"`
	OccurredAt       string `json:"occurred_at"`
	FailureReason    string `json:"failure_reason,omitempty"`
}

func (p PaymentWebhookRequest) Validate() error {
	if strings.TrimSpace(p.EventID) == "" || len(p.EventID) > 128 ||
		strings.TrimSpace(p.PaymentReference) == "" || len(p.PaymentReference) > 128 ||
		p.Amount < 0 || strings.TrimSpace(p.OccurredAt) == "" {
		return errWebhookMalformedPayload
	}
	if _, err := time.Parse(time.RFC3339, p.OccurredAt); err != nil {
		return errWebhookMalformedPayload
	}
	switch p.EventType {
	case "payment.paid", "payment.failed", "payment.expired":
		return nil
	default:
		return errWebhookInvalidEvent
	}
}

func decodePaymentWebhook(body []byte) (PaymentWebhookRequest, error) {
	var request PaymentWebhookRequest
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&request); err != nil {
		return PaymentWebhookRequest{}, errWebhookMalformedPayload
	}
	if err := request.Validate(); err != nil {
		return PaymentWebhookRequest{}, err
	}
	return request, nil
}

var (
	errWebhookSignature = errors.New("invalid payment webhook signature")
	errWebhookTimestamp = errors.New("invalid payment webhook timestamp")
)

func verifyWebhookSignature(secret, timestamp, suppliedSignature string, body []byte, now time.Time) error {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errWebhookTimestamp
	}
	occurredAt := time.Unix(seconds, 0)
	if occurredAt.Before(now.Add(-webhookTimestampTolerance)) || occurredAt.After(now.Add(webhookTimestampTolerance)) {
		return errWebhookTimestamp
	}

	provided, err := hex.DecodeString(suppliedSignature)
	if err != nil {
		return errWebhookSignature
	}
	message := timestamp + "."
	messageBytes := append([]byte(message), body...)
	hash := sha256.New
	mac := hmac.New(hash, []byte(secret))
	_, _ = mac.Write(messageBytes)
	expected := mac.Sum(nil)
	if len(provided) != len(expected) || subtle.ConstantTimeCompare(provided, expected) != 1 {
		return errWebhookSignature
	}
	return nil
}
