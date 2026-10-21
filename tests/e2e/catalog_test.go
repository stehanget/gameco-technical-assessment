package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicCatalogListAndDetail(t *testing.T) {
	router := newE2ERouter(t)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=2&sort_by=name&sort_order=asc", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("catalog list: expected status 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload struct {
		Success bool `json:"success"`
		Data    []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
		Meta struct {
			PerPage    int     `json:"per_page"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode catalog list: %v", err)
	}
	if !listPayload.Success || len(listPayload.Data) != 2 {
		t.Fatalf("catalog list: expected two products, got %+v", listPayload)
	}
	if listPayload.Meta.PerPage != 2 || !listPayload.Meta.HasMore || listPayload.Meta.NextCursor == nil || *listPayload.Meta.NextCursor == "" {
		t.Fatalf("catalog list: expected cursor metadata, got %+v", listPayload.Meta)
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+listPayload.Data[0].PublicID, nil)
	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("catalog detail: expected status 200, got %d: %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detailPayload struct {
		Success bool `json:"success"`
		Data    struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode catalog detail: %v", err)
	}
	if !detailPayload.Success || detailPayload.Data.PublicID != listPayload.Data[0].PublicID {
		t.Fatalf("catalog detail: unexpected response %+v", detailPayload)
	}
}

func TestSellerProductRejectsUnauthenticatedRequest(t *testing.T) {
	router := newE2ERouter(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("seller products without token: expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSellerProductRejectsInvalidCreatePayload(t *testing.T) {
	router := newE2ERouter(t)
	accessToken := e2ELogin(t, router)

	requestBody := bytes.NewBufferString(`{"name":"Invalid Product","description":"invalid price","price_amount":-1,"stock_quantity":1,"category_public_id":"00000000-0000-0000-0000-000000000000"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/seller/products", requestBody)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("seller create invalid payload: expected status 400, got %d: %s", response.Code, response.Body.String())
	}
}

func e2ELogin(t *testing.T, router http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@mail.com","password":"admin123"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Data.AccessToken == "" {
		t.Fatal("login: expected access token")
	}
	return payload.Data.AccessToken
}
