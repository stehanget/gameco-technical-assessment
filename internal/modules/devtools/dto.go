package devtools

import "gameco-technical-assessment/internal/platform/httpx"

type APISignatureRequest struct {
	KeyID     string `json:"key_id" binding:"required" example:"partner-dev-01"`
	Method    string `json:"method" binding:"required" example:"POST"`
	Path      string `json:"path" binding:"required" example:"/api/v1/products"`
	Body      string `json:"body" example:"{\"name\":\"Demo product\",\"price\":125000}"`
	Timestamp *int64 `json:"timestamp,omitempty" example:"1792668600"`
	Nonce     string `json:"nonce,omitempty" example:"nonce-dev-000001"`
}

type WebhookSignatureRequest struct {
	Body      string `json:"body" binding:"required" example:"{\"event_id\":\"evt_20261022_000001\",\"event_type\":\"payment.paid\",\"payment_reference\":\"pay_20261022_000001\",\"amount\":125000,\"occurred_at\":\"2026-10-22T11:30:00Z\"}"`
	Timestamp *int64 `json:"timestamp,omitempty" example:"1792668600"`
}

type SignatureResponse struct {
	Type      string            `json:"type" binding:"required"`
	Timestamp int64             `json:"timestamp" binding:"required"`
	Nonce     string            `json:"nonce,omitempty"`
	Body      string            `json:"body" binding:"required"`
	Signature string            `json:"signature" binding:"required"`
	Headers   map[string]string `json:"headers" binding:"required"`
}

type Response struct {
	Success bool              `json:"success" binding:"required"`
	Data    SignatureResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta       `json:"meta" binding:"required"`
}
