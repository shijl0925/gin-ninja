package ninja

import "testing"

func TestResponseCodeStringAndInt(t *testing.T) {
	if got := CodeOK.String(); got != "0" {
		t.Fatalf("CodeOK.String() = %q", got)
	}

	got, err := CodeNotFound.Int()
	if err != nil {
		t.Fatalf("CodeNotFound.Int() error = %v", err)
	}
	if got != 404 {
		t.Fatalf("CodeNotFound.Int() = %d", got)
	}

	if _, err := ResponseCode("VALIDATION_ERROR").Int(); err == nil {
		t.Fatal("expected non-numeric response code to fail integer parsing")
	}
}
