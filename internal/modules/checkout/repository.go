package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	catalogconstants "gameco-technical-assessment/internal/constants/catalog"
	"gameco-technical-assessment/internal/platform/audit"
	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errIdempotencyConflict     = errors.New("checkout idempotency key has a different request hash")
	errIdempotencyPending      = errors.New("checkout idempotency result is unavailable")
	errEmptyCart               = errors.New("checkout cart is empty")
	errUnavailableCart         = errors.New("checkout cart contains unavailable items")
	errSelfPurchase            = errors.New("cannot purchase own product")
	errInsufficientStock       = errors.New("checkout stock changed during reservation")
	errWebhookEventConflict    = errors.New("payment webhook event payload conflicts with previous delivery")
	errWebhookInvalidEvent     = errors.New("payment webhook event type is invalid")
	errWebhookPaymentNotFound  = errors.New("payment webhook payment reference was not found")
	errWebhookAmountMismatch   = errors.New("payment webhook amount does not match payment")
	errWebhookMalformedPayload = errors.New("payment webhook payload is malformed")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func RequestHash() string {
	sum := sha256.Sum256([]byte("checkout:v1:active-cart"))
	return hex.EncodeToString(sum[:])
}

type checkoutLine struct {
	ProductID       uint
	ProductPublicID uuid.UUID
	SellerID        uint
	ProductName     string
	UnitPriceAmount int64
	Quantity        int
	StockQuantity   int
	Status          string
	DeletedAt       *time.Time
}

func (r *Repository) Checkout(ctx context.Context, buyerID uint, key, requestHash string) (OrderResponse, error) {
	var response OrderResponse
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var idem IdempotencyKey
		result := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
			Where("buyer_id = ? AND idempotency_key = ?", buyerID, key).First(&idem)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			idem = IdempotencyKey{BuyerID: buyerID, IdempotencyKey: key, RequestHash: requestHash}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&idem).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
				Where("buyer_id = ? AND idempotency_key = ?", buyerID, key).First(&idem).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		}
		if idem.RequestHash != requestHash {
			return errIdempotencyConflict
		}
		if idem.OrderID != nil {
			var loadErr error
			response, loadErr = loadOrder(tx, *idem.OrderID)
			return loadErr
		}
		if idem.ResponseStatus != nil {
			return errIdempotencyPending
		}

		var cartID uint
		cartQuery := tx.Table("carts").
			Select("id").
			Where("buyer_id = ? AND status = 'active'", buyerID)
		cartQuery = cartQuery.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
		if err := cartQuery.Scan(&cartID).Error; err != nil {
			return err
		}
		if cartID == 0 {
			return errEmptyCart
		}
		var lines []checkoutLine
		productQuery := tx.Table("cart_items ci").
			Select(`p.id AS product_id, p.public_id AS product_public_id, p.seller_id,
				p.name AS product_name, p.price_amount AS unit_price_amount,
				ci.quantity, p.stock_quantity, p.status, p.deleted_at`).
			Joins("JOIN products p ON p.id = ci.product_id").
			Where("ci.cart_id = ?", cartID).
			Order("p.id")
		productQuery = productQuery.Clauses(clause.Locking{
			Strength: clause.LockingStrengthUpdate,
			Table:    clause.Table{Name: "p"},
		})
		if err := productQuery.Scan(&lines).Error; err != nil {
			return err
		}
		if len(lines) == 0 {
			return errEmptyCart
		}
		for _, line := range lines {
			if line.DeletedAt != nil || line.Status != catalogconstants.ProductStatusActive || line.Quantity > line.StockQuantity {
				return errUnavailableCart
			}
			if line.SellerID == buyerID {
				return errSelfPurchase
			}
			if line.Quantity <= 0 {
				return errUnavailableCart
			}
		}

		publicID, err := identifier.NewPublicID()
		if err != nil {
			return err
		}
		paymentReference := "pay_" + uuid.NewString()
		var total int64
		for _, line := range lines {
			total += line.UnitPriceAmount * int64(line.Quantity)
		}
		order := Order{PublicID: publicID, BuyerID: buyerID, Status: "pending_payment", TotalAmount: total, PaymentReference: paymentReference}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		paymentPublicID, err := identifier.NewPublicID()
		if err != nil {
			return err
		}
		payment := Payment{PublicID: paymentPublicID, OrderID: order.ID, PaymentReference: paymentReference, Provider: "simulated", Status: "pending", Amount: total}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		for _, line := range lines {
			result := tx.Model(&struct{}{}).Table("products").Where("id = ? AND stock_quantity >= ?", line.ProductID, line.Quantity).
				UpdateColumn("stock_quantity", gorm.Expr("stock_quantity - ?", line.Quantity))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errInsufficientStock
			}
			item := OrderItem{OrderID: order.ID, ProductID: line.ProductID, SellerID: line.SellerID, ProductName: line.ProductName, UnitPriceAmount: line.UnitPriceAmount, Quantity: line.Quantity, SubtotalAmount: line.UnitPriceAmount * int64(line.Quantity)}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			if err := tx.Create(&StockMovement{ProductID: line.ProductID, OrderID: order.ID, OrderItemID: item.ID, ActorUserID: buyerID, MovementType: "checkout", QuantityDelta: -line.Quantity, StockBefore: line.StockQuantity, StockAfter: line.StockQuantity - line.Quantity, Reason: "checkout stock reservation"}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("cart_id = ?", cartID).Delete(&cartItemRow{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&IdempotencyKey{}).Where("id = ?", idem.ID).Updates(map[string]any{"order_id": order.ID, "response_status": 201}).Error; err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, audit.Event{ActorUserID: &buyerID, EventType: "checkout.succeeded", AggregateType: "order", AggregateID: order.PublicID.String(), Metadata: map[string]any{"status": order.Status, "total_amount": order.TotalAmount}}); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, audit.Event{ActorUserID: &buyerID, EventType: "payment.created", AggregateType: "payment", AggregateID: payment.PublicID.String(), Metadata: map[string]any{"provider": payment.Provider, "status": payment.Status, "amount": payment.Amount}}); err != nil {
			return err
		}
		response, err = loadOrder(tx, order.ID)
		return err
	})
	return response, err
}

func (r *Repository) FindOrderByPublicID(ctx context.Context, buyerID uint, publicID uuid.UUID) (OrderResponse, error) {
	var order Order
	if err := r.db.WithContext(ctx).Where("buyer_id = ? AND public_id = ?", buyerID, publicID).First(&order).Error; err != nil {
		return OrderResponse{}, err
	}
	return loadOrder(r.db.WithContext(ctx), order.ID)
}

type orderListParams struct {
	Query         string
	Status        string
	PaymentStatus string
	SortBy        string
	SortOrder     string
	PerPage       int
	Cursor        *cursorPosition
}

type orderSummary struct {
	OrderID          uint      `gorm:"column:order_id"`
	PublicID         uuid.UUID `gorm:"column:public_id"`
	Status           string    `gorm:"column:status"`
	TotalAmount      int64     `gorm:"column:total_amount"`
	PaymentReference string    `gorm:"column:payment_reference"`
	PaymentStatus    string    `gorm:"column:payment_status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

type orderItemListView struct {
	OrderID         uint      `gorm:"column:order_id"`
	ProductPublicID uuid.UUID `gorm:"column:product_public_id"`
	ProductName     string    `gorm:"column:product_name"`
	UnitPriceAmount int64     `gorm:"column:unit_price_amount"`
	Quantity        int       `gorm:"column:quantity"`
	SubtotalAmount  int64     `gorm:"column:subtotal_amount"`
}

func (r *Repository) ListOrdersByBuyer(ctx context.Context, buyerID uint, params orderListParams) ([]orderSummary, []orderItemListView, bool, error) {
	base := r.db.WithContext(ctx).Table("orders o").
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN payments pay ON pay.order_id = o.id").
		Where("o.buyer_id = ?", buyerID)
	if params.Query != "" {
		pattern := "%" + escapeLike(strings.ToLower(params.Query)) + "%"
		base = base.Where(`
			(
				LOWER(oi.product_name) LIKE ? ESCAPE '\\'
				OR CAST(o.public_id AS TEXT) LIKE ?
			)
		`, pattern, pattern)
	}
	if params.Status != "" {
		base = base.Where("o.status = ?", params.Status)
	}
	if params.PaymentStatus != "" {
		base = base.Where("pay.status = ?", params.PaymentStatus)
	}
	if params.Cursor != nil {
		condition, args, err := orderCursorCondition(params.SortBy, params.SortOrder, *params.Cursor)
		if err != nil {
			return nil, nil, false, err
		}
		base = base.Where(condition, args...)
	}
	sortColumn := "o.created_at"
	if params.SortBy == "total_amount" {
		sortColumn = "o.total_amount"
	}
	var orders []orderSummary
	if err := base.Select(`
			o.id AS order_id,
			o.public_id,
			o.status,
			o.total_amount,
			o.payment_reference,
			pay.status AS payment_status,
			o.created_at
		`).
		Group("o.id, o.public_id, o.status, o.total_amount, o.payment_reference, pay.status, o.created_at").
		Order(fmt.Sprintf("%s %s, o.public_id ASC", sortColumn, params.SortOrder)).
		Limit(params.PerPage + 1).Find(&orders).Error; err != nil {
		return nil, nil, false, err
	}
	hasMore := len(orders) > params.PerPage
	if hasMore {
		orders = orders[:params.PerPage]
	}
	if len(orders) == 0 {
		return orders, []orderItemListView{}, false, nil
	}
	orderIDs := make([]uint, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.OrderID)
	}
	var items []orderItemListView
	if err := r.db.WithContext(ctx).Table("order_items oi").
		Select(`
			oi.order_id,
			p.public_id AS product_public_id,
			oi.product_name,
			oi.unit_price_amount,
			oi.quantity,
			oi.subtotal_amount
		`).
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("oi.order_id IN ?", orderIDs).
		Order("oi.order_id, oi.id").Find(&items).Error; err != nil {
		return nil, nil, false, err
	}
	return orders, items, hasMore, nil
}

func orderCursorCondition(sortBy, sortOrder string, cursor cursorPosition) (string, []any, error) {
	column := "o.created_at"
	var value any
	switch sortBy {
	case "created_at":
		parsed, err := timestamp.Parse(cursor.Value)
		if err != nil {
			return "", nil, err
		}
		value = parsed.Time
	case "total_amount":
		parsed, err := strconv.ParseInt(cursor.Value, 10, 64)
		if err != nil {
			return "", nil, err
		}
		column = "o.total_amount"
		value = parsed
	default:
		return "", nil, errors.New("unsupported order sort")
	}
	operator := ">"
	if sortOrder == "desc" {
		operator = "<"
	}
	return fmt.Sprintf("(%s %s ? OR (%s = ? AND o.public_id > ?))", column, operator, column), []any{value, value, cursor.Tie}, nil
}

type sellerOrderListParams struct {
	Query         string
	Status        string
	PaymentStatus string
	SortBy        string
	SortOrder     string
	PerPage       int
	Cursor        *cursorPosition
}

type sellerOrderSummary struct {
	OrderID           uint      `gorm:"column:order_id"`
	PublicID          uuid.UUID `gorm:"column:public_id"`
	Status            string    `gorm:"column:status"`
	PaymentStatus     string    `gorm:"column:payment_status"`
	SellerTotalAmount int64     `gorm:"column:seller_total_amount"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

type sellerOrderItemView struct {
	OrderID         uint      `gorm:"column:order_id"`
	ProductPublicID uuid.UUID `gorm:"column:product_public_id"`
	ProductName     string    `gorm:"column:product_name"`
	UnitPriceAmount int64     `gorm:"column:unit_price_amount"`
	Quantity        int       `gorm:"column:quantity"`
	SubtotalAmount  int64     `gorm:"column:subtotal_amount"`
}

func (r *Repository) FindOrderBySellerPublicID(ctx context.Context, sellerID uint, publicID uuid.UUID) (sellerOrderSummary, []sellerOrderItemView, error) {
	var order sellerOrderSummary
	if err := r.db.WithContext(ctx).Table("orders o").
		Joins("JOIN order_items oi ON oi.order_id = o.id AND oi.seller_id = ?", sellerID).
		Joins("JOIN payments pay ON pay.order_id = o.id").
		Where("o.public_id = ?", publicID).
		Select(`
			o.id AS order_id,
			o.public_id,
			o.status,
			pay.status AS payment_status,
			SUM(oi.subtotal_amount) AS seller_total_amount,
			o.created_at
		`).
		Group("o.id, o.public_id, o.status, pay.status, o.created_at").
		Limit(1).Find(&order).Error; err != nil {
		return sellerOrderSummary{}, nil, err
	}
	if order.OrderID == 0 {
		return sellerOrderSummary{}, nil, gorm.ErrRecordNotFound
	}
	var items []sellerOrderItemView
	if err := r.db.WithContext(ctx).Table("order_items oi").
		Select(`
			oi.order_id,
			p.public_id AS product_public_id,
			oi.product_name,
			oi.unit_price_amount,
			oi.quantity,
			oi.subtotal_amount
		`).
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("oi.seller_id = ? AND oi.order_id = ?", sellerID, order.OrderID).
		Order("oi.id").Find(&items).Error; err != nil {
		return sellerOrderSummary{}, nil, err
	}
	return order, items, nil
}

func (r *Repository) ListOrdersBySeller(ctx context.Context, sellerID uint, params sellerOrderListParams) ([]sellerOrderSummary, []sellerOrderItemView, bool, error) {
	base := r.db.WithContext(ctx).Table("orders o").
		Joins("JOIN order_items oi ON oi.order_id = o.id AND oi.seller_id = ?", sellerID).
		Joins("JOIN payments pay ON pay.order_id = o.id")
	if params.Query != "" {
		pattern := "%" + escapeLike(strings.ToLower(params.Query)) + "%"
		base = base.Where(`
			(
				LOWER(oi.product_name) LIKE ? ESCAPE '\\'
				OR CAST(o.public_id AS TEXT) LIKE ?
			)
		`, pattern, pattern)
	}
	if params.Status != "" {
		base = base.Where("o.status = ?", params.Status)
	}
	if params.PaymentStatus != "" {
		base = base.Where("pay.status = ?", params.PaymentStatus)
	}
	if params.Cursor != nil {
		condition, args, err := sellerOrderCursorCondition(params.SortBy, params.SortOrder, *params.Cursor)
		if err != nil {
			return nil, nil, false, err
		}
		if params.SortBy == "seller_total" {
			base = base.Having(condition, args...)
		} else {
			base = base.Where(condition, args...)
		}
	}

	sortColumn := "o.created_at"
	if params.SortBy == "seller_total" {
		sortColumn = "SUM(oi.subtotal_amount)"
	}
	var orders []sellerOrderSummary
	if err := base.Select(`
			o.id AS order_id,
			o.public_id,
			o.status,
			pay.status AS payment_status,
			SUM(oi.subtotal_amount) AS seller_total_amount,
			o.created_at
		`).
		Group("o.id, o.public_id, o.status, pay.status, o.created_at").
		Order(fmt.Sprintf("%s %s, o.public_id ASC", sortColumn, params.SortOrder)).
		Limit(params.PerPage + 1).Find(&orders).Error; err != nil {
		return nil, nil, false, err
	}
	hasMore := len(orders) > params.PerPage
	if hasMore {
		orders = orders[:params.PerPage]
	}
	if len(orders) == 0 {
		return orders, []sellerOrderItemView{}, false, nil
	}
	orderIDs := make([]uint, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.OrderID)
	}
	var items []sellerOrderItemView
	if err := r.db.WithContext(ctx).Table("order_items oi").
		Select(`
			oi.order_id,
			p.public_id AS product_public_id,
			oi.product_name,
			oi.unit_price_amount,
			oi.quantity,
			oi.subtotal_amount
		`).
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("oi.seller_id = ? AND oi.order_id IN ?", sellerID, orderIDs).
		Order("oi.order_id, oi.id").Find(&items).Error; err != nil {
		return nil, nil, false, err
	}
	return orders, items, hasMore, nil
}

func sellerOrderCursorCondition(sortBy, sortOrder string, cursor cursorPosition) (string, []any, error) {
	column := "o.created_at"
	var value any
	switch sortBy {
	case "created_at":
		parsed, err := timestamp.Parse(cursor.Value)
		if err != nil {
			return "", nil, err
		}
		value = parsed.Time
	case "seller_total":
		parsed, err := strconv.ParseInt(cursor.Value, 10, 64)
		if err != nil {
			return "", nil, err
		}
		column = "SUM(oi.subtotal_amount)"
		value = parsed
	default:
		return "", nil, errors.New("unsupported seller order sort")
	}
	operator := ">"
	if sortOrder == "desc" {
		operator = "<"
	}
	return fmt.Sprintf("(%s %s ? OR (%s = ? AND o.public_id > ?))", column, operator, column), []any{value, value, cursor.Tie}, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\\`, `\\\\`)
	value = strings.ReplaceAll(value, "%", `\\%`)
	return strings.ReplaceAll(value, "_", `\\_`)
}

func (r *Repository) ProcessWebhook(ctx context.Context, event PaymentWebhookRequest, payloadHash string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inbox := PaymentWebhookEvent{
			Provider:         simulatedPaymentProvider,
			EventID:          event.EventID,
			PaymentReference: event.PaymentReference,
			EventType:        event.EventType,
			PayloadHash:      payloadHash,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&inbox).Error; err != nil {
			return err
		}
		var stored PaymentWebhookEvent
		if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
			Where("provider = ? AND event_id = ?", simulatedPaymentProvider, event.EventID).
			First(&stored).Error; err != nil {
			return err
		}
		if stored.PayloadHash != payloadHash {
			return errWebhookEventConflict
		}
		if stored.ProcessedAt != nil {
			return nil
		}

		var payment Payment
		paymentQuery := tx.Where("payment_reference = ?", event.PaymentReference)
		paymentQuery = paymentQuery.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
		if err := paymentQuery.First(&payment).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errWebhookPaymentNotFound
		} else if err != nil {
			return err
		}
		if payment.Amount != event.Amount {
			return errWebhookAmountMismatch
		}

		var order Order
		orderQuery := tx.Where("id = ?", payment.OrderID)
		orderQuery = orderQuery.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
		if err := orderQuery.First(&order).Error; err != nil {
			return err
		}
		switch payment.Status {
		case "pending":
			switch event.EventType {
			case "payment.paid":
				if err := tx.Model(&payment).Updates(map[string]any{"status": "paid", "paid_at": now}).Error; err != nil {
					return err
				}
				if err := tx.Model(&order).Updates(map[string]any{"status": "paid", "paid_at": now}).Error; err != nil {
					return err
				}
				if err := audit.Write(ctx, tx, audit.Event{EventType: "order.status_changed", AggregateType: "order", AggregateID: order.PublicID.String(), Metadata: map[string]any{"from": "pending_payment", "to": "paid", "source": "payment_webhook"}}); err != nil {
					return err
				}
			case "payment.failed", "payment.expired":
				updates := map[string]any{"status": map[string]string{"payment.failed": "failed", "payment.expired": "expired"}[event.EventType]}
				if event.FailureReason != "" {
					updates["failure_reason"] = event.FailureReason
				}
				if event.EventType == "payment.failed" {
					updates["failed_at"] = now
				}
				if err := tx.Model(&payment).Updates(updates).Error; err != nil {
					return err
				}
				orderStatus := "payment_failed"
				if event.EventType == "payment.expired" {
					if err := restoreOrderStock(tx, order.ID); err != nil {
						return err
					}
					orderStatus = "cancelled"
				}
				if err := tx.Model(&order).Update("status", orderStatus).Error; err != nil {
					return err
				}
				if err := audit.Write(ctx, tx, audit.Event{EventType: "order.status_changed", AggregateType: "order", AggregateID: order.PublicID.String(), Metadata: map[string]any{"from": "pending_payment", "to": orderStatus, "source": "payment_webhook"}}); err != nil {
					return err
				}
			default:
				return errWebhookInvalidEvent
			}
		case "paid", "failed", "expired":
			// Terminal payment states are safe no-ops for a new event.
		default:
			return errWebhookInvalidEvent
		}
		return tx.Model(&stored).Update("processed_at", now).Error
	})
}

func (r *Repository) CleanupProcessedWebhookEvents(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("processed_at IS NOT NULL AND processed_at < ?", cutoff).
		Delete(&PaymentWebhookEvent{})
	return result.RowsAffected, result.Error
}

func (r *Repository) CleanupCompletedIdempotencyKeys(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("order_id IS NOT NULL AND updated_at < ?", cutoff).
		Delete(&IdempotencyKey{})
	return result.RowsAffected, result.Error
}

func restoreOrderStock(tx *gorm.DB, orderID uint) error {
	var items []struct {
		ID          uint
		ProductID   uint
		Quantity    int
		StockBefore int
	}
	query := tx.Table("order_items oi").
		Select("oi.id, oi.product_id, oi.quantity, p.stock_quantity AS stock_before").
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("oi.order_id = ?", orderID).
		Order("oi.product_id, oi.id")
	query = query.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
	if err := query.Scan(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if err := tx.Model(&struct{}{}).Table("products").
			Where("id = ?", item.ProductID).
			UpdateColumn("stock_quantity", gorm.Expr("stock_quantity + ?", item.Quantity)).Error; err != nil {
			return err
		}
		if err := tx.Table("stock_movements").Create(map[string]any{
			"product_id": item.ProductID, "order_id": orderID, "order_item_id": item.ID,
			"movement_type": "cancellation", "quantity_delta": item.Quantity,
			"stock_before": item.StockBefore, "stock_after": item.StockBefore + item.Quantity,
			"reason": "payment expired stock restoration",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadOrder(tx *gorm.DB, orderID uint) (OrderResponse, error) {
	var order Order
	if err := tx.First(&order, orderID).Error; err != nil {
		return OrderResponse{}, err
	}
	rows, err := loadOrderItems(tx, order.ID)
	if err != nil {
		return OrderResponse{}, err
	}
	items := make([]OrderItemResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, OrderItemResponse(row))
	}
	var payment Payment
	if err := tx.Where("order_id = ?", order.ID).First(&payment).Error; err != nil {
		return OrderResponse{}, err
	}
	return OrderResponse{PublicID: order.PublicID, Status: order.Status, TotalAmount: order.TotalAmount, PaymentReference: order.PaymentReference, PaymentStatus: payment.Status, Items: items, CreatedAt: timestamp.New(order.CreatedAt)}, nil
}

type orderItemView struct {
	ProductPublicID uuid.UUID `gorm:"column:product_public_id"`
	ProductName     string
	UnitPriceAmount int64
	Quantity        int
	SubtotalAmount  int64
}

func loadOrderItems(tx *gorm.DB, orderID uint) ([]orderItemView, error) {
	var rows []orderItemView
	query := tx.Table("order_items oi").
		Select(`p.public_id AS product_public_id,
			oi.product_name,
			oi.unit_price_amount,
			oi.quantity,
			oi.subtotal_amount`).
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("oi.order_id = ?", orderID).
		Order("oi.id")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
