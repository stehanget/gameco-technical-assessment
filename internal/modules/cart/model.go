package cart

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	PublicID  uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"public_id"`
	BuyerID   uint       `gorm:"not null;index" json:"buyer_id"`
	Status    string     `gorm:"type:cart_status;not null" json:"status"`
	Items     []CartItem `gorm:"foreignKey:CartID" json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Cart) TableName() string { return "carts" }

type CartItem struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	CartID    uint      `gorm:"not null;index" json:"-"`
	ProductID uint      `gorm:"not null;index" json:"-"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CartItem) TableName() string { return "cart_items" }
