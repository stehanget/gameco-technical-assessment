package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManagerGenerateAndParseAccessToken(t *testing.T) {
	manager := NewJWTManager("secret")

	token, err := manager.GenerateAccessToken("42", "session-jti", time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != "42" {
		t.Fatalf("expected user id 42, got %s", claims.UserID)
	}
	if claims.TokenID != "session-jti" {
		t.Fatalf("unexpected token id: %s", claims.TokenID)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected compact JWT with three parts, got %d", len(parts))
	}
	parsed, _, err := new(jwt.Parser).ParseUnverified(token, &AccessTokenClaims{})
	if err != nil {
		t.Fatalf("parse generated token header: %v", err)
	}
	if _, exists := parsed.Header["kid"]; exists {
		t.Fatal("generated token must not contain a kid header")
	}
}

func TestJWTManagerSupportsPreviousSecrets(t *testing.T) {
	oldManager := NewJWTManager("old-secret")
	token, err := oldManager.GenerateAccessToken("42", "old-session", time.Minute)
	if err != nil {
		t.Fatalf("generate old token: %v", err)
	}

	rotatedManager := NewJWTManager("new-secret", "old-secret")
	if _, err := rotatedManager.ParseAccessToken(token); err != nil {
		t.Fatalf("parse token signed by previous secret: %v", err)
	}

	newToken, err := rotatedManager.GenerateAccessToken("42", "new-session", time.Minute)
	if err != nil {
		t.Fatalf("generate new token: %v", err)
	}
	if _, err := rotatedManager.ParseAccessToken(newToken); err != nil {
		t.Fatalf("parse token signed by current secret: %v", err)
	}

	withoutPrevious := NewJWTManager("new-secret")
	if _, err := withoutPrevious.ParseAccessToken(token); err == nil {
		t.Fatal("token signed by removed previous secret must be rejected")
	}
}
