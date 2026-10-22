package checkout

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDecodeCursorSupportsSellerTotalSort(t *testing.T) {
	encoded, err := encodeCursor(cursorPosition{
		Version:   1,
		SortBy:    "seller_total",
		SortOrder: "desc",
		Filter:    "fingerprint",
		Value:     "1250",
		Tie:       uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode seller_total cursor: %v", err)
	}
	if decoded.SortBy != "seller_total" || decoded.Value != "1250" {
		t.Fatalf("unexpected cursor: %+v", decoded)
	}
}

func TestDecodeCursorSupportsCreatedAtSort(t *testing.T) {
	encoded, err := encodeCursor(cursorPosition{
		Version:   1,
		SortBy:    "created_at",
		SortOrder: "desc",
		Filter:    "fingerprint",
		Value:     time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		Tie:       uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor(encoded); err != nil {
		t.Fatalf("decode created_at cursor: %v", err)
	}
}

func TestDecodeCursorSupportsTotalAmountSort(t *testing.T) {
	encoded, err := encodeCursor(cursorPosition{Version: 1, SortBy: "total_amount", SortOrder: "desc", Filter: "fingerprint", Value: "1250", Tie: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeCursor(encoded)
	if err != nil || decoded.SortBy != "total_amount" {
		t.Fatalf("decode total_amount cursor: %v, %+v", err, decoded)
	}
	if _, _, err := orderCursorCondition("total_amount", "desc", decoded); err != nil {
		t.Fatalf("validate total_amount cursor: %v", err)
	}
}

func TestNormalizeOrderQueryRejectsCursorFingerprintMismatch(t *testing.T) {
	encoded, err := encodeCursor(cursorPosition{Version: 1, SortBy: "created_at", SortOrder: "desc", Filter: "stale", Value: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), Tie: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := normalizeOrderQuery(OrderListQuery{Cursor: encoded}); err == nil {
		t.Fatal("expected cursor fingerprint mismatch")
	}
}
