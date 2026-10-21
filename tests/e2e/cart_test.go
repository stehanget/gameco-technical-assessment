//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuyerCartItemLifecycle(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	accessToken := e2ERegister(t, router)

	productRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=1", nil)
	productResponse := httptest.NewRecorder()
	router.ServeHTTP(productResponse, productRequest)
	if productResponse.Code != http.StatusOK {
		t.Fatalf("products: expected status 200, got %d: %s", productResponse.Code, productResponse.Body.String())
	}
	var products struct {
		Data []struct {
			PublicID      string `json:"public_id"`
			StockQuantity int    `json:"stock_quantity"`
		} `json:"data"`
	}
	if err := json.Unmarshal(productResponse.Body.Bytes(), &products); err != nil {
		t.Fatalf("decode products: %v", err)
	}
	if len(products.Data) != 1 || products.Data[0].StockQuantity < 1 {
		t.Fatalf("products: expected one in-stock product, got %+v", products.Data)
	}
	productID := products.Data[0].PublicID
	stockBefore := products.Data[0].StockQuantity

	request := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+productID, bytes.NewBufferString(`{"quantity":1}`))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("set cart item: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var stockAfter int
	if err := db.Raw("SELECT stock_quantity FROM products WHERE public_id = ?", productID).Scan(&stockAfter).Error; err != nil {
		t.Fatalf("read product stock: %v", err)
	}
	if stockAfter != stockBefore {
		t.Fatalf("cart mutation changed stock: before=%d after=%d", stockBefore, stockAfter)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("get cart: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var cart struct {
		Data struct {
			Items []struct {
				ProductPublicID string `json:"product_public_id"`
				Quantity        int    `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &cart); err != nil {
		t.Fatalf("decode cart: %v", err)
	}
	if len(cart.Data.Items) != 1 || cart.Data.Items[0].ProductPublicID != productID || cart.Data.Items[0].Quantity != 1 {
		t.Fatalf("unexpected cart response: %+v", cart.Data.Items)
	}

	request = httptest.NewRequest(http.MethodDelete, "/api/v1/cart/items/"+productID, nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("delete cart item: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &cart); err != nil {
		t.Fatalf("decode empty cart: %v", err)
	}
	if len(cart.Data.Items) != 0 {
		t.Fatalf("expected empty cart after delete, got %+v", cart.Data.Items)
	}
}

func TestCartRejectsUnauthenticatedRequest(t *testing.T) {
	router := newE2ERouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("cart without token: expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func e2ERegister(t *testing.T, router http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(fmt.Sprintf(`{"email":"buyer-%d@example.com","password":"correct-horse-battery-staple"}`, time.Now().UnixNano())))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register: expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	return decodeAccessToken(t, response.Body.Bytes())
}
