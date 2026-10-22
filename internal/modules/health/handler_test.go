package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLiveDoesNotDependOnInfrastructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, NewHandler(nil, nil))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("livez: expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestReadyRejectsMissingInfrastructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, NewHandler(nil, nil))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz: expected 503, got %d: %s", response.Code, response.Body.String())
	}
}
