//go:build integration

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestCheckoutHTTPAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)

	login := postJSON(t, server.URL+"/api/v1/auth/register", map[string]string{
		"email":    "checkout-buyer@example.com",
		"password": "correct-horse-battery-staple",
	})
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", login.StatusCode, login.Body)
	}
	var token tokenResponse
	decode(t, login.Body, &token)

	products := request(t, http.MethodGet, server.URL+"/api/v1/products?per_page=1", token.Data.AccessToken, nil)
	if products.StatusCode != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", products.StatusCode, products.Body)
	}
	var catalog catalogListEnvelope
	decode(t, products.Body, &catalog)
	if len(catalog.Data) != 1 {
		t.Fatalf("products: expected one product, got %+v", catalog.Data)
	}
	productID := catalog.Data[0].PublicID.String()

	added := request(t, http.MethodPut, server.URL+"/api/v1/cart/items/"+productID, token.Data.AccessToken, map[string]int{"quantity": 1})
	if added.StatusCode != http.StatusOK {
		t.Fatalf("set cart item: expected 200, got %d: %s", added.StatusCode, added.Body)
	}

	first := checkoutRequest(t, server.URL+"/api/v1/checkout", token.Data.AccessToken, "postgres-checkout-key")
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", first.StatusCode, first.Body)
	}
	var firstPayload checkoutEnvelope
	decode(t, first.Body, &firstPayload)
	if firstPayload.Data.PublicID == "" || firstPayload.Data.PaymentReference == "" {
		t.Fatalf("checkout: missing order/payment result: %+v", firstPayload.Data)
	}

	detail := request(t, http.MethodGet, server.URL+"/api/v1/orders/"+firstPayload.Data.PublicID, token.Data.AccessToken, nil)
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("order detail: expected 200, got %d: %s", detail.StatusCode, detail.Body)
	}
	var detailPayload struct {
		Data struct {
			PublicID      string `json:"public_id"`
			PaymentStatus string `json:"payment_status"`
			Items         []struct {
				Quantity int `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	decode(t, detail.Body, &detailPayload)
	if detailPayload.Data.PublicID != firstPayload.Data.PublicID || detailPayload.Data.PaymentStatus != "pending" || len(detailPayload.Data.Items) != 1 || detailPayload.Data.Items[0].Quantity != 1 {
		t.Fatalf("unexpected order detail: %+v", detailPayload.Data)
	}

	foreignRegistration := postJSON(t, server.URL+"/api/v1/auth/register", map[string]string{
		"email":    "checkout-foreign-buyer@example.com",
		"password": "correct-horse-battery-staple",
	})
	if foreignRegistration.StatusCode != http.StatusCreated {
		t.Fatalf("foreign buyer register: expected 201, got %d: %s", foreignRegistration.StatusCode, foreignRegistration.Body)
	}
	var foreignToken tokenResponse
	decode(t, foreignRegistration.Body, &foreignToken)
	foreignDetail := request(t, http.MethodGet, server.URL+"/api/v1/orders/"+firstPayload.Data.PublicID, foreignToken.Data.AccessToken, nil)
	if foreignDetail.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign order detail: expected 404, got %d: %s", foreignDetail.StatusCode, foreignDetail.Body)
	}

	second := checkoutRequest(t, server.URL+"/api/v1/checkout", token.Data.AccessToken, "postgres-checkout-key")
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("checkout retry: expected 201, got %d: %s", second.StatusCode, second.Body)
	}
	var secondPayload checkoutEnvelope
	decode(t, second.Body, &secondPayload)
	if firstPayload.Data != secondPayload.Data {
		t.Fatalf("checkout retry did not replay original result: first=%+v second=%+v", firstPayload.Data, secondPayload.Data)
	}

	cart := request(t, http.MethodGet, server.URL+"/api/v1/cart", token.Data.AccessToken, nil)
	if cart.StatusCode != http.StatusOK {
		t.Fatalf("cart after checkout: expected 200, got %d: %s", cart.StatusCode, cart.Body)
	}
	var cartPayload cartEnvelope
	decode(t, cart.Body, &cartPayload)
	if len(cartPayload.Data.Items) != 0 || len(cartPayload.Data.UnavailableItems) != 0 {
		t.Fatalf("cart after checkout is not empty: %+v", cartPayload.Data)
	}
}

type checkoutEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		PublicID         string `json:"public_id"`
		Status           string `json:"status"`
		TotalAmount      int64  `json:"total_amount"`
		PaymentReference string `json:"payment_reference"`
	} `json:"data"`
}

func checkoutRequest(t *testing.T, url, accessToken, idempotencyKey string) httpResponse {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(""))
	if err != nil {
		t.Fatalf("create checkout request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Idempotency-Key", idempotencyKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("checkout request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read checkout response: %v", err)
	}
	return httpResponse{StatusCode: resp.StatusCode, Body: string(body)}
}
