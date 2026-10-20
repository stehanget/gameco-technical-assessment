package clock

import (
	"testing"
	"time"
)

func TestFixedClockReturnsUTC(t *testing.T) {
	fixed := Fixed{Current: time.Date(2026, 10, 21, 17, 15, 30, 0, time.FixedZone("WIB", 7*60*60))}
	if got, want := fixed.Now().Format(time.RFC3339), "2026-10-21T10:15:30Z"; got != want {
		t.Fatalf("now = %q, want %q", got, want)
	}
}
