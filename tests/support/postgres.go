package testsupport

import (
	"testing"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/platform/database"

	"gorm.io/gorm"
)

func OpenPostgres(t *testing.T, cfg config.Config) *gorm.DB {
	t.Helper()
	db, err := database.NewGORM(cfg)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		_ = database.CloseGORM(db)
	})
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	return db
}
