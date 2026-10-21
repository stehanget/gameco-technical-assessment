//go:build runtime

package runtime_test

import (
	"net/http"
	"testing"
)

func TestHealthAndPublicCatalog(t *testing.T) {
	baseURL := runtimeBaseURL(t)

	status, body := requestJSON(t, http.MethodGet, baseURL+"/health", nil, "")
	if status != http.StatusOK {
		t.Fatalf("health: expected 200, got %d: %s", status, body)
	}

	status, body = requestJSON(t, http.MethodGet, baseURL+"/api/v1/products", nil, "")
	if status != http.StatusOK {
		t.Fatalf("public catalog: expected 200, got %d: %s", status, body)
	}
}
