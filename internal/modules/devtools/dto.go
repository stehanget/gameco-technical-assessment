package devtools

import "gameco-technical-assessment/internal/platform/httpx"

type APISignatureRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	Method    string `json:"method" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Body      string `json:"body"`
	Timestamp *int64 `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

type WebhookSignatureRequest struct {
	Body      string `json:"body" binding:"required"`
	Timestamp *int64 `json:"timestamp,omitempty"`
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
