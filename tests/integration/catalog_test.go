//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	testsupport "gameco-technical-assessment/tests/support"
)

func TestCatalogHTTPAgainstPostgres(t *testing.T) {
	cfg := testsupport.IntegrationConfig(t)
	server := testsupport.NewAPI(t, cfg)

	list := request(t, http.MethodGet, server.URL+"/api/v1/products?per_page=2&sort_by=name&sort_order=asc", "", nil)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("public catalog: expected 200, got %d: %s", list.StatusCode, list.Body)
	}
	var catalog catalogListEnvelope
	decode(t, list.Body, &catalog)
	if !catalog.Success || len(catalog.Data) != 2 {
		t.Fatalf("public catalog: expected two products, got success=%v data=%d", catalog.Success, len(catalog.Data))
	}
	if catalog.Meta.PerPage != 2 || !catalog.Meta.HasMore || catalog.Meta.NextCursor == nil || *catalog.Meta.NextCursor == "" {
		t.Fatalf("public catalog: expected cursor metadata, got %+v", catalog.Meta)
	}

	page := request(t, http.MethodGet, server.URL+"/api/v1/products?per_page=2&sort_by=name&sort_order=asc&cursor="+*catalog.Meta.NextCursor, "", nil)
	if page.StatusCode != http.StatusOK {
		t.Fatalf("public catalog next page: expected 200, got %d: %s", page.StatusCode, page.Body)
	}

	categories := request(t, http.MethodGet, server.URL+"/api/v1/categories/options", "", nil)
	if categories.StatusCode != http.StatusOK {
		t.Fatalf("category options: expected 200, got %d: %s", categories.StatusCode, categories.Body)
	}
	var categoryOptions categoryOptionsEnvelope
	decode(t, categories.Body, &categoryOptions)
	if len(categoryOptions.Data) < 4 {
		t.Fatalf("category options: expected seeded categories, got %d", len(categoryOptions.Data))
	}

	login := postJSON(t, server.URL+"/api/v1/auth/login", map[string]string{
		"email":    "user@mail.com",
		"password": "admin123",
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("seller login: expected 200, got %d: %s", login.StatusCode, login.Body)
	}
	var token tokenResponse
	decode(t, login.Body, &token)
	if token.Data.AccessToken == "" {
		t.Fatal("seller login: expected access token")
	}

	sellerList := request(t, http.MethodGet, server.URL+"/api/v1/seller/products?per_page=1", token.Data.AccessToken, nil)
	if sellerList.StatusCode != http.StatusOK {
		t.Fatalf("seller list: expected 200, got %d: %s", sellerList.StatusCode, sellerList.Body)
	}
	var sellerProducts sellerListEnvelope
	decode(t, sellerList.Body, &sellerProducts)
	if len(sellerProducts.Data) != 1 || !sellerProducts.Meta.HasMore {
		t.Fatalf("seller list: expected one item and more pages, got data=%d meta=%+v", len(sellerProducts.Data), sellerProducts.Meta)
	}

	name := fmt.Sprintf("Postgres Integration Product %d", time.Now().UnixNano())
	created := request(t, http.MethodPost, server.URL+"/api/v1/seller/products", token.Data.AccessToken, map[string]any{
		"name":               name,
		"description":        "Created through the PostgreSQL integration HTTP test",
		"price_amount":       1999,
		"stock_quantity":     7,
		"category_public_id": categoryOptions.Data[0].PublicID,
	})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("seller create: expected 201, got %d: %s", created.StatusCode, created.Body)
	}
	var createdProduct sellerItemEnvelope
	decode(t, created.Body, &createdProduct)
	if createdProduct.Data.PublicID == uuid.Nil || createdProduct.Data.Name != name {
		t.Fatalf("seller create: unexpected product %+v", createdProduct.Data)
	}

	detail := request(t, http.MethodGet, server.URL+"/api/v1/seller/products/"+createdProduct.Data.PublicID.String(), token.Data.AccessToken, nil)
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("seller detail: expected 200, got %d: %s", detail.StatusCode, detail.Body)
	}
	var detailProduct sellerItemEnvelope
	decode(t, detail.Body, &detailProduct)
	if detailProduct.Data.PublicID != createdProduct.Data.PublicID || detailProduct.Data.Name != name {
		t.Fatalf("seller detail: unexpected product %+v", detailProduct.Data)
	}
}

type catalogListEnvelope struct {
	Success bool             `json:"success"`
	Data    []catalogProduct `json:"data"`
	Meta    catalogMeta      `json:"meta"`
}

type catalogProduct struct {
	PublicID uuid.UUID `json:"public_id"`
}

type sellerListEnvelope struct {
	Success bool            `json:"success"`
	Data    []sellerProduct `json:"data"`
	Meta    catalogMeta     `json:"meta"`
}

type sellerItemEnvelope struct {
	Success bool          `json:"success"`
	Data    sellerProduct `json:"data"`
}

type sellerProduct struct {
	PublicID uuid.UUID `json:"public_id"`
	Name     string    `json:"name"`
}

type categoryOptionsEnvelope struct {
	Success bool             `json:"success"`
	Data    []categoryOption `json:"data"`
}

type categoryOption struct {
	PublicID uuid.UUID `json:"public_id"`
}

type catalogMeta struct {
	PerPage    int     `json:"per_page"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

func request(t *testing.T, method, url, accessToken string, payload any) httpResponse {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
		body = strings.NewReader(string(encoded))
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return httpResponse{StatusCode: response.StatusCode, Body: string(responseBody)}
}
