package catalog

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	catalogconstants "gameco-technical-assessment/internal/constants/catalog"
	"gameco-technical-assessment/internal/platform/timestamp"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type listParams struct {
	Query     string
	Category  string
	MinPrice  *int64
	MaxPrice  *int64
	SortBy    string
	SortOrder string
	PerPage   int
	Cursor    *cursorPosition
}

type productView struct {
	PublicID         string    `gorm:"column:public_id"`
	Name             string    `gorm:"column:name"`
	Description      string    `gorm:"column:description"`
	PriceAmount      int64     `gorm:"column:price_amount"`
	StockQuantity    int       `gorm:"column:stock_quantity"`
	CategorySlug     string    `gorm:"column:category_slug"`
	CategoryName     string    `gorm:"column:category_name"`
	CategoryPublicID string    `gorm:"column:category_public_id"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (r *Repository) List(ctx context.Context, params listParams) ([]productView, bool, error) {
	query := r.db.WithContext(ctx).Table("products AS p").
		Select(`
			p.public_id,
			p.name,
			p.description,
			p.price_amount,
			p.stock_quantity,
			c.slug AS category_slug,
			c.name AS category_name,
			c.public_id AS category_public_id,
			p.status,
			p.created_at,
			p.updated_at
		`).
		Joins("JOIN categories AS c ON c.id = p.category_id").
		Where(`
			p.status = ?
			AND p.deleted_at IS NULL
		`, catalogconstants.ProductStatusActive)

	if params.Query != "" {
		pattern := "%" + escapeLike(strings.ToLower(params.Query)) + "%"
		query = query.Where(`
			(
				LOWER(p.name) LIKE ? ESCAPE '\'
				OR LOWER(p.description) LIKE ? ESCAPE '\'
			)
		`, pattern, pattern)
	}
	if params.Category != "" {
		query = query.Where("c.slug = ?", params.Category)
	}
	if params.MinPrice != nil {
		query = query.Where("p.price_amount >= ?", *params.MinPrice)
	}
	if params.MaxPrice != nil {
		query = query.Where("p.price_amount <= ?", *params.MaxPrice)
	}
	if params.Cursor != nil {
		condition, args, err := cursorCondition(params.SortBy, params.SortOrder, *params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query = query.Where(condition, args...)
	}

	column := sortColumn(params.SortBy)
	order := fmt.Sprintf("%s %s, p.public_id ASC", column, params.SortOrder)
	var rows []productView
	if err := query.Order(order).Limit(params.PerPage + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(rows) > params.PerPage
	if hasMore {
		rows = rows[:params.PerPage]
	}
	return rows, hasMore, nil
}

func (r *Repository) FindByPublicID(ctx context.Context, publicID string) (productView, error) {
	var row productView
	err := r.db.WithContext(ctx).Table("products AS p").
		Select(`
			p.public_id,
			p.name,
			p.description,
			p.price_amount,
			p.stock_quantity,
			c.slug AS category_slug,
			c.name AS category_name,
			c.public_id AS category_public_id,
			p.status,
			p.created_at,
			p.updated_at
		`).
		Joins("JOIN categories AS c ON c.id = p.category_id").
		Where(`
			p.public_id = ?
			AND p.status = ?
			AND p.deleted_at IS NULL
		`, publicID, catalogconstants.ProductStatusActive).
		First(&row).Error
	return row, err
}

func (r *Repository) ListBySeller(ctx context.Context, sellerID uint, params listParams) ([]productView, bool, error) {
	query := r.db.WithContext(ctx).Table("products AS p").
		Select(`
			p.public_id,
			p.name,
			p.description,
			p.price_amount,
			p.stock_quantity,
			p.status,
			c.public_id AS category_public_id,
			c.slug AS category_slug,
			c.name AS category_name,
			p.created_at,
			p.updated_at
		`).
		Joins("JOIN categories AS c ON c.id = p.category_id").
		Where("p.seller_id = ? AND p.deleted_at IS NULL", sellerID)

	query = applyListFilters(query, params)
	if params.Cursor != nil {
		condition, args, err := cursorCondition(params.SortBy, params.SortOrder, *params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query = query.Where(condition, args...)
	}

	order := fmt.Sprintf("%s %s, p.public_id ASC", sortColumn(params.SortBy), params.SortOrder)
	var rows []productView
	if err := query.Order(order).Limit(params.PerPage + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > params.PerPage
	if hasMore {
		rows = rows[:params.PerPage]
	}
	return rows, hasMore, nil
}

func (r *Repository) FindByPublicIDForSeller(ctx context.Context, sellerID uint, publicID string) (productView, error) {
	var row productView
	err := r.db.WithContext(ctx).Table("products AS p").
		Select(`
			p.public_id,
			p.name,
			p.description,
			p.price_amount,
			p.stock_quantity,
			p.status,
			c.public_id AS category_public_id,
			c.slug AS category_slug,
			c.name AS category_name,
			p.created_at,
			p.updated_at
		`).
		Joins("JOIN categories AS c ON c.id = p.category_id").
		Where("p.public_id = ? AND p.seller_id = ? AND p.deleted_at IS NULL", publicID, sellerID).
		First(&row).Error
	return row, err
}

func (r *Repository) ListCategoryOptions(ctx context.Context) ([]Category, error) {
	var rows []Category
	err := r.db.WithContext(ctx).
		Order("name ASC, public_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) FindCategoryByPublicID(ctx context.Context, publicID string) (Category, error) {
	var category Category
	err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&category).Error
	return category, err
}

func (r *Repository) CreateProduct(ctx context.Context, product *Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *Repository) UpdateProduct(ctx context.Context, sellerID uint, publicID string, values map[string]any) error {
	result := r.db.WithContext(ctx).Model(&Product{}).
		Where("seller_id = ? AND public_id = ? AND deleted_at IS NULL", sellerID, publicID).
		Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func applyListFilters(query *gorm.DB, params listParams) *gorm.DB {
	if params.Query != "" {
		pattern := "%" + escapeLike(strings.ToLower(params.Query)) + "%"
		query = query.Where(`
			(
				LOWER(p.name) LIKE ? ESCAPE '\'
				OR LOWER(p.description) LIKE ? ESCAPE '\'
			)
		`, pattern, pattern)
	}
	if params.Category != "" {
		query = query.Where("c.slug = ?", params.Category)
	}
	if params.MinPrice != nil {
		query = query.Where("p.price_amount >= ?", *params.MinPrice)
	}
	if params.MaxPrice != nil {
		query = query.Where("p.price_amount <= ?", *params.MaxPrice)
	}
	return query
}

func sortColumn(sortBy string) string {
	switch sortBy {
	case "name":
		return "p.name"
	case "price":
		return "p.price_amount"
	default:
		return "p.created_at"
	}
}

func cursorCondition(sortBy, sortOrder string, cursor cursorPosition) (string, []any, error) {
	column := sortColumn(sortBy)
	var value any
	switch sortBy {
	case "price":
		parsed, err := strconv.ParseInt(cursor.Value, 10, 64)
		if err != nil {
			return "", nil, err
		}
		value = parsed
	case "created_at":
		parsed, err := timestamp.Parse(cursor.Value)
		if err != nil {
			return "", nil, err
		}
		value = parsed.Time
	case "name":
		value = cursor.Value
	default:
		return "", nil, errors.New("unsupported catalog sort")
	}

	operator := ">"
	if sortOrder == "desc" {
		operator = "<"
	}
	condition := fmt.Sprintf(`
		(%s %s ? OR (%s = ? AND p.public_id > ?))
	`, column, operator, column)
	return condition, []any{value, value, cursor.Tie}, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	return strings.ReplaceAll(value, "_", `\_`)
}
