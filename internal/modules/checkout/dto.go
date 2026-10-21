package checkout

import (
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
)

type Response struct {
	Success bool          `json:"success" binding:"required"`
	Data    OrderResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta   `json:"meta" binding:"required"`
}

type OrderResponse struct {
	PublicID         uuid.UUID           `json:"public_id" binding:"required"`
	Status           string              `json:"status" binding:"required"`
	TotalAmount      int64               `json:"total_amount" binding:"required"`
	PaymentReference string              `json:"payment_reference" binding:"required"`
	Items            []OrderItemResponse `json:"items" binding:"required"`
	CreatedAt        timestamp.Timestamp `json:"created_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
}

type OrderItemResponse struct {
	ProductPublicID uuid.UUID `json:"product_public_id" binding:"required"`
	ProductName     string    `json:"product_name" binding:"required"`
	UnitPriceAmount int64     `json:"unit_price_amount" binding:"required"`
	Quantity        int       `json:"quantity" binding:"required"`
	SubtotalAmount  int64     `json:"subtotal_amount" binding:"required"`
}
