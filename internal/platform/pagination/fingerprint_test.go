package pagination

import "testing"

func TestFilterFingerprintIsIndependentOfMapInsertionOrder(t *testing.T) {
	first := FilterFingerprint(map[string]string{
		"q":          "gaming",
		"sort_by":    "created_at",
		"sort_order": "desc",
	})
	second := FilterFingerprint(map[string]string{
		"sort_order": "desc",
		"q":          "gaming",
		"sort_by":    "created_at",
	})
	if first != second {
		t.Fatalf("fingerprint changed with map insertion order: %q != %q", first, second)
	}
}

func TestFilterFingerprintChangesWhenEffectiveInputChanges(t *testing.T) {
	first := FilterFingerprint(map[string]string{"q": "gaming"})
	second := FilterFingerprint(map[string]string{"q": "headset"})
	if first == second {
		t.Fatal("fingerprint did not change when effective input changed")
	}
}
