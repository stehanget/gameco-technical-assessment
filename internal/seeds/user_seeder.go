package seeds

import (
	"strings"

	moduleauth "gameco-technical-assessment/internal/modules/auth"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/identifier"

	"gorm.io/gorm"
)

type userSeedRecord struct {
	Email    string
	Password string
}

type UserSeeder struct{}

var userSeedRecords = []userSeedRecord{
	{Email: bootstrapDemoUserEmail, Password: "admin123"},
}

func NewUserSeeder() UserSeeder { return UserSeeder{} }
func (UserSeeder) Name() string { return "users" }

func (s UserSeeder) Seed(db *gorm.DB) error {
	for _, record := range userSeedRecords {
		email := strings.ToLower(strings.TrimSpace(record.Email))
		passwordHash, err := platformauth.HashPassword(record.Password)
		if err != nil {
			return err
		}

		var user moduleauth.User
		result := db.Where("LOWER(email) = ?", email).Find(&user)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			publicID, err := identifier.NewPublicID()
			if err != nil {
				return err
			}
			if err := db.Create(&moduleauth.User{PublicID: publicID, Email: email, PasswordHash: passwordHash}).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Model(&user).Update("password_hash", passwordHash).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s UserSeeder) Clear(db *gorm.DB) error {
	emails := make([]string, 0, len(userSeedRecords))
	for _, record := range userSeedRecords {
		emails = append(emails, strings.ToLower(strings.TrimSpace(record.Email)))
	}
	return db.Where("LOWER(email) IN ?", emails).Delete(&moduleauth.User{}).Error
}
