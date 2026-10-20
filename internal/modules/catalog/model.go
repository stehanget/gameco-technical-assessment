package catalog

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	PublicID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_id"`
	Name      string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Slug      string    `gorm:"size:160;uniqueIndex;not null" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Category) TableName() string { return "categories" }

type Product struct {
	ID            uint           `gorm:"primaryKey" json:"-"`
	PublicID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"public_id"`
	SellerID      uint           `gorm:"not null;index" json:"seller_id"`
	CategoryID    uint           `gorm:"not null;index" json:"category_id"`
	Name          string         `gorm:"size:255;not null" json:"name"`
	Description   string         `gorm:"type:text;not null" json:"description"`
	PriceAmount   int64          `gorm:"not null" json:"price_amount"`
	StockQuantity int            `gorm:"not null;default:0" json:"stock_quantity"`
	Status        string         `gorm:"type:product_status;not null" json:"status"`
	DeletedAt     gorm.DeletedAt `json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (Product) TableName() string { return "products" }
