package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitUsesRedisAndReturnsRateHeaders(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimit(client, RateLimitPolicy{
		Name:   "test",
		Limit:  2,
		Window: time.Minute,
	}))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		record := performRateLimitRequest(router)
		if record.Code != http.StatusOK {
			t.Fatalf("request %d: expected status 200, got %d", requestNumber, record.Code)
		}
		if got := record.Header().Get("X-RateLimit-Limit"); got != "2" {
			t.Fatalf("request %d: expected limit header 2, got %q", requestNumber, got)
		}
	}

	record := performRateLimitRequest(router)
	if record.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", record.Code)
	}
	if record.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if got := record.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("expected remaining header 0, got %q", got)
	}
}

func TestRateLimitSeparatesClientsByRemoteAddress(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimit(client, RateLimitPolicy{Name: "test", Limit: 1, Window: time.Minute}))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

	first := performRateLimitRequestFrom(router, "192.0.2.1:1234")
	second := performRateLimitRequestFrom(router, "192.0.2.2:1234")
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected separate clients to be allowed, got %d and %d", first.Code, second.Code)
	}
}

func performRateLimitRequest(router http.Handler) *httptest.ResponseRecorder {
	return performRateLimitRequestFrom(router, "192.0.2.1:1234")
}

func performRateLimitRequestFrom(router http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.RemoteAddr = remoteAddr
	router.ServeHTTP(record, request)
	return record
}
