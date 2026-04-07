package middleware

import (
	"errors"
	"testing"
)

func TestGenerateCSRFToken_PanicsWhenRandomFails(t *testing.T) {
	original := csrfRandRead
	csrfRandRead = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	defer func() {
		csrfRandRead = original
	}()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when crypto/rand fails")
		}
	}()

	_ = generateCSRFToken()
}
