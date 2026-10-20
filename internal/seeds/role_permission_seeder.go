package seeds

import (
	"errors"

	"gameco-technical-assessment/internal/constants/permission"
	"gameco-technical-assessment/internal/constants/role"
	moduleauth "gameco-technical-assessment/internal/modules/auth"

	"gorm.io/gorm"
)

type rolePermissionSeedRecord struct {
	RoleCode       string
	PermissionCode string
}

type RolePermissionSeeder struct{}

var rolePermissionSeedRecords = []rolePermissionSeedRecord{
	{RoleCode: role.Buyer, PermissionCode: permission.AuthMe},
	{RoleCode: role.Buyer, PermissionCode: permission.ProductRead},
	{RoleCode: role.Buyer, PermissionCode: permission.CartRead},
	{RoleCode: role.Buyer, PermissionCode: permission.CartItemWrite},
	{RoleCode: role.Buyer, PermissionCode: permission.CheckoutCreate},
	{RoleCode: role.Buyer, PermissionCode: permission.OrderRead},
	{RoleCode: role.Seller, PermissionCode: permission.AuthMe},
	{RoleCode: role.Seller, PermissionCode: permission.ProductRead},
	{RoleCode: role.Seller, PermissionCode: permission.ProductCreate},
	{RoleCode: role.Seller, PermissionCode: permission.ProductUpdate},
	{RoleCode: role.Seller, PermissionCode: permission.ProductDelete},
	{RoleCode: role.Seller, PermissionCode: permission.SellerOrderRead},
	{RoleCode: role.Admin, PermissionCode: permission.AuthMe},
	{RoleCode: role.Admin, PermissionCode: permission.ProductRead},
	{RoleCode: role.Admin, PermissionCode: permission.ProductCreate},
	{RoleCode: role.Admin, PermissionCode: permission.ProductUpdate},
	{RoleCode: role.Admin, PermissionCode: permission.ProductDelete},
	{RoleCode: role.Admin, PermissionCode: permission.CartRead},
	{RoleCode: role.Admin, PermissionCode: permission.CartItemWrite},
	{RoleCode: role.Admin, PermissionCode: permission.CheckoutCreate},
	{RoleCode: role.Admin, PermissionCode: permission.OrderRead},
	{RoleCode: role.Admin, PermissionCode: permission.SellerOrderRead},
	{RoleCode: role.Admin, PermissionCode: permission.PaymentWebhookProcess},
	{RoleCode: role.Admin, PermissionCode: permission.UserRead},
	{RoleCode: role.Admin, PermissionCode: permission.UserUpdate},
	{RoleCode: role.Admin, PermissionCode: permission.AuditRead},
}

func NewRolePermissionSeeder() RolePermissionSeeder { return RolePermissionSeeder{} }
func (RolePermissionSeeder) Name() string           { return "role_permissions" }

func (s RolePermissionSeeder) Seed(db *gorm.DB) error {
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&moduleauth.RolePermission{}).Error; err != nil {
		return err
	}
	for _, record := range rolePermissionSeedRecords {
		var role moduleauth.Role
		if err := db.Where("code = ?", record.RoleCode).First(&role).Error; err != nil {
			return err
		}
		var permission moduleauth.Permission
		if err := db.Where("code = ?", record.PermissionCode).First(&permission).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("permission not found: " + record.PermissionCode)
			}
			return err
		}
		if err := db.Create(&moduleauth.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s RolePermissionSeeder) Clear(db *gorm.DB) error {
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&moduleauth.RolePermission{}).Error
}
