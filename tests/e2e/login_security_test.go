//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginDoesNotEnumerateUnknownEmail(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	login := func(email string) string {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"`+email+`","password":"wrong-password"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("login %s: expected status 401, got %d: %s", email, response.Code, response.Body.String())
		}
		return response.Body.String()
	}
	known := login("user@mail.com")
	unknown := login("not-registered@example.com")
	var knownPayload, unknownPayload struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(known), &knownPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(unknown), &unknownPayload); err != nil {
		t.Fatal(err)
	}
	if knownPayload != unknownPayload {
		t.Fatalf("known and unknown login responses differ: known=%s unknown=%s", known, unknown)
	}
	var auditCount int64
	if err := db.Raw("SELECT COUNT(*) FROM audit_events WHERE event_type = ?", "auth.login_failed").Scan(&auditCount).Error; err != nil {
		t.Fatalf("count failed-login audit events: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("expected two failed-login audit events, got %d", auditCount)
	}
	var metadata string
	if err := db.Raw("SELECT metadata::text FROM audit_events WHERE event_type = ? ORDER BY id DESC LIMIT 1", "auth.login_failed").Scan(&metadata).Error; err != nil {
		t.Fatalf("read failed-login audit metadata: %v", err)
	}
	if strings.Contains(metadata, "not-registered@example.com") || strings.Contains(metadata, "wrong-password") {
		t.Fatalf("failed-login audit leaked sensitive input: %s", metadata)
	}
}

func TestLoginLocksAfterRepeatedFailures(t *testing.T) {
	router := newE2ERouter(t)
	for attempt := 1; attempt <= 5; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@mail.com","password":"wrong-password"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d: expected status 401, got %d: %s", attempt, response.Code, response.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@mail.com","password":"admin123"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("locked login: expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}
