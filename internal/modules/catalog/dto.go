package catalog

import (
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
)

type ListQuery struct {
	Query     string `form:"q"`
	Category  string `form:"category"`
	MinPrice  *int64 `form:"min_price"`
	MaxPrice  *int64 `form:"max_price"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
	Cursor    string `form:"cursor"`
	PerPage   int    `form:"per_page"`
}

type ProductResponse struct {
	PublicID    uuid.UUID           `json:"public_id" binding:"required"`
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description" binding:"required"`
	PriceAmount int64               `json:"price_amount" binding:"required"`
	Stock       int                 `json:"stock_quantity" binding:"required"`
	Category    CategoryResponse    `json:"category" binding:"required"`
	CreatedAt   timestamp.Timestamp `json:"created_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
	UpdatedAt   timestamp.Timestamp `json:"updated_at" binding:"required" swaggertype:"string" format:"date-time" example:"2026-10-21T10:15:30Z"`
}

type CategoryResponse struct {
	Slug string `json:"slug" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type ListMeta struct {
	RequestID  *string `json:"request_id" binding:"required" extensions:"x-nullable"`
	PerPage    int     `json:"per_page" binding:"required"`
	HasMore    bool    `json:"has_more" binding:"required"`
	NextCursor *string `json:"next_cursor" binding:"required" extensions:"x-nullable"`
}

type ListResponse struct {
	Success bool              `json:"success" binding:"required"`
	Data    []ProductResponse `json:"data" binding:"required"`
	Meta    ListMeta          `json:"meta" binding:"required"`
}

type ItemResponse struct {
	Success bool            `json:"success" binding:"required"`
	Data    ProductResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta     `json:"meta" binding:"required"`
}
