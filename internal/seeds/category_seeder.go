package seeds

import (
	modulecatalog "gameco-technical-assessment/internal/modules/catalog"
	"gameco-technical-assessment/internal/platform/identifier"

	"gorm.io/gorm"
)

type categorySeedRecord struct {
	Name string
	Slug string
}

type CategorySeeder struct{}

var categorySeedRecords = []categorySeedRecord{
	{Name: "PC Components", Slug: "pc-components"},
	{Name: "Gaming Peripherals", Slug: "gaming-peripherals"},
	{Name: "Consoles & Handhelds", Slug: "consoles-handhelds"},
	{Name: "Gaming Accessories", Slug: "gaming-accessories"},
}

func NewCategorySeeder() CategorySeeder { return CategorySeeder{} }
func (CategorySeeder) Name() string     { return "categories" }

func (s CategorySeeder) Seed(db *gorm.DB) error {
	for _, record := range categorySeedRecords {
		var category modulecatalog.Category
		result := db.Where("slug = ?", record.Slug).Find(&category)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			publicID, err := identifier.NewPublicID()
			if err != nil {
				return err
			}
			if err := db.Create(&modulecatalog.Category{
				PublicID: publicID,
				Name:     record.Name,
				Slug:     record.Slug,
			}).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Model(&category).Update("name", record.Name).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s CategorySeeder) Clear(db *gorm.DB) error {
	slugs := make([]string, 0, len(categorySeedRecords))
	for _, record := range categorySeedRecords {
		slugs = append(slugs, record.Slug)
	}
	return db.Where("slug IN ?", slugs).Delete(&modulecatalog.Category{}).Error
}
