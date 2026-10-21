//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
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

	meRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("create me request: %v", err)
	}
	meRequest.Header.Set("Authorization", "Bearer "+token.Data.AccessToken)
	response, err := http.DefaultClient.Do(meRequest)
	if err != nil {
		t.Fatalf("request me: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("me: expected 200, got %d", response.StatusCode)
	}

	refreshed := request(t, http.MethodPost, server.URL+"/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": token.Data.RefreshToken,
	})
	if refreshed.StatusCode != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", refreshed.StatusCode, refreshed.Body)
	}
	var refreshedToken tokenResponse
	decode(t, refreshed.Body, &refreshedToken)
	if refreshedToken.Data.AccessToken == "" || refreshedToken.Data.RefreshToken == "" {
		t.Fatal("refresh: expected rotated token pair")
	}

	logout := request(t, http.MethodPost, server.URL+"/api/v1/auth/logout", "", map[string]string{
		"refresh_token": refreshedToken.Data.RefreshToken,
	})
	if logout.StatusCode != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", logout.StatusCode, logout.Body)
	}

	revoked := request(t, http.MethodPost, server.URL+"/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": refreshedToken.Data.RefreshToken,
	})
	if revoked.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked refresh: expected 401, got %d: %s", revoked.StatusCode, revoked.Body)
	}
}

func TestLoginLocksAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)
	email := fmt.Sprintf("lockout-%d@example.com", time.Now().UnixNano())

	register := postJSON(t, server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "correct-horse-battery-staple",
	})
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", register.StatusCode, register.Body)
	}

	for attempt := 1; attempt <= 5; attempt++ {
		login := postJSON(t, server.URL+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": "wrong-password",
		})
		if login.StatusCode != http.StatusUnauthorized {
			t.Fatalf("failed login %d: expected 401, got %d: %s", attempt, login.StatusCode, login.Body)
		}
	}

	lockedLogin := postJSON(t, server.URL+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "correct-horse-battery-staple",
	})
	if lockedLogin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("locked login: expected 401, got %d: %s", lockedLogin.StatusCode, lockedLogin.Body)
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
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return httpResponse{StatusCode: response.StatusCode, Body: string(responseBody)}
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
