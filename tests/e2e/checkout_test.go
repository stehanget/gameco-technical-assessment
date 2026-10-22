//go:build e2e

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

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+firstEnvelope.Data.PublicID, nil)
	detailRequest.Header.Set("Authorization", "Bearer "+token)
	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("order detail: expected 200, got %d: %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detailEnvelope struct {
		Data struct {
			PublicID         string `json:"public_id"`
			PaymentReference string `json:"payment_reference"`
			PaymentStatus    string `json:"payment_status"`
			Items            []struct {
				ProductName string `json:"product_name"`
				Quantity    int    `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detailEnvelope); err != nil {
		t.Fatal(err)
	}
	if detailEnvelope.Data.PublicID != firstEnvelope.Data.PublicID || detailEnvelope.Data.PaymentReference != firstEnvelope.Data.PaymentReference || detailEnvelope.Data.PaymentStatus != "pending" || len(detailEnvelope.Data.Items) != 1 || detailEnvelope.Data.Items[0].Quantity != 1 {
		t.Fatalf("unexpected order detail: %+v", detailEnvelope.Data)
	}

	otherBuyerToken := e2ERegister(t, router)
	foreignRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+firstEnvelope.Data.PublicID, nil)
	foreignRequest.Header.Set("Authorization", "Bearer "+otherBuyerToken)
	foreignResponse := httptest.NewRecorder()
	router.ServeHTTP(foreignResponse, foreignRequest)
	if foreignResponse.Code != http.StatusNotFound {
		t.Fatalf("foreign order detail: expected 404, got %d: %s", foreignResponse.Code, foreignResponse.Body.String())
	}

	malformedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders/not-a-uuid", nil)
	malformedRequest.Header.Set("Authorization", "Bearer "+token)
	malformedResponse := httptest.NewRecorder()
	router.ServeHTTP(malformedResponse, malformedRequest)
	if malformedResponse.Code != http.StatusNotFound {
		t.Fatalf("malformed order detail: expected 404, got %d: %s", malformedResponse.Code, malformedResponse.Body.String())
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

func TestSellerOrdersAreScopedToSellerItems(t *testing.T) {
	router := newE2ERouter(t)
	buyerToken := e2ERegister(t, router)

	productsResponse := httptest.NewRecorder()
	router.ServeHTTP(productsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil))
	if productsResponse.Code != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", productsResponse.Code, productsResponse.Body.String())
	}
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productsResponse.Body.Bytes(), &products); err != nil {
		t.Fatal(err)
	}
	if len(products.Data) != 1 {
		t.Fatalf("expected one seeded product, got %s", productsResponse.Body.String())
	}

	cartRequest := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+products.Data[0].PublicID, bytes.NewBufferString(`{"quantity":1}`))
	cartRequest.Header.Set("Authorization", "Bearer "+buyerToken)
	cartRequest.Header.Set("Content-Type", "application/json")
	cartResponse := httptest.NewRecorder()
	router.ServeHTTP(cartResponse, cartRequest)
	if cartResponse.Code != http.StatusOK {
		t.Fatalf("cart: expected 200, got %d: %s", cartResponse.Code, cartResponse.Body.String())
	}
	checkoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", nil)
	checkoutRequest.Header.Set("Authorization", "Bearer "+buyerToken)
	checkoutRequest.Header.Set("Idempotency-Key", "seller-orders-e2e-key")
	checkoutResponse := httptest.NewRecorder()
	router.ServeHTTP(checkoutResponse, checkoutRequest)
	if checkoutResponse.Code != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", checkoutResponse.Code, checkoutResponse.Body.String())
	}
	var checkoutPayload struct {
		Data struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkoutResponse.Body.Bytes(), &checkoutPayload); err != nil {
		t.Fatal(err)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@mail.com","password":"admin123"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("seller login: expected 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	sellerToken := decodeAccessToken(t, loginResponse.Body.Bytes())

	ordersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/seller/orders?per_page=1", nil)
	ordersRequest.Header.Set("Authorization", "Bearer "+sellerToken)
	ordersResponse := httptest.NewRecorder()
	router.ServeHTTP(ordersResponse, ordersRequest)
	if ordersResponse.Code != http.StatusOK {
		t.Fatalf("seller orders: expected 200, got %d: %s", ordersResponse.Code, ordersResponse.Body.String())
	}
	var ordersPayload struct {
		Data []struct {
			PublicID          string `json:"public_id"`
			SellerTotalAmount int64  `json:"seller_total_amount"`
			Items             []struct {
				ProductPublicID string `json:"product_public_id"`
			} `json:"items"`
		} `json:"data"`
		Meta struct {
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(ordersResponse.Body.Bytes(), &ordersPayload); err != nil {
		t.Fatal(err)
	}
	if len(ordersPayload.Data) != 1 || ordersPayload.Meta.HasMore || ordersPayload.Meta.NextCursor != nil || ordersPayload.Data[0].PublicID != checkoutPayload.Data.PublicID || len(ordersPayload.Data[0].Items) != 1 || ordersPayload.Data[0].Items[0].ProductPublicID != products.Data[0].PublicID {
		t.Fatalf("unexpected seller orders response: %s", ordersResponse.Body.String())
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
