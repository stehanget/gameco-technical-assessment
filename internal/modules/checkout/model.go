package checkout

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID               uint       `gorm:"primaryKey"`
	PublicID         uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	BuyerID          uint       `gorm:"not null;index"`
	Status           string     `gorm:"type:order_status;not null"`
	TotalAmount      int64      `gorm:"not null"`
	PaymentReference string     `gorm:"size:128;uniqueIndex;not null"`
	PaymentExpiresAt *time.Time `json:"-"`
	PaidAt           *time.Time `json:"-"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID              uint   `gorm:"primaryKey"`
	OrderID         uint   `gorm:"not null;index"`
	ProductID       uint   `gorm:"not null;index"`
	SellerID        uint   `gorm:"not null;index"`
	ProductName     string `gorm:"size:255;not null"`
	UnitPriceAmount int64  `gorm:"not null"`
	Quantity        int    `gorm:"not null"`
	SubtotalAmount  int64  `gorm:"not null"`
	CreatedAt       time.Time
}

func (OrderItem) TableName() string { return "order_items" }

type Payment struct {
	ID               uint       `gorm:"primaryKey"`
	PublicID         uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	OrderID          uint       `gorm:"not null;uniqueIndex"`
	PaymentReference string     `gorm:"size:128;uniqueIndex;not null"`
	Provider         string     `gorm:"size:64;not null"`
	Status           string     `gorm:"type:payment_status;not null"`
	Amount           int64      `gorm:"not null"`
	ExpiresAt        *time.Time `json:"-"`
	PaidAt           *time.Time `json:"-"`
	FailedAt         *time.Time `json:"-"`
	FailureReason    *string    `json:"-"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Payment) TableName() string { return "payments" }

type PaymentWebhookEvent struct {
	ID               uint       `gorm:"primaryKey"`
	Provider         string     `gorm:"size:64;not null;uniqueIndex:uq_payment_webhook_events_provider_event,priority:1"`
	EventID          string     `gorm:"size:128;not null;uniqueIndex:uq_payment_webhook_events_provider_event,priority:2"`
	PaymentReference string     `gorm:"size:128;not null;index"`
	EventType        string     `gorm:"size:64;not null"`
	PayloadHash      string     `gorm:"size:64;not null"`
	ProcessedAt      *time.Time `json:"-"`
	CreatedAt        time.Time
}

func (PaymentWebhookEvent) TableName() string { return "payment_webhook_events" }

type IdempotencyKey struct {
	ID             uint   `gorm:"primaryKey"`
	BuyerID        uint   `gorm:"not null"`
	IdempotencyKey string `gorm:"size:255;not null"`
	RequestHash    string `gorm:"size:64;not null"`
	OrderID        *uint  `gorm:"index"`
	ResponseStatus *int   `gorm:"column:response_status"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (IdempotencyKey) TableName() string { return "checkout_idempotency_keys" }

type StockMovement struct {
	ProductID     uint   `gorm:"not null"`
	OrderID       uint   `gorm:"not null"`
	OrderItemID   uint   `gorm:"not null"`
	ActorUserID   uint   `gorm:"not null"`
	MovementType  string `gorm:"type:stock_movement_type;not null"`
	QuantityDelta int
	StockBefore   int
	StockAfter    int
	Reason        string
}

func (StockMovement) TableName() string { return "stock_movements" }

type cartItemRow struct{ ID uint }

func (cartItemRow) TableName() string { return "cart_items" }
