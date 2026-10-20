package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&Category{}, &Product{}); err != nil {
		t.Fatalf("migrate catalog tables: %v", err)
	}
	return db
}

func seedCatalogProducts(t *testing.T, db *gorm.DB) {
	t.Helper()
	category := Category{PublicID: uuid.New(), Name: "Gaming Peripherals", Slug: "gaming-peripherals"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	base := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	products := []Product{
		{PublicID: uuid.New(), SellerID: 1, CategoryID: category.ID, Name: "Gaming Mouse", Description: "Precision wireless mouse", PriceAmount: 100, StockQuantity: 10, Status: "active", CreatedAt: base, UpdatedAt: base},
		{PublicID: uuid.New(), SellerID: 1, CategoryID: category.ID, Name: "Gaming Headset", Description: "Wireless audio headset", PriceAmount: 200, StockQuantity: 10, Status: "active", CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)},
		{PublicID: uuid.New(), SellerID: 1, CategoryID: category.ID, Name: "Inactive Keyboard", Description: "Hidden product", PriceAmount: 50, StockQuantity: 10, Status: "inactive", CreatedAt: base.Add(2 * time.Minute), UpdatedAt: base.Add(2 * time.Minute)},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("create products: %v", err)
	}
}

func TestServiceListUsesCursorAndSearchesNameAndDescription(t *testing.T) {
	db := newCatalogTestDB(t)
	seedCatalogProducts(t, db)
	service := NewService(NewRepository(db))

	first, err := service.List(context.Background(), ListQuery{PerPage: 1})
	if err != nil {
		t.Fatalf("first list: %v", err)
	}
	if len(first.Items) != 1 || !first.HasMore || first.NextCursor == nil {
		t.Fatalf("unexpected first page: %+v", first)
	}
	if first.Items[0].Name != "Gaming Headset" {
		t.Fatalf("unexpected first product: %q", first.Items[0].Name)
	}

	second, err := service.List(context.Background(), ListQuery{PerPage: 1, Cursor: *first.NextCursor})
	if err != nil {
		t.Fatalf("second list: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Name != "Gaming Mouse" {
		t.Fatalf("unexpected second page: %+v", second)
	}

	search, err := service.List(context.Background(), ListQuery{Query: "audio", PerPage: 20})
	if err != nil {
		t.Fatalf("search list: %v", err)
	}
	if len(search.Items) != 1 || search.Items[0].Name != "Gaming Headset" {
		t.Fatalf("unexpected search result: %+v", search.Items)
	}
}

func TestServiceListRejectsCursorWithDifferentSort(t *testing.T) {
	db := newCatalogTestDB(t)
	seedCatalogProducts(t, db)
	service := NewService(NewRepository(db))

	first, err := service.List(context.Background(), ListQuery{PerPage: 1})
	if err != nil || first.NextCursor == nil {
		t.Fatalf("create cursor: result=%+v err=%v", first, err)
	}
	if _, err := service.List(context.Background(), ListQuery{Cursor: *first.NextCursor, SortBy: "price"}); err == nil {
		t.Fatal("expected cursor sort mismatch error")
	}
}
