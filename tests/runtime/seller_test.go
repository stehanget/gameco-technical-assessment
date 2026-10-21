//go:build runtime

package runtime_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSellerProductFlow(t *testing.T) {
	baseURL := runtimeBaseURL(t)

	loginBody, err := json.Marshal(map[string]string{
		"email":    "user@mail.com",
		"password": "admin123",
	})
	if err != nil {
		t.Fatalf("encode login request: %v", err)
	}
	status, body := requestJSON(t, http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(loginBody), "")
	if status != http.StatusOK {
		t.Fatalf("seller login: expected 200, got %d: %s", status, body)
	}
	var loginResponse struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &loginResponse)
	if loginResponse.Data.AccessToken == "" {
		t.Fatal("seller login: expected access token")
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/categories/options", nil, "")
	if status != http.StatusOK {
		t.Fatalf("category options: expected 200, got %d: %s", status, body)
	}
	var categoriesResponse struct {
		Data []struct {
			PublicID uuid.UUID `json:"public_id"`
		} `json:"data"`
	}
	decodeJSON(t, body, &categoriesResponse)
	if len(categoriesResponse.Data) == 0 || categoriesResponse.Data[0].PublicID == uuid.Nil {
		t.Fatalf("category options: expected seeded category, got %s", body)
	}

	name := fmt.Sprintf("Runtime Integration Product %d", time.Now().UnixNano())
	productBody, err := json.Marshal(map[string]any{
		"name":               name,
		"description":        "Created through the runtime binary test",
		"price_amount":       2999,
		"stock_quantity":     5,
		"category_public_id": categoriesResponse.Data[0].PublicID,
	})
	if err != nil {
		t.Fatalf("encode product request: %v", err)
	}
	status, body = requestJSON(t, http.MethodPost, baseURL+"/api/v1/seller/products", bytes.NewReader(productBody), loginResponse.Data.AccessToken)
	if status != http.StatusCreated {
		t.Fatalf("seller create: expected 201, got %d: %s", status, body)
	}
	var createResponse struct {
		Data struct {
			PublicID uuid.UUID `json:"public_id"`
			Name     string    `json:"name"`
		} `json:"data"`
	}
	decodeJSON(t, body, &createResponse)
	if createResponse.Data.PublicID == uuid.Nil || createResponse.Data.Name != name {
		t.Fatalf("seller create: unexpected response %s", body)
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/seller/products/"+createResponse.Data.PublicID.String(), nil, loginResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("seller detail: expected 200, got %d: %s", status, body)
	}
}
