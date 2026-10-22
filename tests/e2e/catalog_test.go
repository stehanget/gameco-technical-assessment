//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicCatalogListAndDetail(t *testing.T) {
	router := newE2ERouter(t)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=2&sort_by=name&sort_order=asc", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("catalog list: expected status 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload struct {
		Success bool `json:"success"`
		Data    []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
		Meta struct {
			PerPage    int     `json:"per_page"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode catalog list: %v", err)
	}
	if !listPayload.Success || len(listPayload.Data) != 2 {
		t.Fatalf("catalog list: expected two products, got %+v", listPayload)
	}
	if listPayload.Meta.PerPage != 2 || !listPayload.Meta.HasMore || listPayload.Meta.NextCursor == nil || *listPayload.Meta.NextCursor == "" {
		t.Fatalf("catalog list: expected cursor metadata, got %+v", listPayload.Meta)
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+listPayload.Data[0].PublicID, nil)
	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("catalog detail: expected status 200, got %d: %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detailPayload struct {
		Success bool `json:"success"`
		Data    struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode catalog detail: %v", err)
	}
	if !detailPayload.Success || detailPayload.Data.PublicID != listPayload.Data[0].PublicID {
		t.Fatalf("catalog detail: unexpected response %+v", detailPayload)
	}
}

func TestSellerProductRejectsUnauthenticatedRequest(t *testing.T) {
	router := newE2ERouter(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("seller products without token: expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSellerProductRejectsBuyerOnlyActor(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	if err := db.Exec(`DELETE FROM user_roles WHERE user_id = (SELECT id FROM users WHERE email = ?) AND role_id = (SELECT id FROM roles WHERE code = ?)`, "user@mail.com", "seller").Error; err != nil {
		t.Fatalf("remove seller role: %v", err)
	}
	accessToken := e2ELogin(t, router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("buyer-only seller products: expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSellerProductRejectsMissingPermission(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	if err := db.Exec(`DELETE FROM role_permissions WHERE permission_id = (SELECT id FROM permissions WHERE code = ?)`, "product.read").Error; err != nil {
		t.Fatalf("remove seller permission: %v", err)
	}
	accessToken := e2ELogin(t, router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("seller without permission: expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSellerProductRejectsInvalidCreatePayload(t *testing.T) {
	router := newE2ERouter(t)
	accessToken := e2ELogin(t, router)

	requestBody := bytes.NewBufferString(`{"name":"Invalid Product","description":"invalid price","price_amount":-1,"stock_quantity":1,"category_public_id":"00000000-0000-0000-0000-000000000000"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/seller/products", requestBody)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("seller create invalid payload: expected status 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSellerPriceChangeCreatesAuditEvent(t *testing.T) {
	router, db := newE2ERouterWithDB(t)
	accessToken := e2ELogin(t, router)
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products?per_page=1", nil)
	listRequest.Header.Set("Authorization", "Bearer "+accessToken)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("seller products: expected status 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var products struct {
		Data []struct {
			PublicID string `json:"public_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &products); err != nil {
		t.Fatal(err)
	}
	if len(products.Data) != 1 {
		t.Fatalf("expected one seller product, got %s", listResponse.Body.String())
	}
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/seller/products/"+products.Data[0].PublicID, bytes.NewBufferString(`{"price_amount":7777}`))
	updateRequest.Header.Set("Authorization", "Bearer "+accessToken)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("seller price update: expected status 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}
	var auditCount int64
	if err := db.Raw("SELECT COUNT(*) FROM audit_events WHERE event_type = ? AND aggregate_type = ? AND aggregate_id = ?", "product.price_changed", "product", products.Data[0].PublicID).Scan(&auditCount).Error; err != nil {
		t.Fatalf("count product audit events: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one product price audit event, got %d", auditCount)
	}
}

func e2ELogin(t *testing.T, router http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@mail.com","password":"admin123"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login: expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Data.AccessToken == "" {
		t.Fatal("login: expected access token")
	}
	return payload.Data.AccessToken
}
