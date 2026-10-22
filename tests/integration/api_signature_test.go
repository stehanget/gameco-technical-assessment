//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestAuthAPISignatureAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	cfg.App.APISignature.Enabled = true
	cfg.App.APISignature.Keys = map[string]string{"integration-test": "integration-api-signing-secret"}
	server := testsupport.NewAPI(t, cfg)

	email := fmt.Sprintf("signed-integration-%d@example.com", time.Now().UnixNano())
	path := "/api/v1/auth/register"
	body, err := json.Marshal(map[string]string{"email": email, "password": "correct-horse-battery-staple"})
	if err != nil {
		t.Fatalf("encode signed body: %v", err)
	}
	timestamp := time.Now().Unix()
	headers := testsupport.APISignatureHeaders(http.MethodPost, path, body, timestamp, "integration-nonce-valid", "integration-test", "integration-api-signing-secret")
	status, responseBody := signedHTTPJSON(t, server.URL+path, http.MethodPost, body, headers)
	if status != http.StatusCreated {
		t.Fatalf("valid signed register: expected 201, got %d: %s", status, responseBody)
	}

	tamperedBody, err := json.Marshal(map[string]string{"email": email, "password": "tampered-password"})
	if err != nil {
		t.Fatalf("encode tampered body: %v", err)
	}
	status, responseBody = signedHTTPJSON(t, server.URL+path, http.MethodPost, tamperedBody, headers)
	if status != http.StatusUnauthorized {
		t.Fatalf("tampered signed register: expected 401, got %d: %s", status, responseBody)
	}

	status, responseBody = signedHTTPJSON(t, server.URL+path, http.MethodPost, body, headers)
	if status != http.StatusUnauthorized {
		t.Fatalf("replayed signed register: expected 401, got %d: %s", status, responseBody)
	}
}

func signedHTTPJSON(t *testing.T, url, method string, body []byte, headers map[string]string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create signed request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send signed request: %v", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read signed response: %v", err)
	}
	return response.StatusCode, string(responseBody)
}
