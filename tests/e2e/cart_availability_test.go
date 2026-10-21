package e2e

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCartRejectsUnavailableProducts(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	accessToken := e2ELogin(t, router)

	var product struct {
		PublicID string `gorm:"column:public_id"`
	}
	if err := db.Table("products").Select("public_id").First(&product).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if err := db.Exec("UPDATE products SET status = ? WHERE public_id = ?", "inactive", product.PublicID).Error; err != nil {
		t.Fatalf("deactivate product: %v", err)
	}
	assertCartProductUnavailable(t, router, accessToken, product.PublicID)

	if err := db.Exec("UPDATE products SET status = ?, deleted_at = ? WHERE public_id = ?", "active", time.Now(), product.PublicID).Error; err != nil {
		t.Fatalf("soft-delete product: %v", err)
	}
	assertCartProductUnavailable(t, router, accessToken, product.PublicID)
}

func assertCartProductUnavailable(t *testing.T, router http.Handler, accessToken, productID string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/cart/items/"+productID, bytes.NewBufferString(`{"quantity":1}`))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unavailable product: expected status 404, got %d: %s", response.Code, response.Body.String())
	}
}
