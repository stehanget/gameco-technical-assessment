package timestamp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseNormalizesOffsetToUTC(t *testing.T) {
	value, err := Parse("2026-10-21T17:15:30.123456+07:00")
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	if got, want := value.String(), "2026-10-21T10:15:30.123456Z"; got != want {
		t.Fatalf("timestamp = %q, want %q", got, want)
	}
}

func TestParseRejectsTimestampWithoutTimezone(t *testing.T) {
	if _, err := Parse("2026-10-21T17:15:30"); err == nil {
		t.Fatal("expected timezone-less timestamp to be rejected")
	}
}

func TestJSONRoundTripUsesUTC(t *testing.T) {
	value := New(time.Date(2026, 10, 21, 17, 15, 30, 123456000, time.FixedZone("WIB", 7*60*60)))
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal timestamp: %v", err)
	}
	if got, want := string(payload), `"2026-10-21T10:15:30.123456Z"`; got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}

	var decoded Timestamp
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal timestamp: %v", err)
	}
	if decoded.String() != value.String() {
		t.Fatalf("decoded = %q, want %q", decoded.String(), value.String())
	}
}
