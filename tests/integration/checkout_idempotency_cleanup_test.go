//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gameco-technical-assessment/internal/modules/checkout"
	testsupport "gameco-technical-assessment/tests/support"
)

func TestCheckoutIdempotencyRetentionCleanupAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)

	email := fmt.Sprintf("idempotency-cleanup-%d@example.com", time.Now().UnixNano())
	registration := postJSON(t, server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "correct-horse-battery-staple",
	})
	if registration.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", registration.StatusCode, registration.Body)
	}
	var token tokenResponse
	decode(t, registration.Body, &token)

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
	checkoutResponse := checkoutRequest(t, server.URL+"/api/v1/checkout", token.Data.AccessToken, "idempotency-retention-source")
	if checkoutResponse.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", checkoutResponse.StatusCode, checkoutResponse.Body)
	}
	var payload checkoutEnvelope
	decode(t, checkoutResponse.Body, &payload)

	db := testsupport.OpenPostgres(t, cfg)
	var order struct {
		ID      uint
		BuyerID uint
	}
	if err := db.Raw("SELECT id, buyer_id FROM orders WHERE public_id = ?", payload.Data.PublicID).Scan(&order).Error; err != nil {
		t.Fatalf("load order: %v", err)
	}
	if order.ID == 0 || order.BuyerID == 0 {
		t.Fatalf("order lookup returned empty identifiers: %+v", order)
	}

	now := time.Now().UTC()
	oldKey := "idempotency-retention-old"
	recentKey := "idempotency-retention-recent"
	pendingKey := "idempotency-retention-pending"
	requestHash := checkout.RequestHash()
	for _, row := range []struct {
		key       string
		orderID   *uint
		updatedAt time.Time
	}{
		{key: oldKey, orderID: &order.ID, updatedAt: now.Add(-48 * time.Hour)},
		{key: recentKey, orderID: &order.ID, updatedAt: now.Add(-time.Hour)},
		{key: pendingKey, updatedAt: now.Add(-48 * time.Hour)},
	} {
		if err := db.Exec(`
			INSERT INTO checkout_idempotency_keys
				(buyer_id, idempotency_key, request_hash, order_id, response_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			order.BuyerID, row.key, requestHash, row.orderID, http.StatusCreated, row.updatedAt, row.updatedAt).Error; err != nil {
			t.Fatalf("insert %s idempotency row: %v", row.key, err)
		}
	}

	deleted, err := checkout.NewRepository(db).CleanupCompletedIdempotencyKeys(t.Context(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("cleanup idempotency rows: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one completed idempotency row deleted, got %d", deleted)
	}

	var remaining int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM checkout_idempotency_keys
		WHERE buyer_id = ? AND idempotency_key IN (?, ?, ?)`, order.BuyerID, oldKey, recentKey, pendingKey).Scan(&remaining).Error; err != nil {
		t.Fatalf("count idempotency rows: %v", err)
	}
	if remaining != 2 {
		t.Fatalf("expected recent completed and pending rows to remain, got %d", remaining)
	}
}
