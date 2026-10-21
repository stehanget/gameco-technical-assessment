package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCartRejectsOwnProduct(t *testing.T) {
	router := newE2ERouter(t)
	accessToken := e2ELogin(t, router)

	productsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil)
	productsResponse := httptest.NewRecorder()
	router.ServeHTTP(productsResponse, productsRequest)
	if productsResponse.Code != http.StatusOK {
		t.Fatalf("products: expected status 200, got %d: %s", productsResponse.Code, productsResponse.Body.String())
	}
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productsResponse.Body.Bytes(), &products); err != nil {
		t.Fatalf("decode products: %v", err)
	}
	if len(products.Data) != 1 {
		t.Fatalf("products: expected one product, got %+v", products.Data)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+products.Data[0].PublicID, bytes.NewBufferString(`{"quantity":1}`))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("own product: expected status 409, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"commerce.self_purchase_not_allowed"`)) {
		t.Fatalf("own product: expected stable error code, got %s", response.Body.String())
	}
}

func TestCartRejectsMalformedAndUnknownProductIDs(t *testing.T) {
	router := newE2ERouter(t)
	accessToken := e2ERegister(t, router)

	cases := []struct {
		name   string
		id     string
		status int
		code   string
	}{
		{name: "malformed", id: "not-a-uuid", status: http.StatusBadRequest, code: "common.invalid_request"},
		{name: "unknown", id: "00000000-0000-0000-0000-000000000000", status: http.StatusNotFound, code: "cart.product_not_found"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+testCase.id, bytes.NewBufferString(`{"quantity":1}`))
			request.Header.Set("Authorization", "Bearer "+accessToken)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != testCase.status {
				t.Fatalf("product id: expected status %d, got %d: %s", testCase.status, response.Code, response.Body.String())
			}
			if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"`+testCase.code+`"`)) {
				t.Fatalf("product id: expected code %s, got %s", testCase.code, response.Body.String())
			}
		})
	}
}
