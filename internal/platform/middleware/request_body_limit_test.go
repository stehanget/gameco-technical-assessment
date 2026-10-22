package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestBodyLimitRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit(4))
	router.POST("/resource", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil {
			t.Fatalf("expected body read error, got body %q", body)
		}
		c.Status(400)
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/resource", strings.NewReader("12345"))
	router.ServeHTTP(record, request)
	if record.Code != 400 {
		t.Fatalf("status = %d, want 400", record.Code)
	}
}

func TestRequestBodyLimitPreservesBodyBelowLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit(5))
	router.POST("/resource", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatal(err)
		}
		c.String(200, string(body))
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/resource", strings.NewReader("12345"))
	router.ServeHTTP(record, request)
	if record.Code != 200 || record.Body.String() != "12345" {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}
