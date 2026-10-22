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
	PaymentStatus    string              `json:"payment_status" binding:"required"`
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

type SellerOrderListQuery struct {
	Query         string `form:"q"`
	Status        string `form:"status"`
	PaymentStatus string `form:"payment_status"`
	SortBy        string `form:"sort_by"`
	SortOrder     string `form:"sort_order"`
	Cursor        string `form:"cursor"`
	PerPage       int    `form:"per_page"`
}

type SellerOrderResponse struct {
	PublicID          uuid.UUID                 `json:"public_id" binding:"required"`
	Status            string                    `json:"status" binding:"required"`
	SellerTotalAmount int64                     `json:"seller_total_amount" binding:"required"`
	PaymentStatus     string                    `json:"payment_status" binding:"required"`
	Items             []SellerOrderItemResponse `json:"items" binding:"required"`
	CreatedAt         timestamp.Timestamp       `json:"created_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
}

type SellerOrderItemResponse struct {
	ProductPublicID uuid.UUID `json:"product_public_id" binding:"required"`
	ProductName     string    `json:"product_name" binding:"required"`
	UnitPriceAmount int64     `json:"unit_price_amount" binding:"required"`
	Quantity        int       `json:"quantity" binding:"required"`
	SubtotalAmount  int64     `json:"subtotal_amount" binding:"required"`
}

type SellerOrderListMeta struct {
	RequestID  *string `json:"request_id" binding:"required" extensions:"x-nullable"`
	PerPage    int     `json:"per_page" binding:"required"`
	HasMore    bool    `json:"has_more" binding:"required"`
	NextCursor *string `json:"next_cursor" binding:"required" extensions:"x-nullable"`
}

type SellerOrderListResponse struct {
	Success bool                  `json:"success" binding:"required"`
	Data    []SellerOrderResponse `json:"data" binding:"required"`
	Meta    SellerOrderListMeta   `json:"meta" binding:"required"`
}
