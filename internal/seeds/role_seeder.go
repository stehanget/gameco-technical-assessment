package seeds

import (
	"errors"

	moduleauth "gameco-technical-assessment/internal/modules/auth"
	"gameco-technical-assessment/internal/modules/auth/role"

	"gorm.io/gorm"
)

type roleSeedRecord struct {
	Code        string
	Name        string
	Description string
}

type RoleSeeder struct{}

var roleSeedRecords = []roleSeedRecord{
	{Code: role.Buyer, Name: "Buyer", Description: "Customer who browses, manages a cart, and checks out"},
	{Code: role.Seller, Name: "Seller", Description: "Merchant who manages owned products and seller orders"},
	{Code: role.Admin, Name: "Admin", Description: "Internal operator with administrative access"},
}

func NewRoleSeeder() RoleSeeder { return RoleSeeder{} }
func (RoleSeeder) Name() string { return "roles" }

func (s RoleSeeder) Seed(db *gorm.DB) error {
	for _, record := range roleSeedRecords {
		var role moduleauth.Role
		result := db.Where("code = ?", record.Code).First(&role)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if err := db.Create(&moduleauth.Role{Code: record.Code, Name: record.Name, Description: record.Description}).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Model(&role).Updates(map[string]any{"name": record.Name, "description": record.Description}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s RoleSeeder) Clear(db *gorm.DB) error {
	codes := make([]string, 0, len(roleSeedRecords))
	for _, record := range roleSeedRecords {
		codes = append(codes, record.Code)
	}
	return db.Where("code IN ?", codes).Delete(&moduleauth.Role{}).Error
}
