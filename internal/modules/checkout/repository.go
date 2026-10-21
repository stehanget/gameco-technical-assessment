package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	catalogconstants "gameco-technical-assessment/internal/constants/catalog"
	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errIdempotencyConflict = errors.New("checkout idempotency key has a different request hash")
	errIdempotencyPending  = errors.New("checkout idempotency result is unavailable")
	errEmptyCart           = errors.New("checkout cart is empty")
	errUnavailableCart     = errors.New("checkout cart contains unavailable items")
	errSelfPurchase        = errors.New("cannot purchase own product")
	errInsufficientStock   = errors.New("checkout stock changed during reservation")
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
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("buyer_id = ? AND idempotency_key = ?", buyerID, key).First(&idem)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			idem = IdempotencyKey{BuyerID: buyerID, IdempotencyKey: key, RequestHash: requestHash}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&idem).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
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
		cartLock := ""
		if tx.Dialector.Name() == "postgres" {
			cartLock = " FOR UPDATE"
		}
		if err := tx.Raw("SELECT id FROM carts WHERE buyer_id = ? AND status = 'active'"+cartLock, buyerID).Scan(&cartID).Error; err != nil {
			return err
		}
		if cartID == 0 {
			return errEmptyCart
		}
		var lines []checkoutLine
		productLock := ""
		if tx.Dialector.Name() == "postgres" {
			productLock = " FOR UPDATE OF p"
		}
		if err := tx.Raw(`
			SELECT p.id AS product_id, p.public_id AS product_public_id, p.seller_id,
			       p.name AS product_name, p.price_amount AS unit_price_amount,
			       ci.quantity, p.stock_quantity, p.status, p.deleted_at
			FROM cart_items ci
			JOIN products p ON p.id = ci.product_id
			WHERE ci.cart_id = ?
			ORDER BY p.id`+productLock, cartID).Scan(&lines).Error; err != nil {
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
		response, err = loadOrder(tx, order.ID)
		return err
	})
	return response, err
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
	return OrderResponse{PublicID: order.PublicID, Status: order.Status, TotalAmount: order.TotalAmount, PaymentReference: order.PaymentReference, Items: items, CreatedAt: timestamp.New(order.CreatedAt)}, nil
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
