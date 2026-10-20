package seeds

import (
	"errors"

	"gameco-technical-assessment/internal/constants/catalog"
	moduleauth "gameco-technical-assessment/internal/modules/auth"
	modulecatalog "gameco-technical-assessment/internal/modules/catalog"
	"gameco-technical-assessment/internal/platform/identifier"

	"gorm.io/gorm"
)

type productSeedRecord struct {
	Name         string
	CategorySlug string
	Description  string
	PriceAmount  int64
	Stock        int
}

type ProductSeeder struct{}

var productSeedRecords = []productSeedRecord{
	{
		Name:         "Gaming GPU",
		CategorySlug: "pc-components",
		Description:  "High-performance graphics card for modern gaming builds.",
		PriceAmount:  7999000000,
		Stock:        8,
	},
	{
		Name:         "Mechanical Gaming Keyboard",
		CategorySlug: "pc-components",
		Description:  "Mechanical keyboard with responsive switches and gaming-focused controls.",
		PriceAmount:  14990000,
		Stock:        24,
	},
	{
		Name:         "Wireless Gaming Headset",
		CategorySlug: "gaming-peripherals",
		Description:  "Low-latency wireless headset with an integrated gaming microphone.",
		PriceAmount:  24990000,
		Stock:        18,
	},
	{
		Name:         "High-DPI Gaming Mouse",
		CategorySlug: "gaming-peripherals",
		Description:  "Lightweight precision mouse with adjustable high-DPI tracking.",
		PriceAmount:  8990000,
		Stock:        30,
	},
	{
		Name:         "Gaming Monitor",
		CategorySlug: "gaming-peripherals",
		Description:  "High-refresh-rate monitor designed for competitive gaming.",
		PriceAmount:  39990000,
		Stock:        12,
	},
	{
		Name:         "Current-Generation Console",
		CategorySlug: "consoles-handhelds",
		Description:  "Current-generation home console for living-room gaming.",
		PriceAmount:  79990000,
		Stock:        6,
	},
	{
		Name:         "Gaming Handheld",
		CategorySlug: "consoles-handhelds",
		Description:  "Portable gaming handheld for playing a library on the go.",
		PriceAmount:  69990000,
		Stock:        10,
	},
	{
		Name:         "Controller Charging Dock",
		CategorySlug: "gaming-accessories",
		Description:  "Charging dock for keeping compatible gaming controllers ready to play.",
		PriceAmount:  12990000,
		Stock:        20,
	},
	{
		Name:         "Extended Mouse Pad",
		CategorySlug: "gaming-accessories",
		Description:  "Extended desk mat with a smooth surface for keyboard and mouse movement.",
		PriceAmount:  4990000,
		Stock:        35,
	},
	{
		Name:         "USB Gaming Hub",
		CategorySlug: "gaming-accessories",
		Description:  "Powered USB hub for connecting gaming peripherals and accessories.",
		PriceAmount:  7990000,
		Stock:        22,
	},
}

func NewProductSeeder() ProductSeeder { return ProductSeeder{} }
func (ProductSeeder) Name() string    { return "products" }

func (s ProductSeeder) Seed(db *gorm.DB) error {
	var seller moduleauth.User
	if err := db.Where("LOWER(email) = ?", bootstrapDemoUserEmail).First(&seller).Error; err != nil {
		return err
	}

	categories := make(map[string]modulecatalog.Category, len(categorySeedRecords))
	for _, record := range categorySeedRecords {
		var category modulecatalog.Category
		if err := db.Where("slug = ?", record.Slug).First(&category).Error; err != nil {
			return err
		}
		categories[record.Slug] = category
	}

	for _, record := range productSeedRecords {
		category, ok := categories[record.CategorySlug]
		if !ok {
			return errors.New("product seed category is not registered")
		}

		var product modulecatalog.Product
		result := db.Unscoped().Where("seller_id = ? AND name = ?", seller.ID, record.Name).First(&product)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			publicID, err := identifier.NewPublicID()
			if err != nil {
				return err
			}
			if err := db.Create(&modulecatalog.Product{
				PublicID:      publicID,
				SellerID:      seller.ID,
				CategoryID:    category.ID,
				Name:          record.Name,
				Description:   record.Description,
				PriceAmount:   record.PriceAmount,
				StockQuantity: record.Stock,
				Status:        catalog.ProductStatusActive,
			}).Error; err != nil {
				return err
			}
			continue
		}

		if err := db.Model(&product).Updates(map[string]any{
			"category_id":    category.ID,
			"description":    record.Description,
			"price_amount":   record.PriceAmount,
			"stock_quantity": record.Stock,
			"status":         catalog.ProductStatusActive,
			"deleted_at":     nil,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s ProductSeeder) Clear(db *gorm.DB) error {
	var seller moduleauth.User
	if err := db.Where("LOWER(email) = ?", bootstrapDemoUserEmail).First(&seller).Error; err != nil {
		return err
	}

	names := make([]string, 0, len(productSeedRecords))
	for _, record := range productSeedRecords {
		names = append(names, record.Name)
	}
	return db.Where("seller_id = ? AND name IN ?", seller.ID, names).Delete(&modulecatalog.Product{}).Error
}
