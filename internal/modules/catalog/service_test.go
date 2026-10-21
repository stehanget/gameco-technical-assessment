package catalog

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	platformauth "gameco-technical-assessment/internal/platform/auth"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var catalogProductColumns = []string{
	"public_id", "name", "description", "price_amount", "stock_quantity",
	"category_slug", "category_name", "category_public_id", "status", "created_at", "updated_at",
}

func newCatalogTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open mock database: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("database mock expectations: %v", err)
		}
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		DriverName:           "postgres",
		PreferSimpleProtocol: true,
		WithoutReturning:     false,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm mock database: %v", err)
	}
	return db, mock
}

func productRow(publicID uuid.UUID, name, description string, price int64, stock int, categoryPublicID uuid.UUID, status string, createdAt time.Time) []any {
	return []any{publicID.String(), name, description, price, stock, "gaming-peripherals", "Gaming Peripherals", categoryPublicID.String(), status, createdAt, createdAt}
}

func expectProductQuery(mock sqlmock.Sqlmock, pattern string, rows ...[]any) {
	result := sqlmock.NewRows(catalogProductColumns)
	for _, row := range rows {
		values := make([]driver.Value, len(row))
		for index, value := range row {
			values[index] = value
		}
		result.AddRow(values...)
	}
	mock.ExpectQuery(pattern).WillReturnRows(result)
}

func expectNoProduct(mock sqlmock.Sqlmock, pattern string) {
	mock.ExpectQuery(pattern).WillReturnRows(sqlmock.NewRows(catalogProductColumns))
}

func expectProductQueryWithArgs(mock sqlmock.Sqlmock, pattern string, args []driver.Value, rows ...[]any) {
	result := sqlmock.NewRows(catalogProductColumns)
	for _, row := range rows {
		values := make([]driver.Value, len(row))
		for index, value := range row {
			values[index] = value
		}
		result.AddRow(values...)
	}
	mock.ExpectQuery(pattern).WithArgs(args...).WillReturnRows(result)
}

func expectNoProductWithArgs(mock sqlmock.Sqlmock, pattern string, args []driver.Value) {
	mock.ExpectQuery(pattern).WithArgs(args...).WillReturnRows(sqlmock.NewRows(catalogProductColumns))
}

func TestServiceListUsesCursorAndSearchesNameAndDescription(t *testing.T) {
	db, mock := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	categoryID := uuid.New()
	base := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	headsetID := uuid.New()
	mouseID := uuid.New()

	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.status = \$1.*p\.deleted_at IS NULL.*ORDER BY p\.created_at desc.*LIMIT \$2`,
		productRow(headsetID, "Gaming Headset", "Wireless audio headset", 200, 10, categoryID, "active", base.Add(time.Minute)),
		productRow(mouseID, "Gaming Mouse", "Precision wireless mouse", 100, 10, categoryID, "active", base),
	)
	first, err := service.List(context.Background(), ListQuery{PerPage: 1})
	if err != nil {
		t.Fatalf("first list: %v", err)
	}
	if len(first.Items) != 1 || !first.HasMore || first.NextCursor == nil || first.Items[0].Name != "Gaming Headset" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.status = \$1.*p\.deleted_at IS NULL.*p\.created_at < \$2.*p\.created_at = \$3.*p\.public_id > \$4.*ORDER BY p\.created_at desc.*LIMIT \$5`, productRow(mouseID, "Gaming Mouse", "Precision wireless mouse", 100, 10, categoryID, "active", base))
	second, err := service.List(context.Background(), ListQuery{PerPage: 1, Cursor: *first.NextCursor})
	if err != nil {
		t.Fatalf("second list: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Name != "Gaming Mouse" {
		t.Fatalf("unexpected second page: %+v", second)
	}

	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.status = \$1.*p\.deleted_at IS NULL.*LOWER\(p\.name\) LIKE.*LOWER\(p\.description\) LIKE.*ORDER BY p\.created_at desc.*LIMIT \$4`, productRow(headsetID, "Gaming Headset", "Wireless audio headset", 200, 10, categoryID, "active", base.Add(time.Minute)))
	search, err := service.List(context.Background(), ListQuery{Query: "audio", PerPage: 20})
	if err != nil || len(search.Items) != 1 || search.Items[0].Name != "Gaming Headset" {
		t.Fatalf("unexpected search result: result=%+v err=%v", search.Items, err)
	}
}

func TestServiceListRejectsCursorWithDifferentSort(t *testing.T) {
	db, mock := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	rowID := uuid.New()
	secondID := uuid.New()
	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.status = \$1.*p\.deleted_at IS NULL.*ORDER BY p\.created_at desc.*LIMIT \$2`,
		productRow(rowID, "Gaming Headset", "Wireless audio headset", 200, 10, uuid.New(), "active", time.Now().UTC()),
		productRow(secondID, "Gaming Mouse", "Precision wireless mouse", 100, 10, uuid.New(), "active", time.Now().UTC()),
	)
	first, err := service.List(context.Background(), ListQuery{PerPage: 1})
	if err != nil || first.NextCursor == nil {
		t.Fatalf("create cursor: result=%+v err=%v", first, err)
	}
	if _, err := service.List(context.Background(), ListQuery{Cursor: *first.NextCursor, SortBy: "price"}); err == nil {
		t.Fatal("expected cursor sort mismatch error")
	}
}

func TestServiceSellerListAndDetailAreOwnerScoped(t *testing.T) {
	db, mock := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	categoryID := uuid.New()
	base := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	firstID, secondID, thirdID, otherID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	expectProductQueryWithArgs(mock, `(?s)SELECT .*FROM products AS p.*p\.seller_id = \$1.*p\.deleted_at IS NULL.*ORDER BY p\.created_at desc.*LIMIT \$2`, []driver.Value{1, 21},
		productRow(firstID, "Gaming Headset", "Wireless audio headset", 200, 10, categoryID, "active", base.Add(time.Minute)),
		productRow(secondID, "Gaming Mouse", "Precision wireless mouse", 100, 10, categoryID, "active", base),
		productRow(thirdID, "Inactive Keyboard", "Hidden product", 50, 10, categoryID, "inactive", base.Add(2*time.Minute)),
	)
	result, err := service.ListBySeller(context.Background(), platformauth.Actor{UserID: 1}, ListQuery{PerPage: 20})
	if err != nil || len(result.Items) != 3 || result.Items[0].Status == "" {
		t.Fatalf("unexpected seller list: result=%+v err=%v", result, err)
	}

	expectProductQueryWithArgs(mock, `(?s)SELECT .*FROM products AS p.*p\.seller_id = \$1.*p\.deleted_at IS NULL.*ORDER BY p\.created_at desc.*LIMIT \$2`, []driver.Value{2, 21}, productRow(otherID, "Other Seller GPU", "Private seller product", 300, 10, categoryID, "active", base.Add(-time.Minute)))
	other, err := service.ListBySeller(context.Background(), platformauth.Actor{UserID: 2}, ListQuery{PerPage: 20})
	if err != nil || len(other.Items) != 1 {
		t.Fatalf("unexpected other seller list: result=%+v err=%v", other, err)
	}

	expectNoProductWithArgs(mock, `(?s)SELECT .*FROM products AS p.*p\.public_id = \$1.*p\.seller_id = \$2.*p\.deleted_at IS NULL.*LIMIT \$3`, []driver.Value{other.Items[0].PublicID.String(), 1, 1})
	if _, err := service.FindSellerProductByPublicID(context.Background(), platformauth.Actor{UserID: 1}, other.Items[0].PublicID.String()); err == nil {
		t.Fatal("expected cross-seller detail lookup to return not found")
	}
}

func TestServiceCategoryOptionsReturnsCanonicalCategories(t *testing.T) {
	db, mock := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	categoryID := uuid.New()
	created := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT .*FROM "categories".*ORDER BY name ASC, public_id ASC`).WillReturnRows(sqlmock.NewRows([]string{"id", "public_id", "name", "slug", "created_at", "updated_at"}).AddRow(1, categoryID.String(), "Gaming Peripherals", "gaming-peripherals", created, created))

	options, err := service.CategoryOptions(context.Background())
	if err != nil || len(options) != 1 || options[0].Slug != "gaming-peripherals" || options[0].PublicID == uuid.Nil {
		t.Fatalf("unexpected category options: options=%+v err=%v", options, err)
	}
}

func TestServiceCreateAndUpdateProductAreSellerScoped(t *testing.T) {
	db, mock := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	categoryID := uuid.New()
	categoryCreated := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	productID := uuid.New()
	actor1 := platformauth.Actor{UserID: 1}
	actor2 := platformauth.Actor{UserID: 2}

	mock.ExpectQuery(`(?s)SELECT .*FROM "categories".*WHERE public_id = \$1.*LIMIT \$2`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "public_id", "name", "slug", "created_at", "updated_at"}).AddRow(7, categoryID.String(), "Gaming Peripherals", "gaming-peripherals", categoryCreated, categoryCreated),
	)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO "products".*RETURNING "id"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectCommit()
	createdAt := categoryCreated.Add(time.Minute)
	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.public_id = \$1.*p\.seller_id = \$2.*p\.deleted_at IS NULL.*LIMIT \$3`, productRow(productID, "New Controller", "Wireless controller", 250, 8, categoryID, "active", createdAt))

	created, err := service.CreateProduct(context.Background(), actor1, CreateProductRequest{
		Name: "New Controller", Description: "Wireless controller", PriceAmount: 250,
		StockQuantity: 8, CategoryPublicID: categoryID,
	})
	if err != nil || created.Name != "New Controller" || created.Status != "active" {
		t.Fatalf("unexpected created product: product=%+v err=%v", created, err)
	}

	newName := "Updated Controller"
	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.public_id = \$1.*p\.seller_id = \$2.*p\.deleted_at IS NULL.*LIMIT \$3`, productRow(productID, "New Controller", "Wireless controller", 250, 8, categoryID, "active", createdAt))
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE "products" SET .*WHERE \(seller_id = \$3 AND public_id = \$4 AND deleted_at IS NULL\)`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectProductQuery(mock, `(?s)SELECT .*FROM products AS p.*p\.public_id = \$1.*p\.seller_id = \$2.*p\.deleted_at IS NULL.*LIMIT \$3`, productRow(productID, "Updated Controller", "Wireless controller", 250, 8, categoryID, "active", createdAt))

	updated, err := service.UpdateProduct(context.Background(), actor1, productID.String(), UpdateProductRequest{Name: &newName})
	if err != nil || updated.Name != newName {
		t.Fatalf("unexpected updated product: product=%+v err=%v", updated, err)
	}

	expectNoProduct(mock, `(?s)SELECT .*FROM products AS p.*p\.public_id = \$1.*p\.seller_id = \$2.*p\.deleted_at IS NULL.*LIMIT \$3`)
	if _, err := service.UpdateProduct(context.Background(), actor2, productID.String(), UpdateProductRequest{Name: &newName}); err == nil {
		t.Fatal("expected foreign seller update to fail")
	}
}

func TestServiceValidationDoesNotHitDatabase(t *testing.T) {
	db, _ := newCatalogTestDB(t)
	service := NewService(NewRepository(db))
	if _, err := service.List(context.Background(), ListQuery{PerPage: 101}); err == nil {
		t.Fatal("expected invalid page size")
	}
	if _, err := service.FindByPublicID(context.Background(), "not-a-uuid"); err == nil {
		t.Fatal("expected invalid public id")
	}
}

func TestUpdateProductRequestRejectsExplicitNull(t *testing.T) {
	var request UpdateProductRequest
	if err := json.Unmarshal([]byte(`{"name":null}`), &request); err == nil {
		t.Fatal("expected explicit null name to be rejected")
	}
}
