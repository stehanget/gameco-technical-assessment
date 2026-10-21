//go:build runtime

package runtime_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestPaymentWebhookRuntimeFlow(t *testing.T) {
	baseURL := runtimeBaseURL(t)
	secret := os.Getenv("APP_PAYMENT_WEBHOOK_SECRET")
	if secret == "" {
		t.Fatal("APP_PAYMENT_WEBHOOK_SECRET must be set")
	}

	registerBody, err := json.Marshal(map[string]string{
		"email":    runtimeEmail(),
		"password": "correct-horse-battery-staple",
	})
	if err != nil {
		t.Fatalf("encode register request: %v", err)
	}
	status, body := requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(registerBody), "")
	if status != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", status, body)
	}
	var registerResponse struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &registerResponse)

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/products?per_page=1", nil, "")
	if status != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", status, body)
	}
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	decodeJSON(t, body, &products)
	if len(products.Data) != 1 {
		t.Fatalf("products: expected one product, got %+v", products.Data)
	}

	status, body = requestJSON(t, http.MethodPut, baseURL+"/api/v1/cart/items/"+products.Data[0].PublicID, bytes.NewBufferString(`{"quantity":1}`), registerResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("cart: expected 200, got %d: %s", status, body)
	}
	status, body = requestJSONWithHeaders(t, http.MethodPost, baseURL+"/api/v1/checkout", bytes.NewBuffer(nil), registerResponse.Data.AccessToken, map[string]string{"Idempotency-Key": "runtime-webhook-checkout"})
	if status != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", status, body)
	}
	var checkout struct {
		Data struct {
			PublicID         string `json:"public_id"`
			PaymentReference string `json:"payment_reference"`
			TotalAmount      int64  `json:"total_amount"`
		} `json:"data"`
	}
	decodeJSON(t, body, &checkout)

	occurredAt := time.Now().UTC().Truncate(time.Second)
	payload, err := json.Marshal(map[string]any{
		"event_id":          "runtime-payment-paid-1",
		"event_type":        "payment.paid",
		"payment_reference": checkout.Data.PaymentReference,
		"amount":            checkout.Data.TotalAmount,
		"occurred_at":       occurredAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("encode webhook payload: %v", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(append([]byte(timestamp+"."), payload...))
	status, body = requestJSONWithHeaders(t, http.MethodPost, baseURL+"/api/v1/webhooks/payment", bytes.NewReader(payload), "", map[string]string{
		"X-Payment-Timestamp": timestamp,
		"X-Payment-Signature": hex.EncodeToString(mac.Sum(nil)),
	})
	if status != http.StatusNoContent {
		t.Fatalf("payment webhook: expected 204, got %d: %s", status, body)
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/orders/"+checkout.Data.PublicID, nil, registerResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("paid order detail: expected 200, got %d: %s", status, body)
	}
	var detail struct {
		Data struct {
			Status        string `json:"status"`
			PaymentStatus string `json:"payment_status"`
		} `json:"data"`
	}
	decodeJSON(t, body, &detail)
	if detail.Data.Status != "paid" || detail.Data.PaymentStatus != "paid" {
		t.Fatalf("unexpected paid order state: %+v", detail.Data)
	}
}
