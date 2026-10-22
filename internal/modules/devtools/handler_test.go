package devtools

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gameco-technical-assessment/internal/config"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

func demoHandler() *Handler {
	return NewHandler(RouteDependencies{
		APISignature: config.APISignatureConfig{
			Version: "v1", RotationCount: 4, Keys: map[string]string{"demo": "demo-secret"},
		},
		PaymentWebhookSecret: "webhook-secret",
		Clock:                platformclock.Fixed{Current: time.Unix(1792584000, 0)},
	})
}

func TestAPISignatureGeneratesHeadersWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := demoHandler()
	router := gin.New()
	router.POST("/dev", handler.APISignature)
	wireBody := `{"amount":1000}`
	payload, err := json.Marshal(APISignatureRequest{KeyID: "demo", Method: "POST", Path: "/api/v1/checkout", Body: wireBody, Nonce: "demo-nonce"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/dev", strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	record := httptest.NewRecorder()
	router.ServeHTTP(record, request)
	if record.Code != 200 {
		t.Fatalf("status=%d body=%s", record.Code, record.Body.String())
	}
	if strings.Contains(record.Body.String(), "demo-secret") {
		t.Fatal("response leaked signing secret")
	}
	var response struct {
		Data SignatureResponse `json:"data"`
	}
	if err := json.Unmarshal(record.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Type != "api" || response.Data.Nonce != "demo-nonce" {
		t.Fatalf("unexpected response: %+v", response.Data)
	}
	want, err := middleware.SignAPISignature(handler.apiSignature, "demo", "1792584000", "demo-nonce", "POST", "/api/v1/checkout", []byte(wireBody))
	if err != nil {
		t.Fatal(err)
	}
	if response.Data.Signature != want || response.Data.Headers["X-API-Signature"] != want {
		t.Fatalf("signature mismatch: got=%s want=%s", response.Data.Signature, want)
	}
}

func TestWebhookSignatureGeneratesHeadersWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := demoHandler()
	router := gin.New()
	router.POST("/dev", handler.WebhookSignature)
	payload, err := json.Marshal(WebhookSignatureRequest{Body: `{"event_id":"event-1"}`})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/dev", strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	record := httptest.NewRecorder()
	router.ServeHTTP(record, request)
	if record.Code != 200 {
		t.Fatalf("status=%d body=%s", record.Code, record.Body.String())
	}
	if strings.Contains(record.Body.String(), "webhook-secret") {
		t.Fatal("response leaked webhook secret")
	}
	var response struct {
		Data SignatureResponse `json:"data"`
	}
	if err := json.Unmarshal(record.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Type != "webhook" || response.Data.Signature == "" || response.Data.Headers["X-Payment-Signature"] != response.Data.Signature {
		t.Fatalf("unexpected response: %+v", response.Data)
	}
}
