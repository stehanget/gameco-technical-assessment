package cart

import (
	"context"
	"errors"
	"time"

	cartconstants "gameco-technical-assessment/internal/constants/cart"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"
)

var errInsufficientStock = errors.New("insufficient product stock")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type itemView struct {
	ProductPublicID string     `gorm:"column:product_public_id"`
	Name            string     `gorm:"column:name"`
	PriceAmount     int64      `gorm:"column:price_amount"`
	Quantity        int        `gorm:"column:quantity"`
	StockQuantity   int        `gorm:"column:stock_quantity"`
	Status          string     `gorm:"column:status"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

type cartView struct {
	Cart
	Rows []itemView
}

func (r *Repository) FindOrCreateActiveCart(ctx context.Context, buyerID uint, publicID uuid.UUID) (Cart, error) {
	var cart Cart
	err := r.db.WithContext(ctx).Where("buyer_id = ? AND status = ?", buyerID, cartconstants.StatusActive).First(&cart).Error
	if err == nil {
		return cart, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Cart{}, err
	}
	cart = Cart{PublicID: publicID, BuyerID: buyerID, Status: cartconstants.StatusActive}
	if err := r.db.WithContext(ctx).Create(&cart).Error; err != nil {
		return Cart{}, err
	}
	return cart, nil
}

func (r *Repository) ReadActiveCart(ctx context.Context, buyerID uint) (cartView, error) {
	var result cartView
	if err := r.db.WithContext(ctx).Where("buyer_id = ? AND status = ?", buyerID, cartconstants.StatusActive).First(&result.Cart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		return result, err
	}
	return r.loadItems(ctx, result)
}

func (r *Repository) SetItem(ctx context.Context, buyerID uint, productPublicID string, quantity int, cartPublicID uuid.UUID) (cartView, error) {
	var result cartView
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cart Cart
		if err := tx.Where("buyer_id = ? AND status = ?", buyerID, cartconstants.StatusActive).First(&cart).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			cart = Cart{PublicID: cartPublicID, BuyerID: buyerID, Status: "active"}
			if err := tx.Create(&cart).Error; err != nil {
				return err
			}
		}

		var product struct {
			ID            uint
			Name          string
			PriceAmount   int64
			StockQuantity int
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("products").
			Select("id, name, price_amount, stock_quantity").
			Where("public_id = ? AND status = ? AND deleted_at IS NULL", productPublicID, "active").
			First(&product).Error; err != nil {
			return err
		}
		if quantity > product.StockQuantity {
			return errInsufficientStock
		}

		var item CartItem
		err := tx.Where("cart_id = ? AND product_id = ?", cart.ID, product.ID).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			item = CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: quantity}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			item.Quantity = quantity
			if err := tx.Save(&item).Error; err != nil {
				return err
			}
		}
		result.Cart = cart
		return nil
	})
	if err != nil {
		return result, err
	}
	return r.loadItems(ctx, result)
}

func (r *Repository) DeleteItem(ctx context.Context, buyerID uint, productPublicID string) (cartView, error) {
	var result cartView
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		db := tx.Model(&CartItem{}).
			Where("cart_id IN (SELECT id FROM carts WHERE buyer_id = ? AND status = ?)", buyerID, cartconstants.StatusActive).
			Where("product_id IN (SELECT id FROM products WHERE public_id = ?)", productPublicID)
		if err := db.Delete(&CartItem{}).Error; err != nil {
			return err
		}
		return tx.Where("buyer_id = ? AND status = ?", buyerID, cartconstants.StatusActive).First(&result.Cart).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	return r.loadItems(ctx, result)
}

func (r *Repository) loadItems(ctx context.Context, result cartView) (cartView, error) {
	err := r.db.WithContext(ctx).Table("cart_items AS ci").
		Select("p.public_id AS product_public_id, p.name, p.price_amount, ci.quantity, p.stock_quantity, p.status, p.deleted_at").
		Joins("JOIN carts AS c ON c.id = ci.cart_id").
		Joins("JOIN products AS p ON p.id = ci.product_id").
		Where("c.id = ?", result.Cart.ID).
		Order("ci.id ASC").Find(&result.Rows).Error
	return result, err
}
