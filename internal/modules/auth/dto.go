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
	PublicID    uuid.UUID `json:"public_id" binding:"required"`
	Email       string    `json:"email" binding:"required"`
	Roles       []string  `json:"roles" binding:"required"`
	Permissions []string  `json:"permissions" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token" binding:"required"`
	RefreshToken string       `json:"refresh_token" binding:"required"`
	TokenType    string       `json:"token_type" binding:"required"`
	ExpiresIn    int64        `json:"expires_in" binding:"required"`
	User         UserResponse `json:"user" binding:"required"`
}

type TokenEnvelope struct {
	Success bool          `json:"success" binding:"required"`
	Data    TokenResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta   `json:"meta" binding:"required"`
}

type UserEnvelope struct {
	Success bool         `json:"success" binding:"required"`
	Data    UserResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta  `json:"meta" binding:"required"`
}

type MessageResponse struct {
	Message string `json:"message" binding:"required"`
}

type MessageEnvelope struct {
	Success bool            `json:"success" binding:"required"`
	Data    MessageResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta     `json:"meta" binding:"required"`
}
