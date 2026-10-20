package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

func TestLoggingWritesStructuredRequestFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	router := gin.New()
	router.Use(RequestID(), Logging(logger))
	router.GET("/resources/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/resources/42", nil)
	router.ServeHTTP(record, request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("parse structured log: %v", err)
	}
	if entry["msg"] != "http request" {
		t.Fatalf("expected request log message, got %#v", entry["msg"])
	}
	if entry["route"] != "/resources/:id" {
		t.Fatalf("expected route template, got %#v", entry["route"])
	}
	if entry["status"] != float64(http.StatusNoContent) {
		t.Fatalf("expected status 204, got %#v", entry["status"])
	}
	if _, ok := entry["duration_ms"].(float64); !ok {
		t.Fatalf("expected numeric duration_ms, got %#v", entry["duration_ms"])
	}
	if entry["level"] != "INFO" {
		t.Fatalf("expected INFO level, got %#v", entry["level"])
	}
	if entry["request_id"] == "" {
		t.Fatal("expected request_id")
	}
}

func TestLoggingClassifiesTypedErrorsWithoutLeakingCause(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	router := gin.New()
	router.Use(RequestID(), Logging(logger))
	router.GET("/error", func(c *gin.Context) {
		httpx.HandleError(c, faults.New(
			faults.KindConflict,
			errorconst.AuthEmailAlreadyRegistered,
			errors.New("duplicate email password=secret"),
		))
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/error", nil)
	router.ServeHTTP(record, request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("parse structured log: %v", err)
	}
	if entry["level"] != "WARN" {
		t.Fatalf("expected WARN level, got %#v", entry["level"])
	}
	if entry["error_code"] != errorconst.AuthEmailAlreadyRegistered {
		t.Fatalf("expected error_code, got %#v", entry["error_code"])
	}
	if entry["error_kind"] != string(faults.KindConflict) {
		t.Fatalf("expected error_kind, got %#v", entry["error_kind"])
	}
	if _, leaked := entry["error"]; leaked {
		t.Fatal("raw error field must not be logged")
	}
	if bytes.Contains(output.Bytes(), []byte("password=secret")) {
		t.Fatal("error cause leaked into structured log")
	}
}

func TestLoggingIncludesRedactedInternalErrorDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	router := gin.New()
	router.Use(RequestID(), Logging(logger))
	router.GET("/error", func(c *gin.Context) {
		httpx.HandleError(c, errors.New("database password=secret connection failed"))
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/error", nil)
	router.ServeHTTP(record, request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("parse structured log: %v", err)
	}
	if entry["msg"] != "http request failed" {
		t.Fatalf("expected failed request log message, got %#v", entry["msg"])
	}
	if entry["level"] != "ERROR" {
		t.Fatalf("expected ERROR level, got %#v", entry["level"])
	}
	if entry["error"] != "database password=[REDACTED] connection failed" {
		t.Fatalf("expected redacted diagnostic error, got %#v", entry["error"])
	}
	if bytes.Contains(output.Bytes(), []byte("password=secret")) {
		t.Fatal("sensitive error detail leaked into structured log")
	}
}

func TestRequestIDRejectsInvalidIncomingValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/request-id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	request.Header.Set("X-Request-ID", "not-a-uuid")
	router.ServeHTTP(record, request)

	requestID := record.Header().Get("X-Request-ID")
	if requestID == "not-a-uuid" || !validRequestID(requestID) {
		t.Fatalf("expected generated UUID request ID, got %q", requestID)
	}
}
