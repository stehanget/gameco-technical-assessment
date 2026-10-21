package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryLogsPanicAndStackWithoutLeakingSensitiveValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = io.Discard
	t.Cleanup(func() { gin.DefaultErrorWriter = previousErrorWriter })
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	router := gin.New()
	router.Use(RequestID(), Recovery(logger))
	router.GET("/panic", func(*gin.Context) {
		panic("database password=secret failed")
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(record, request)

	if record.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", record.Code)
	}
	if !strings.Contains(output.String(), `"msg":"panic recovered"`) {
		t.Fatal("expected panic log entry")
	}
	if !strings.Contains(output.String(), `"stack":"`) {
		t.Fatal("expected panic stack trace")
	}
	if strings.Contains(output.String(), "password=secret") {
		t.Fatal("panic secret leaked into structured log")
	}
	if !strings.Contains(output.String(), "password=[REDACTED]") {
		t.Fatal("expected panic detail redaction")
	}
}
