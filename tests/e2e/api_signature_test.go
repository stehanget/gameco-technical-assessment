//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestAuthAPISignatureRouteIntegrity(t *testing.T) {
	router, _ := newE2ERouterWithDBSignature(t, true)
	now := time.Now().Unix()
	path := "/api/v1/auth/register"
	body, err := json.Marshal(map[string]string{"email": "signed-e2e@example.com", "password": "correct-horse-battery-staple"})
	if err != nil {
		t.Fatalf("encode signed body: %v", err)
	}

	validRequest := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	validRequest.Header.Set("Content-Type", "application/json")
	for key, value := range testsupport.APISignatureHeaders(http.MethodPost, path, body, now, "e2e-nonce-valid", "e2e-integration", "e2e-api-signing-secret") {
		validRequest.Header.Set(key, value)
	}
	validResponse := httptest.NewRecorder()
	router.ServeHTTP(validResponse, validRequest)
	if validResponse.Code != http.StatusCreated {
		t.Fatalf("valid signed register: expected 201, got %d: %s", validResponse.Code, validResponse.Body.String())
	}

	var token struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(validResponse.Body.Bytes(), &token); err != nil {
		t.Fatalf("decode signed register response: %v", err)
	}
	if token.Data.AccessToken == "" {
		t.Fatal("signed register: expected access token")
	}

	tamperedBody, err := json.Marshal(map[string]string{"email": "signed-e2e@example.com", "password": "tampered-password"})
	if err != nil {
		t.Fatalf("encode tampered body: %v", err)
	}
	tamperedRequest := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(tamperedBody))
	tamperedRequest.Header.Set("Content-Type", "application/json")
	for key, value := range testsupport.APISignatureHeaders(http.MethodPost, path, body, now, "e2e-nonce-tampered", "e2e-integration", "e2e-api-signing-secret") {
		tamperedRequest.Header.Set(key, value)
	}
	tamperedResponse := httptest.NewRecorder()
	router.ServeHTTP(tamperedResponse, tamperedRequest)
	if tamperedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("tampered signed register: expected 401, got %d: %s", tamperedResponse.Code, tamperedResponse.Body.String())
	}

	replayRequest := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	replayRequest.Header.Set("Content-Type", "application/json")
	for key, value := range testsupport.APISignatureHeaders(http.MethodPost, path, body, now, "e2e-nonce-valid", "e2e-integration", "e2e-api-signing-secret") {
		replayRequest.Header.Set(key, value)
	}
	replayResponse := httptest.NewRecorder()
	router.ServeHTTP(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusUnauthorized {
		t.Fatalf("replayed signed register: expected 401, got %d: %s", replayResponse.Code, replayResponse.Body.String())
	}

	mePath := "/api/v1/auth/me"
	meRequest := httptest.NewRequest(http.MethodGet, mePath, nil)
	meRequest.Header.Set("Authorization", "Bearer "+token.Data.AccessToken)
	for key, value := range testsupport.APISignatureHeaders(http.MethodGet, mePath, nil, now, "e2e-nonce-me", "e2e-integration", "e2e-api-signing-secret") {
		meRequest.Header.Set(key, value)
	}
	meResponse := httptest.NewRecorder()
	router.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("signed auth me: expected 200, got %d: %s", meResponse.Code, meResponse.Body.String())
	}

	catalogPath := "/api/v1/products?per_page=1"
	unsignedCatalogRequest := httptest.NewRequest(http.MethodGet, catalogPath, nil)
	unsignedCatalogResponse := httptest.NewRecorder()
	router.ServeHTTP(unsignedCatalogResponse, unsignedCatalogRequest)
	if unsignedCatalogResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned catalog list: expected 401, got %d: %s", unsignedCatalogResponse.Code, unsignedCatalogResponse.Body.String())
	}

	signedCatalogRequest := httptest.NewRequest(http.MethodGet, catalogPath, nil)
	for key, value := range testsupport.APISignatureHeaders(http.MethodGet, catalogPath, nil, now, "e2e-nonce-catalog", "e2e-integration", "e2e-api-signing-secret") {
		signedCatalogRequest.Header.Set(key, value)
	}
	signedCatalogResponse := httptest.NewRecorder()
	router.ServeHTTP(signedCatalogResponse, signedCatalogRequest)
	if signedCatalogResponse.Code != http.StatusOK {
		t.Fatalf("signed catalog list: expected 200, got %d: %s", signedCatalogResponse.Code, signedCatalogResponse.Body.String())
	}

	sellerPath := "/api/v1/seller/products?per_page=1"
	signedSellerRequest := httptest.NewRequest(http.MethodGet, sellerPath, nil)
	signedSellerRequest.Header.Set("Authorization", "Bearer "+token.Data.AccessToken)
	for key, value := range testsupport.APISignatureHeaders(http.MethodGet, sellerPath, nil, now, "e2e-nonce-seller", "e2e-integration", "e2e-api-signing-secret") {
		signedSellerRequest.Header.Set(key, value)
	}
	signedSellerResponse := httptest.NewRecorder()
	router.ServeHTTP(signedSellerResponse, signedSellerRequest)
	if signedSellerResponse.Code != http.StatusOK {
		t.Fatalf("signed seller catalog list: expected 200, got %d: %s", signedSellerResponse.Code, signedSellerResponse.Body.String())
	}
}
