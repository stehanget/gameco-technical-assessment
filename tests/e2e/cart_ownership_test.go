package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCartIsScopedToAuthenticatedBuyer(t *testing.T) {
	router := newE2ERouter(t)
	firstToken := e2ERegister(t, router)

	productsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil)
	productsResponse := httptest.NewRecorder()
	router.ServeHTTP(productsResponse, productsRequest)
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productsResponse.Body.Bytes(), &products); err != nil {
		t.Fatalf("decode products: %v", err)
	}
	productID := products.Data[0].PublicID

	setRequest := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+productID, bytes.NewBufferString(`{"quantity":1}`))
	setRequest.Header.Set("Authorization", "Bearer "+firstToken)
	setRequest.Header.Set("Content-Type", "application/json")
	setResponse := httptest.NewRecorder()
	router.ServeHTTP(setResponse, setRequest)
	if setResponse.Code != http.StatusOK {
		t.Fatalf("first buyer set item: expected status 200, got %d: %s", setResponse.Code, setResponse.Body.String())
	}

	register := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"second-buyer@example.com","password":"correct-horse-battery-staple"}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	router.ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("second buyer register: expected status 201, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}
	secondToken := decodeAccessToken(t, registerResponse.Body.Bytes())

	cartRequest := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	cartRequest.Header.Set("Authorization", "Bearer "+secondToken)
	cartResponse := httptest.NewRecorder()
	router.ServeHTTP(cartResponse, cartRequest)
	if cartResponse.Code != http.StatusOK {
		t.Fatalf("second buyer cart: expected status 200, got %d: %s", cartResponse.Code, cartResponse.Body.String())
	}
	var cart struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(cartResponse.Body.Bytes(), &cart); err != nil {
		t.Fatalf("decode second buyer cart: %v", err)
	}
	if len(cart.Data.Items) != 0 {
		t.Fatalf("second buyer saw another buyer's cart items: %+v", cart.Data.Items)
	}
}

func decodeAccessToken(t *testing.T, body []byte) string {
	t.Helper()
	var payload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode access token: %v", err)
	}
	if payload.Data.AccessToken == "" {
		t.Fatal("expected access token")
	}
	return payload.Data.AccessToken
}
