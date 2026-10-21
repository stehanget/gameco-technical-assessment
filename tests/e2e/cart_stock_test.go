//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCartRejectsQuantityAboveCurrentStock(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	accessToken := e2ERegister(t, router)

	var product struct {
		PublicID string `gorm:"column:public_id"`
	}
	if err := db.Table("products").Select("public_id").First(&product).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if err := db.Exec("UPDATE products SET stock_quantity = ? WHERE public_id = ?", 1, product.PublicID).Error; err != nil {
		t.Fatalf("set product stock: %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+product.PublicID, bytes.NewBufferString(`{"quantity":2}`))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("quantity above stock: expected status 409, got %d: %s", response.Code, response.Body.String())
	}

	var cart struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("get cart after rejected quantity: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &cart); err != nil {
		t.Fatalf("decode cart: %v", err)
	}
	if len(cart.Data.Items) != 0 {
		t.Fatalf("rejected quantity created cart item: %+v", cart.Data.Items)
	}
}
