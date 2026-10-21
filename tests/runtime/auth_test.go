//go:build runtime

package runtime_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRegisterThenMe(t *testing.T) {
	baseURL := runtimeBaseURL(t)
	requestBody, err := json.Marshal(map[string]string{
		"email":    runtimeEmail(),
		"password": "correct-horse-battery-staple",
	})
	if err != nil {
		t.Fatalf("encode register request: %v", err)
	}

	status, body := requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(requestBody), "")
	if status != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", status, body)
	}

	var registerResponse struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &registerResponse)
	if registerResponse.Data.AccessToken == "" {
		t.Fatalf("register: expected access token")
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/auth/me", nil, registerResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("me: expected 200, got %d: %s", status, body)
	}
}

func TestRefreshAndLogout(t *testing.T) {
	baseURL := runtimeBaseURL(t)
	registerBody, err := json.Marshal(map[string]string{
		"email":    runtimeEmail(),
		"password": "correct-horse-battery-staple",
	})
	if err != nil {
		t.Fatalf("encode register request: %v", err)
	}
	status, body := requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(registerBody), "")
	if status != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", status, body)
	}
	var registerResponse struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &registerResponse)
	if registerResponse.Data.RefreshToken == "" {
		t.Fatal("register: expected refresh token")
	}

	refreshBody, err := json.Marshal(map[string]string{"refresh_token": registerResponse.Data.RefreshToken})
	if err != nil {
		t.Fatalf("encode refresh request: %v", err)
	}
	status, body = requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/refresh", bytes.NewReader(refreshBody), "")
	if status != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", status, body)
	}
	var refreshResponse struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &refreshResponse)
	if refreshResponse.Data.RefreshToken == "" {
		t.Fatal("refresh: expected rotated refresh token")
	}

	logoutBody, err := json.Marshal(map[string]string{"refresh_token": refreshResponse.Data.RefreshToken})
	if err != nil {
		t.Fatalf("encode logout request: %v", err)
	}
	status, body = requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/logout", bytes.NewReader(logoutBody), "")
	if status != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", status, body)
	}

	status, body = requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/refresh", bytes.NewReader(logoutBody), "")
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked refresh: expected 401, got %d: %s", status, body)
	}
}
