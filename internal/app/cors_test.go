package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsConfiguredOriginAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS())
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/health", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "authorization,idempotency-key")
	router.ServeHTTP(record, request)

	if record.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status 204, got %d", record.Code)
	}
	if got := record.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allowed origin, got %q", got)
	}
	allowedHeaders := strings.ToLower(record.Header().Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allowedHeaders, "authorization") || !strings.Contains(allowedHeaders, "idempotency-key") {
		t.Fatalf("expected authorization and idempotency-key headers, got %q", allowedHeaders)
	}
	if record.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("expected credentials to be disabled")
	}
}

func TestCORSMiddlewareRejectsUnconfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS())
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://malicious.example")
	router.ServeHTTP(record, request)

	if record.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed origin status 403, got %d", record.Code)
	}
	if got := record.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}
