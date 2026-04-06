package pagination

import "testing"

func TestNewPageZeroTotalHasZeroPages(t *testing.T) {
	page := NewPage([]int{}, 0, PageInput{Page: 1, Size: 20})
	if page.Pages != 0 {
		t.Fatalf("expected zero pages for zero total, got %d", page.Pages)
	}
}
