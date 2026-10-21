//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"

	"gorm.io/gorm"
)

func TestCheckoutConcurrencyAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	db := testsupport.OpenPostgres(t, cfg)
	server := testsupport.NewAPI(t, cfg)

	productID := integrationProductID(t, server.URL)
	initialStock := productStock(t, db, productID)
	t.Cleanup(func() {
		setProductStock(t, db, productID, initialStock)
	})
	setProductStock(t, db, productID, 1)
	ordersBefore := countRows(t, db, "orders")
	paymentsBefore := countRows(t, db, "payments")
	movementsBefore := countRows(t, db, "stock_movements")

	buyerA := registerBuyer(t, server.URL, "concurrent-a")
	buyerB := registerBuyer(t, server.URL, "concurrent-b")
	addToCart(t, server.URL, buyerA, productID)
	addToCart(t, server.URL, buyerB, productID)

	results := runConcurrentCheckouts(server.URL, []string{buyerA, buyerB}, []string{"concurrent-key-a", "concurrent-key-b"})
	created := 0
	conflicts := 0
	for _, result := range results {
		switch result.StatusCode {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent checkout response: %d %s", result.StatusCode, result.Body)
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("last-stock checkout results: created=%d conflicts=%d results=%+v", created, conflicts, results)
	}
	if got := productStock(t, db, productID); got != 0 {
		t.Fatalf("last-stock checkout oversold or failed to reserve stock: got stock=%d", got)
	}
	assertCountDelta(t, db, "orders", ordersBefore, 1)
	assertCountDelta(t, db, "payments", paymentsBefore, 1)
	assertCountDelta(t, db, "stock_movements", movementsBefore, 1)

	setProductStock(t, db, productID, 1)
	ordersBefore = countRows(t, db, "orders")
	paymentsBefore = countRows(t, db, "payments")
	movementsBefore = countRows(t, db, "stock_movements")
	buyerC := registerBuyer(t, server.URL, "concurrent-same-key")
	addToCart(t, server.URL, buyerC, productID)

	results = runConcurrentCheckouts(server.URL, []string{buyerC, buyerC}, []string{"same-key", "same-key"})
	for _, result := range results {
		if result.StatusCode != http.StatusCreated {
			t.Fatalf("same-key concurrent checkout: expected 201, got %d: %s", result.StatusCode, result.Body)
		}
	}
	var first, second checkoutEnvelope
	decode(t, results[0].Body, &first)
	decode(t, results[1].Body, &second)
	if first.Data != second.Data {
		t.Fatalf("same-key concurrent checkout did not replay one result: first=%+v second=%+v", first.Data, second.Data)
	}
	assertCountDelta(t, db, "orders", ordersBefore, 1)
	assertCountDelta(t, db, "payments", paymentsBefore, 1)
	assertCountDelta(t, db, "stock_movements", movementsBefore, 1)

	setProductStock(t, db, productID, 1)
	ordersBefore = countRows(t, db, "orders")
	paymentsBefore = countRows(t, db, "payments")
	movementsBefore = countRows(t, db, "stock_movements")
	buyerD := registerBuyer(t, server.URL, "concurrent-rollback")
	addToCart(t, server.URL, buyerD, productID)
	setProductStock(t, db, productID, 0)
	failed := rawCheckoutRequest(server.URL+"/api/v1/checkout", buyerD, "rollback-retry-key")
	if failed.StatusCode != http.StatusConflict {
		t.Fatalf("rollback checkout: expected 409, got %d: %s", failed.StatusCode, failed.Body)
	}
	setProductStock(t, db, productID, 1)
	retried := rawCheckoutRequest(server.URL+"/api/v1/checkout", buyerD, "rollback-retry-key")
	if retried.StatusCode != http.StatusCreated {
		t.Fatalf("rollback retry checkout: expected 201, got %d: %s", retried.StatusCode, retried.Body)
	}
	assertCountDelta(t, db, "orders", ordersBefore, 1)
	assertCountDelta(t, db, "payments", paymentsBefore, 1)
	assertCountDelta(t, db, "stock_movements", movementsBefore, 1)
}

func integrationProductID(t *testing.T, baseURL string) string {
	t.Helper()
	products := request(t, http.MethodGet, baseURL+"/api/v1/products?per_page=1", "", nil)
	if products.StatusCode != http.StatusOK {
		t.Fatalf("products: expected 200, got %d: %s", products.StatusCode, products.Body)
	}
	var catalog catalogListEnvelope
	decode(t, products.Body, &catalog)
	if len(catalog.Data) != 1 {
		t.Fatalf("products: expected one product, got %+v", catalog.Data)
	}
	return catalog.Data[0].PublicID.String()
}

func registerBuyer(t *testing.T, baseURL, label string) string {
	t.Helper()
	email := fmt.Sprintf("%s-%d@example.com", label, time.Now().UnixNano())
	response := postJSON(t, baseURL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "correct-horse-battery-staple",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", label, response.StatusCode, response.Body)
	}
	var token tokenResponse
	decode(t, response.Body, &token)
	return token.Data.AccessToken
}

func addToCart(t *testing.T, baseURL, token, productID string) {
	t.Helper()
	response := request(t, http.MethodPut, baseURL+"/api/v1/cart/items/"+productID, token, map[string]int{"quantity": 1})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("set cart item: expected 200, got %d: %s", response.StatusCode, response.Body)
	}
}

func runConcurrentCheckouts(baseURL string, tokens, keys []string) []httpResponse {
	results := make([]httpResponse, len(tokens))
	var waitGroup sync.WaitGroup
	for index := range tokens {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			results[index] = rawCheckoutRequest(baseURL+"/api/v1/checkout", tokens[index], keys[index])
		}(index)
	}
	waitGroup.Wait()
	return results
}

func rawCheckoutRequest(url, accessToken, idempotencyKey string) httpResponse {
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return httpResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Idempotency-Key", idempotencyKey)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return httpResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return httpResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}
	}
	return httpResponse{StatusCode: response.StatusCode, Body: string(body)}
}

func setProductStock(t *testing.T, db *gorm.DB, productID string, stock int) {
	t.Helper()
	if err := db.Exec("UPDATE products SET stock_quantity = ? WHERE public_id = ?", stock, productID).Error; err != nil {
		t.Fatalf("set product stock: %v", err)
	}
}

func productStock(t *testing.T, db *gorm.DB, productID string) int {
	t.Helper()
	var stock int
	if err := db.Raw("SELECT stock_quantity FROM products WHERE public_id = ?", productID).Scan(&stock).Error; err != nil {
		t.Fatalf("read product stock: %v", err)
	}
	return stock
}

func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func assertCountDelta(t *testing.T, db *gorm.DB, table string, before, delta int64) {
	t.Helper()
	after := countRows(t, db, table)
	if after != before+delta {
		t.Fatalf("unexpected %s count: before=%d after=%d delta=%d", table, before, after, delta)
	}
}
