package ninja

import "testing"

func TestResponseCodeHelpers(t *testing.T) {
	if got := OK.String(); got != "200" {
		t.Fatalf("OK.String() = %q", got)
	}
	if got := NOT_FOUND.Int(); got != 404 {
		t.Fatalf("NOT_FOUND.Int() = %d", got)
	}
	if got := CREATED.Text(); got != "Created" {
		t.Fatalf("CREATED.Text() = %q", got)
	}
	if got := UNAUTHORIZED.Description(); got != "No permission -- see authorization schemes" {
		t.Fatalf("UNAUTHORIZED.Description() = %q", got)
	}
	if got := CodeNotFound; got != NOT_FOUND {
		t.Fatalf("CodeNotFound = %d, want %d", got, NOT_FOUND)
	}
}
