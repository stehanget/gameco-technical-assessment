package auth

import "github.com/golang-jwt/jwt/v5"

type AccessTokenClaims struct {
	UserID  string `json:"sub"`
	TokenID string `json:"jti"`
	jwt.RegisteredClaims
}
