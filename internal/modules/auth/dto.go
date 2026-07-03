package auth

import (
	"github.com/google/uuid"

	"gameco-technical-assessment/internal/platform/httpx"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserResponse struct {
	PublicID    uuid.UUID `json:"public_id"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type TokenEnvelope struct {
	Success bool          `json:"success"`
	Data    TokenResponse `json:"data"`
	Meta    *httpx.Meta   `json:"meta,omitempty"`
}

type UserEnvelope struct {
	Success bool         `json:"success"`
	Data    UserResponse `json:"data"`
	Meta    *httpx.Meta  `json:"meta,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type MessageEnvelope struct {
	Success bool            `json:"success"`
	Data    MessageResponse `json:"data"`
	Meta    *httpx.Meta     `json:"meta,omitempty"`
}
