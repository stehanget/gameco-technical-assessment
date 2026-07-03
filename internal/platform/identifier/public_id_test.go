package identifier

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewPublicIDReturnsUUIDv7(t *testing.T) {
	id, err := NewPublicID()
	if err != nil {
		t.Fatalf("generate public id: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("expected non-nil public id")
	}
	if id.Version() != 7 {
		t.Fatalf("expected UUIDv7, got version %d", id.Version())
	}
}
