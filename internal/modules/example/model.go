package example

import (
	"time"

	"github.com/google/uuid"
)

type Example struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	PublicID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Example) TableName() string {
	return "examples"
}
