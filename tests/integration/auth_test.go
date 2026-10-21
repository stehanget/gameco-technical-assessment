//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestAuthHTTPAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)

	email := fmt.Sprintf("integration-%d@example.com", time.Now().UnixNano())
	register := postJSON(t, server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "correct-horse-battery-staple",
	})
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", register.StatusCode, register.Body)
	}

	var token tokenResponse
	decode(t, register.Body, &token)
	if token.Data.AccessToken == "" || token.Data.RefreshToken == "" {
		t.Fatalf("register: expected access and refresh tokens, got %+v", token)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("create me request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token.Data.AccessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request me: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("me: expected 200, got %d", response.StatusCode)
	}
}

type httpResponse struct {
	StatusCode int
	Body       string
}

func postJSON(t *testing.T, url string, payload any) httpResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post JSON: %v", err)
	}
	defer response.Body.Close()
	var responseBody struct {
		Success bool      `json:"success"`
		Data    tokenData `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	encoded, err := json.Marshal(responseBody)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
	return httpResponse{StatusCode: response.StatusCode, Body: string(encoded)}
}

type tokenResponse struct {
	Data tokenData `json:"data"`
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func decode(t *testing.T, body string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}
}
