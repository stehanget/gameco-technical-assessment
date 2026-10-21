package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckoutIsIdempotentAndReservesStock(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	token := e2ERegister(t, router)

	productsResponse := httptest.NewRecorder()
	router.ServeHTTP(productsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil))
	if productsResponse.Code != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", productsResponse.Code, productsResponse.Body.String())
	}
	var products struct {
		Data []struct {
			PublicID      string `json:"public_id"`
			StockQuantity int    `json:"stock_quantity"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productsResponse.Body.Bytes(), &products); err != nil {
		t.Fatal(err)
	}
	if len(products.Data) != 1 || products.Data[0].StockQuantity < 1 {
		t.Fatalf("expected in-stock product, got %+v", products.Data)
	}
	productID, stockBefore := products.Data[0].PublicID, products.Data[0].StockQuantity

	cartRequest := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+productID, bytes.NewBufferString(`{"quantity":1}`))
	cartRequest.Header.Set("Authorization", "Bearer "+token)
	cartRequest.Header.Set("Content-Type", "application/json")
	cartResponse := httptest.NewRecorder()
	router.ServeHTTP(cartResponse, cartRequest)
	if cartResponse.Code != http.StatusOK {
		t.Fatalf("cart: expected 200, got %d: %s", cartResponse.Code, cartResponse.Body.String())
	}

	checkout := func() (int, []byte) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Idempotency-Key", "checkout-e2e-key")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res.Code, res.Body.Bytes()
	}
	status, first := checkout()
	if status != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", status, first)
	}
	status, second := checkout()
	if status != http.StatusCreated {
		t.Fatalf("checkout retry: expected 201, got %d: %s", status, second)
	}
	var firstEnvelope, secondEnvelope struct {
		Data struct {
			PublicID         string `json:"public_id"`
			PaymentReference string `json:"payment_reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(first, &firstEnvelope); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &secondEnvelope); err != nil {
		t.Fatal(err)
	}
	if firstEnvelope.Data.PublicID == "" || firstEnvelope.Data.PublicID != secondEnvelope.Data.PublicID || firstEnvelope.Data.PaymentReference != secondEnvelope.Data.PaymentReference {
		t.Fatalf("retry did not replay original result: first=%s second=%s", first, second)
	}

	var stockAfter, orderCount, paymentCount, movementCount int64
	db.Raw("SELECT stock_quantity FROM products WHERE public_id = ?", productID).Scan(&stockAfter)
	db.Raw("SELECT COUNT(*) FROM orders").Scan(&orderCount)
	db.Raw("SELECT COUNT(*) FROM payments").Scan(&paymentCount)
	db.Raw("SELECT COUNT(*) FROM stock_movements").Scan(&movementCount)
	if stockAfter != int64(stockBefore-1) || orderCount != 1 || paymentCount != 1 || movementCount != 1 {
		t.Fatalf("unexpected checkout effects: stock=%d orders=%d payments=%d movements=%d", stockAfter, orderCount, paymentCount, movementCount)
	}
}

func TestCheckoutRequiresIdempotencyKey(t *testing.T) {
	router := newE2ERouter(t)
	token := e2ERegister(t, router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency key: expected 400, got %d: %s", res.Code, res.Body.String())
	}
}
