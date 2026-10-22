package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersAreAppliedToResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/resource", func(c *gin.Context) {
		c.Status(204)
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/resource", nil)
	router.ServeHTTP(record, request)

	expectations := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	for header, expected := range expectations {
		if got := record.Header().Get(header); got != expected {
			t.Errorf("%s = %q, want %q", header, got, expected)
		}
	}
}

func TestSecurityHeadersDoNotPreventHandlerResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/resource", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/resource", nil)
	router.ServeHTTP(record, request)

	if record.Code != 200 {
		t.Fatalf("status = %d, want 200", record.Code)
	}
	if got := record.Body.String(); got != `{"status":"ok"}` {
		t.Fatalf("body = %q, want JSON response", got)
	}
}
