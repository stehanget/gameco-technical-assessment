package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platformauth "gameco-technical-assessment/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func TestAuthRejectsSessionOwnerMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := platformauth.NewJWTManager("test-secret")
	token, err := jwtManager.GenerateAccessToken("1", "jti-1", time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	router := gin.New()
	router.Use(Auth(jwtManager, func(context.Context, string) (platformauth.Actor, error) {
		return platformauth.Actor{UserID: 2, Roles: []string{"seller"}, Permissions: []string{"product.read"}}, nil
	}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for session owner mismatch, got %d: %s", response.Code, response.Body.String())
	}
}
