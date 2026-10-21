package testsupport

import (
	"testing"

	"gorm.io/gorm"
)

func AssertMigrationsApplied(t *testing.T, db *gorm.DB) {
	t.Helper()
	var applied int64
	if err := db.Table("goose_db_version").Where("is_applied = ?", true).Count(&applied).Error; err != nil {
		t.Fatalf("read migration state: %v", err)
	}
	if applied == 0 {
		t.Fatal("expected at least one applied migration")
	}
}
