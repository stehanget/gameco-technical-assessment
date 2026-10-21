package seeds

import (
	"strings"

	"gameco-technical-assessment/internal/constants/permission"
	moduleauth "gameco-technical-assessment/internal/modules/auth"

	"gorm.io/gorm"
)

type permissionSeedRecord struct {
	Code        string
	Description string
}

type PermissionSeeder struct{}

var permissionSeedRecords = []permissionSeedRecord{
	{Code: permission.AuthMe, Description: "Read the authenticated user profile"},
	{Code: permission.ProductRead, Description: "Browse product catalog"},
	{Code: permission.ProductCreate, Description: "Create products"},
	{Code: permission.ProductUpdate, Description: "Update products"},
	{Code: permission.ProductDelete, Description: "Delete products"},
	{Code: permission.CartRead, Description: "Read the current user's cart"},
	{Code: permission.CartItemWrite, Description: "Add or update cart items"},
	{Code: permission.CheckoutCreate, Description: "Create an order from the current user's cart"},
	{Code: permission.OrderRead, Description: "Read orders subject to object ownership"},
	{Code: permission.SellerOrderRead, Description: "Read orders containing the seller's products"},
	{Code: permission.PaymentWebhookProcess, Description: "Process signed payment webhooks"},
	{Code: permission.UserRead, Description: "Read user accounts"},
	{Code: permission.UserUpdate, Description: "Update user accounts"},
	{Code: permission.AuditRead, Description: "Read security and business audit events"},
}

func NewPermissionSeeder() PermissionSeeder { return PermissionSeeder{} }
func (PermissionSeeder) Name() string       { return "permissions" }

func (s PermissionSeeder) Seed(db *gorm.DB) error {
	for _, record := range permissionSeedRecords {
		var permission moduleauth.Permission
		result := db.Where("code = ?", record.Code).Find(&permission)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := db.Create(&moduleauth.Permission{Code: record.Code, Description: record.Description}).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Model(&permission).Updates(map[string]any{"description": record.Description}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s PermissionSeeder) Clear(db *gorm.DB) error {
	codes := make([]string, 0, len(permissionSeedRecords))
	for _, record := range permissionSeedRecords {
		codes = append(codes, strings.TrimSpace(record.Code))
	}
	return db.Where("code IN ?", codes).Delete(&moduleauth.Permission{}).Error
}
