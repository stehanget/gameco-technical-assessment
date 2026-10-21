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
