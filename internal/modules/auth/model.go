package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"-"`
	PublicID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_id"`
	Email        string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:255;not null" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:128;uniqueIndex;not null" json:"code"`
	Description string    `gorm:"size:255;not null" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Permission) TableName() string { return "permissions" }

type RolePermission struct {
	RoleID       uint      `gorm:"primaryKey" json:"role_id"`
	PermissionID uint      `gorm:"primaryKey" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (RolePermission) TableName() string { return "role_permissions" }

type UserRole struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserRole) TableName() string { return "user_roles" }

type AuthSession struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	JTI         string     `gorm:"column:jti;size:64;uniqueIndex;not null" json:"-"`
	RefreshHash string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	Roles       []string   `gorm:"serializer:json;not null" json:"roles"`
	Permissions []string   `gorm:"serializer:json;not null" json:"permissions"`
	UserAgent   string     `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress   string     `gorm:"size:64" json:"ip_address,omitempty"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AuthSession) TableName() string { return "auth_sessions" }

type Access struct {
	UserID      uint
	Roles       []string
	Permissions []string
}
