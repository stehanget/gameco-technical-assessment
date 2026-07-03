package auth

import "testing"

func TestHashPasswordAndCompare(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if err := ComparePassword(hash, "password123"); err != nil {
		t.Fatalf("compare password: %v", err)
	}

	if err := ComparePassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected compare password to fail")
	}
}
