package seeds

import (
	"errors"
	"strings"

	moduleauth "gameco-technical-assessment/internal/modules/auth"

	"gorm.io/gorm"
)

type userRoleSeedRecord struct {
	UserEmail string
	RoleCode  string
}

type UserRoleSeeder struct{}

var userRoleSeedRecords = []userRoleSeedRecord{
	{UserEmail: "user@mail.com", RoleCode: "buyer"},
	{UserEmail: "user@mail.com", RoleCode: "seller"},
}

func NewUserRoleSeeder() UserRoleSeeder { return UserRoleSeeder{} }
func (UserRoleSeeder) Name() string     { return "user_roles" }

func (s UserRoleSeeder) Seed(db *gorm.DB) error {
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&moduleauth.UserRole{}).Error; err != nil {
		return err
	}
	for _, record := range userRoleSeedRecords {
		var user moduleauth.User
		if err := db.Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(record.UserEmail))).First(&user).Error; err != nil {
			return err
		}
		var role moduleauth.Role
		if err := db.Where("code = ?", record.RoleCode).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("role not found: " + record.RoleCode)
			}
			return err
		}
		if err := db.Create(&moduleauth.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s UserRoleSeeder) Clear(db *gorm.DB) error {
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&moduleauth.UserRole{}).Error
}
