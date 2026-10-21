//go:build runtime

package runtime_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCartItemLifecycle(t *testing.T) {
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
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	decodeJSON(t, body, &registerResponse)
	if registerResponse.Data.AccessToken == "" {
		t.Fatal("register: expected access token")
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/products?per_page=1", nil, "")
	if status != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", status, body)
	}
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	decodeJSON(t, body, &products)
	if len(products.Data) != 1 {
		t.Fatalf("products: expected one product, got %+v", products.Data)
	}
	productID := products.Data[0].PublicID

	itemBody := bytes.NewBufferString(`{"quantity":1}`)
	status, body = requestJSON(t, http.MethodPut, baseURL+"/api/v1/cart/items/"+productID, itemBody, registerResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("set cart item: expected 200, got %d: %s", status, body)
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/cart", nil, registerResponse.Data.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("get cart: expected 200, got %d: %s", status, body)
	}
	var cart struct {
		Data struct {
			Items []struct {
				ProductPublicID string `json:"product_public_id"`
				Quantity        int    `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	decodeJSON(t, body, &cart)
	if len(cart.Data.Items) != 1 || cart.Data.Items[0].ProductPublicID != productID || cart.Data.Items[0].Quantity != 1 {
		t.Fatalf("unexpected cart: %+v", cart.Data.Items)
	}

	checkoutRequest := bytes.NewBuffer(nil)
	status, body = requestJSONWithHeaders(t, http.MethodPost, baseURL+"/api/v1/checkout", checkoutRequest, registerResponse.Data.AccessToken, map[string]string{"Idempotency-Key": "runtime-checkout-key"})
	if status != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", status, body)
	}
	var firstCheckout struct {
		Data struct {
			PublicID         string `json:"public_id"`
			PaymentReference string `json:"payment_reference"`
		} `json:"data"`
	}
	decodeJSON(t, body, &firstCheckout)
	if firstCheckout.Data.PublicID == "" || firstCheckout.Data.PaymentReference == "" {
		t.Fatalf("checkout: missing order/payment result: %+v", firstCheckout.Data)
	}

	status, body = requestJSONWithHeaders(t, http.MethodPost, baseURL+"/api/v1/checkout", bytes.NewBuffer(nil), registerResponse.Data.AccessToken, map[string]string{"Idempotency-Key": "runtime-checkout-key"})
	if status != http.StatusCreated {
		t.Fatalf("checkout retry: expected 201, got %d: %s", status, body)
	}
	var secondCheckout struct {
		Data struct {
			PublicID         string `json:"public_id"`
			PaymentReference string `json:"payment_reference"`
		} `json:"data"`
	}
	decodeJSON(t, body, &secondCheckout)
	if firstCheckout.Data != secondCheckout.Data {
		t.Fatalf("checkout retry did not replay original result: first=%+v second=%+v", firstCheckout.Data, secondCheckout.Data)
	}
}
