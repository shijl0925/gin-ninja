package contextkeys

import "testing"

func TestContextKeyConstants(t *testing.T) {
	if JWTClaims == "" || Locale == "" {
		t.Fatal("context keys must not be empty")
	}
	if JWTClaims == Locale {
		t.Fatal("context keys must be distinct")
	}
}
