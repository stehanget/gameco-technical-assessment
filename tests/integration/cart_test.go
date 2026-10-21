//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestCartHTTPAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)

	login := postJSON(t, server.URL+"/api/v1/auth/login", map[string]string{
		"email":    "user@mail.com",
		"password": "admin123",
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", login.StatusCode, login.Body)
	}
	var token tokenResponse
	decode(t, login.Body, &token)

	products := request(t, http.MethodGet, server.URL+"/api/v1/products?per_page=1", token.Data.AccessToken, nil)
	if products.StatusCode != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", products.StatusCode, products.Body)
	}
	var catalog catalogListEnvelope
	decode(t, products.Body, &catalog)
	if len(catalog.Data) != 1 || catalog.Data[0].PublicID == uuid.Nil {
		t.Fatalf("products: expected one public product, got %+v", catalog.Data)
	}
	productID := catalog.Data[0].PublicID.String()

	setItem := request(t, http.MethodPut, server.URL+"/api/v1/cart/items/"+productID, token.Data.AccessToken, map[string]int{"quantity": 1})
	if setItem.StatusCode != http.StatusOK {
		t.Fatalf("set cart item: expected 200, got %d: %s", setItem.StatusCode, setItem.Body)
	}

	cart := request(t, http.MethodGet, server.URL+"/api/v1/cart", token.Data.AccessToken, nil)
	if cart.StatusCode != http.StatusOK {
		t.Fatalf("get cart: expected 200, got %d: %s", cart.StatusCode, cart.Body)
	}
	var cartPayload cartEnvelope
	decode(t, cart.Body, &cartPayload)
	if len(cartPayload.Data.Items) != 1 || cartPayload.Data.Items[0].ProductPublicID != productID || cartPayload.Data.Items[0].Quantity != 1 {
		t.Fatalf("unexpected cart: %+v", cartPayload.Data)
	}

	deleted := request(t, http.MethodDelete, server.URL+"/api/v1/cart/items/"+productID, token.Data.AccessToken, nil)
	if deleted.StatusCode != http.StatusOK {
		t.Fatalf("delete cart item: expected 200, got %d: %s", deleted.StatusCode, deleted.Body)
	}
}

type cartEnvelope struct {
	Success bool        `json:"success"`
	Data    cartPayload `json:"data"`
}

type cartPayload struct {
	Items []cartItem `json:"items"`
}

type cartItem struct {
	ProductPublicID string `json:"product_public_id"`
	Quantity        int    `json:"quantity"`
}
