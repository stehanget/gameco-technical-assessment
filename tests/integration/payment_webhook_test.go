//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	checkoutmodule "gameco-technical-assessment/internal/modules/checkout"
	testsupport "gameco-technical-assessment/tests/support"
)

func TestPaymentWebhookConcurrencyAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	db := testsupport.OpenPostgres(t, cfg)
	server := testsupport.NewAPI(t, cfg)
	productID := integrationProductID(t, server.URL)
	buyer := registerBuyer(t, server.URL, "webhook-concurrency")
	addToCart(t, server.URL, buyer, productID)

	checkout := checkoutRequest(t, server.URL+"/api/v1/checkout", buyer, "webhook-concurrency-checkout")
	if checkout.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", checkout.StatusCode, checkout.Body)
	}
	var order checkoutEnvelope
	decode(t, checkout.Body, &order)

	occurredAt := time.Now().UTC().Truncate(time.Second)
	body := paymentWebhookBody(t, "provider-concurrent-paid", "payment.paid", order.Data.PaymentReference, order.Data.TotalAmount, occurredAt)
	results := runConcurrentWebhooks(server.URL, cfg.App.PaymentWebhookSecret, body)
	for _, result := range results {
		if result.StatusCode != http.StatusNoContent {
			t.Fatalf("concurrent webhook: expected 204, got %d: %s", result.StatusCode, result.Body)
		}
	}
	if got := countRows(t, db, "payment_webhook_events"); got != 1 {
		t.Fatalf("expected one persisted webhook event, got %d", got)
	}
	var paymentStatus, orderStatus string
	db.Raw("SELECT status FROM payments WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&paymentStatus)
	db.Raw("SELECT status FROM orders WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&orderStatus)
	if paymentStatus != "paid" || orderStatus != "paid" {
		t.Fatalf("unexpected concurrent final state: payment=%s order=%s", paymentStatus, orderStatus)
	}
}

func TestPaymentWebhookRollbackAndPayloadConflictAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	db := testsupport.OpenPostgres(t, cfg)
	server := testsupport.NewAPI(t, cfg)
	productID := integrationProductID(t, server.URL)
	buyer := registerBuyer(t, server.URL, "webhook-rollback")
	addToCart(t, server.URL, buyer, productID)

	checkout := checkoutRequest(t, server.URL+"/api/v1/checkout", buyer, "webhook-rollback-checkout")
	if checkout.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", checkout.StatusCode, checkout.Body)
	}
	var order checkoutEnvelope
	decode(t, checkout.Body, &order)
	occurredAt := time.Now().UTC().Truncate(time.Second)
	eventsBefore := countRows(t, db, "payment_webhook_events")

	mismatch := paymentWebhookBody(t, "provider-rollback-1", "payment.paid", order.Data.PaymentReference, order.Data.TotalAmount+1, occurredAt)
	if response := sendPaymentWebhook(t, server.URL, cfg.App.PaymentWebhookSecret, mismatch); response.StatusCode != http.StatusConflict {
		t.Fatalf("amount mismatch: expected 409, got %d: %s", response.StatusCode, response.Body)
	}
	if got := countRows(t, db, "payment_webhook_events"); got != eventsBefore {
		t.Fatalf("failed webhook claim was not rolled back: before=%d after=%d", eventsBefore, got)
	}

	valid := paymentWebhookBody(t, "provider-rollback-1", "payment.paid", order.Data.PaymentReference, order.Data.TotalAmount, occurredAt)
	if response := sendPaymentWebhook(t, server.URL, cfg.App.PaymentWebhookSecret, valid); response.StatusCode != http.StatusNoContent {
		t.Fatalf("valid retry after rollback: expected 204, got %d: %s", response.StatusCode, response.Body)
	}
	if got := countRows(t, db, "payment_webhook_events"); got != eventsBefore+1 {
		t.Fatalf("expected one new event after valid retry: before=%d after=%d", eventsBefore, got)
	}

	conflict := paymentWebhookBody(t, "provider-rollback-1", "payment.failed", order.Data.PaymentReference, order.Data.TotalAmount, occurredAt)
	if response := sendPaymentWebhook(t, server.URL, cfg.App.PaymentWebhookSecret, conflict); response.StatusCode != http.StatusConflict {
		t.Fatalf("payload conflict: expected 409, got %d: %s", response.StatusCode, response.Body)
	}
	if got := countRows(t, db, "payment_webhook_events"); got != eventsBefore+1 {
		t.Fatalf("payload conflict created another event: before=%d after=%d", eventsBefore, got)
	}
}

func TestPaymentExpiredRestoresStockAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	db := testsupport.OpenPostgres(t, cfg)
	server := testsupport.NewAPI(t, cfg)
	productID := integrationProductID(t, server.URL)
	initialStock := productStock(t, db, productID)
	buyer := registerBuyer(t, server.URL, "webhook-expired")
	addToCart(t, server.URL, buyer, productID)

	checkout := checkoutRequest(t, server.URL+"/api/v1/checkout", buyer, "webhook-expired-checkout")
	if checkout.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", checkout.StatusCode, checkout.Body)
	}
	var order checkoutEnvelope
	decode(t, checkout.Body, &order)
	if got := productStock(t, db, productID); got != initialStock-1 {
		t.Fatalf("stock after checkout: expected %d, got %d", initialStock-1, got)
	}

	payload := paymentWebhookBody(t, "provider-expired-1", "payment.expired", order.Data.PaymentReference, order.Data.TotalAmount, time.Now().UTC().Truncate(time.Second))
	if response := sendPaymentWebhook(t, server.URL, cfg.App.PaymentWebhookSecret, payload); response.StatusCode != http.StatusNoContent {
		t.Fatalf("expired webhook: expected 204, got %d: %s", response.StatusCode, response.Body)
	}
	if got := productStock(t, db, productID); got != initialStock {
		t.Fatalf("stock was not restored: expected %d, got %d", initialStock, got)
	}
	var paymentStatus, orderStatus string
	db.Raw("SELECT status FROM payments WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&paymentStatus)
	db.Raw("SELECT status FROM orders WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&orderStatus)
	if paymentStatus != "expired" || orderStatus != "cancelled" {
		t.Fatalf("unexpected expired states: payment=%s order=%s", paymentStatus, orderStatus)
	}

	movementsBefore := countRows(t, db, "stock_movements")
	if response := sendPaymentWebhook(t, server.URL, cfg.App.PaymentWebhookSecret, payload); response.StatusCode != http.StatusNoContent {
		t.Fatalf("expired replay: expected 204, got %d: %s", response.StatusCode, response.Body)
	}
	if got := countRows(t, db, "stock_movements"); got != movementsBefore {
		t.Fatalf("expired replay created another stock movement: before=%d after=%d", movementsBefore, got)
	}
}

func TestPaymentWebhookRetentionCleanupAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	db := testsupport.OpenPostgres(t, cfg)
	repository := checkoutmodule.NewRepository(db)
	now := time.Now().UTC()
	eventsBefore := countRows(t, db, "payment_webhook_events")
	if err := db.Exec(`INSERT INTO payment_webhook_events (provider, event_id, payment_reference, event_type, payload_hash, processed_at) VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)`,
		"simulated", "retention-old", "pay_retention_old", "payment.paid", "old-hash", now.Add(-48*time.Hour),
		"simulated", "retention-new", "pay_retention_new", "payment.paid", "new-hash", now.Add(-time.Hour)).Error; err != nil {
		t.Fatalf("insert retention fixtures: %v", err)
	}
	deleted, err := repository.CleanupProcessedWebhookEvents(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("cleanup webhook events: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one expired webhook event deleted, got %d", deleted)
	}
	if got := countRows(t, db, "payment_webhook_events"); got != eventsBefore+1 {
		t.Fatalf("expected one retained webhook event: before=%d after=%d", eventsBefore, got)
	}
}

type webhookResponse struct {
	StatusCode int
	Body       string
}

func runConcurrentWebhooks(baseURL, secret string, body []byte) []webhookResponse {
	results := make([]webhookResponse, 2)
	var waitGroup sync.WaitGroup
	for index := range results {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			results[index] = sendPaymentWebhookRequest(baseURL, secret, body)
		}(index)
	}
	waitGroup.Wait()
	return results
}

func sendPaymentWebhook(t *testing.T, baseURL, secret string, body []byte) webhookResponse {
	t.Helper()
	result := sendPaymentWebhookRequest(baseURL, secret, body)
	return result
}

func sendPaymentWebhookRequest(baseURL, secret string, body []byte) webhookResponse {
	timestamp := time.Now().Unix()
	timestampHeader := formatUnixTimestamp(timestamp)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(append([]byte(timestampHeader+"."), body...))
	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/webhooks/payment", bytes.NewReader(body))
	if err != nil {
		return webhookResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Payment-Timestamp", timestampHeader)
	request.Header.Set("X-Payment-Signature", hex.EncodeToString(mac.Sum(nil)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return webhookResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return webhookResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	return webhookResponse{StatusCode: response.StatusCode, Body: string(responseBody)}
}

func paymentWebhookBody(t *testing.T, eventID, eventType, paymentReference string, amount int64, occurredAt time.Time) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"event_id":          eventID,
		"event_type":        eventType,
		"payment_reference": paymentReference,
		"amount":            amount,
		"occurred_at":       occurredAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal webhook body: %v", err)
	}
	return body
}

func formatUnixTimestamp(timestamp int64) string {
	return strconv.FormatInt(timestamp, 10)
}
