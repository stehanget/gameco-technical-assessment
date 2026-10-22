package audit

import (
	"context"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestWriteUsesEmptyJSONMetadataWhenMetadataIsNil(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open mock database: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		DriverName:           "postgres",
		PreferSimpleProtocol: true,
		WithoutReturning:     false,
	}), &gorm.Config{Logger: gormlogger.New(log.New(os.Stderr, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})})
	if err != nil {
		t.Fatalf("open gorm mock database: %v", err)
	}

	createdAt := time.Unix(1792584000, 0).UTC()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "audit_events"`)).
		WithArgs(nil, "session-1", "auth_session", createdAt, "auth.login_succeeded", "127.0.0.1", "{}", "test-agent").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := Write(context.Background(), db, Event{
		EventType:     "auth.login_succeeded",
		AggregateType: "auth_session",
		AggregateID:   "session-1",
		IPAddress:     "127.0.0.1",
		UserAgent:     "test-agent",
		CreatedAt:     createdAt,
	}); err != nil {
		t.Fatalf("write audit event: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database mock expectations: %v", err)
	}
}
