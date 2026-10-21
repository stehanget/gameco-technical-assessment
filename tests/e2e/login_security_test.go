//go:build e2e

package e2e

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
