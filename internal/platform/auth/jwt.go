package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	current  []byte
	previous [][]byte
}

func NewJWTManager(currentSecret string, previousSecrets ...string) *JWTManager {
	previous := make([][]byte, 0, len(previousSecrets))
	for _, secret := range previousSecrets {
		previous = append(previous, []byte(secret))
	}
	return &JWTManager{current: []byte(currentSecret), previous: previous}
}

func (m *JWTManager) GenerateAccessToken(userID, tokenID string, ttl time.Duration) (string, error) {
	if len(m.current) == 0 {
		return "", fmt.Errorf("current JWT signing key is not configured")
	}

	now := time.Now()
	if tokenID == "" {
		tokenID = uuid.NewString()
	}
	claims := AccessTokenClaims{
		UserID:  userID,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.current)
}

func (m *JWTManager) ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
	keys := make([][]byte, 0, 1+len(m.previous))
	keys = append(keys, m.current)
	keys = append(keys, m.previous...)

	var lastErr error
	for _, secret := range keys {
		claims := &AccessTokenClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return secret, nil
		})
		if err == nil && token.Valid {
			return claims, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, fmt.Errorf("no JWT signing keys are configured")
	}
	return nil, lastErr
}
