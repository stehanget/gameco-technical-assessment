package cart

import (
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
)

type SetItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=999"`
}

type ItemResponse struct {
	ProductPublicID   uuid.UUID `json:"product_public_id" binding:"required"`
	Name              string    `json:"name" binding:"required"`
	PriceAmount       int64     `json:"price_amount" binding:"required"`
	Quantity          int       `json:"quantity" binding:"required"`
	SubtotalAmount    int64     `json:"subtotal_amount" binding:"required"`
	StockQuantity     int       `json:"stock_quantity" binding:"required"`
	Availability      string    `json:"availability" binding:"required"`
	UnavailableReason *string   `json:"unavailable_reason" binding:"required" extensions:"x-nullable"`
}

type Response struct {
	Success bool         `json:"success" binding:"required"`
	Data    CartResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta  `json:"meta" binding:"required"`
}

type CartResponse struct {
	PublicID         uuid.UUID           `json:"public_id" binding:"required"`
	Status           string              `json:"status" binding:"required"`
	Items            []ItemResponse      `json:"items" binding:"required"`
	UnavailableItems []ItemResponse      `json:"unavailable_items" binding:"required"`
	TotalAmount      int64               `json:"total_amount" binding:"required"`
	CreatedAt        timestamp.Timestamp `json:"created_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
	UpdatedAt        timestamp.Timestamp `json:"updated_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
}
