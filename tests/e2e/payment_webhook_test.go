//go:build e2e

package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaymentWebhookRejectsMissingAndInvalidSignature(t *testing.T) {
	router := newE2ERouter(t)
	body := []byte(`{"event_id":"provider-event-1","event_type":"payment.paid","payment_reference":"pay_missing","amount":1000,"occurred_at":"2026-10-21T12:00:00Z"}`)

	missingSignature := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payment", bytes.NewReader(body))
	missingSignature.Header.Set("Content-Type", "application/json")
	missingResponse := httptest.NewRecorder()
	router.ServeHTTP(missingResponse, missingSignature)
	if missingResponse.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: expected 401, got %d: %s", missingResponse.Code, missingResponse.Body.String())
	}

	invalidSignature := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payment", bytes.NewReader(body))
	invalidSignature.Header.Set("Content-Type", "application/json")
	invalidSignature.Header.Set("X-Payment-Timestamp", "1792584000")
	invalidSignature.Header.Set("X-Payment-Signature", "invalid")
	invalidResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidResponse, invalidSignature)
	if invalidResponse.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature: expected 401, got %d: %s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func TestPaymentWebhookProcessesAndReplaysPaidEvent(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	token := e2ERegister(t, router)
	productResponse := httptest.NewRecorder()
	router.ServeHTTP(productResponse, httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil))
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productResponse.Body.Bytes(), &products); err != nil || len(products.Data) != 1 {
		t.Fatalf("product response: %v %s", err, productResponse.Body.String())
	}
	cart := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+products.Data[0].PublicID, bytes.NewBufferString(`{"quantity":1}`))
	cart.Header.Set("Authorization", "Bearer "+token)
	cart.Header.Set("Content-Type", "application/json")
	cartResponse := httptest.NewRecorder()
	router.ServeHTTP(cartResponse, cart)
	if cartResponse.Code != http.StatusOK {
		t.Fatalf("cart: %d %s", cartResponse.Code, cartResponse.Body.String())
	}
	checkout := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", nil)
	checkout.Header.Set("Authorization", "Bearer "+token)
	checkout.Header.Set("Idempotency-Key", "webhook-checkout")
	checkoutResponse := httptest.NewRecorder()
	router.ServeHTTP(checkoutResponse, checkout)
	var order struct {
		Data struct {
			PaymentReference string `json:"payment_reference"`
			TotalAmount      int64  `json:"total_amount"`
		} `json:"data"`
	}
	if checkoutResponse.Code != http.StatusCreated || json.Unmarshal(checkoutResponse.Body.Bytes(), &order) != nil {
		t.Fatalf("checkout: %d %s", checkoutResponse.Code, checkoutResponse.Body.String())
	}

	body := []byte(`{"event_id":"provider-event-paid-1","event_type":"payment.paid","payment_reference":"` + order.Data.PaymentReference + `","amount":` + fmt.Sprint(order.Data.TotalAmount) + `,"occurred_at":"2026-10-21T12:00:00Z"}`)
	send := func(payload []byte) *httptest.ResponseRecorder {
		timestamp := "1792584000"
		mac := hmac.New(sha256.New, []byte("test-payment-webhook-secret"))
		_, _ = mac.Write(append([]byte(timestamp+"."), payload...))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payment", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Payment-Timestamp", timestamp)
		req.Header.Set("X-Payment-Signature", hex.EncodeToString(mac.Sum(nil)))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	if response := send(body); response.Code != http.StatusNoContent {
		t.Fatalf("paid webhook: %d %s", response.Code, response.Body.String())
	}
	if response := send(body); response.Code != http.StatusNoContent {
		t.Fatalf("paid replay: %d %s", response.Code, response.Body.String())
	}
	var paymentStatus, orderStatus string
	db.Raw("SELECT status FROM payments WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&paymentStatus)
	db.Raw("SELECT status FROM orders WHERE payment_reference = ?", order.Data.PaymentReference).Scan(&orderStatus)
	if paymentStatus != "paid" || orderStatus != "paid" {
		t.Fatalf("unexpected final states: payment=%s order=%s", paymentStatus, orderStatus)
	}
}
