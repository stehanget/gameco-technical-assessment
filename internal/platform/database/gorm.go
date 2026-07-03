package database

import (
	"gameco-technical-assessment/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewGORM(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(BuildPostgresDSN(cfg)), &gorm.Config{})
}

func CloseGORM(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
