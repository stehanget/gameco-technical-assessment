package auth

import (
	"testing"
	"time"
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
}
