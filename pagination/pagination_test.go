package pagination

import "testing"

func TestPageInputDefaultsAndBounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		input  PageInput
		page   int
		size   int
		offset int
		limit  int
	}{
		{
			name:   "defaults",
			input:  PageInput{},
			page:   DefaultPage,
			size:   DefaultSize,
			offset: 0,
			limit:  DefaultSize,
		},
		{
			name:   "invalid values are clamped",
			input:  PageInput{Page: -2, Size: -5},
			page:   DefaultPage,
			size:   DefaultSize,
			offset: 0,
			limit:  DefaultSize,
		},
		{
			name:   "oversized page size is capped",
			input:  PageInput{Page: 3, Size: MaxSize + 25},
			page:   3,
			size:   MaxSize,
			offset: 2 * MaxSize,
			limit:  MaxSize,
		},
		{
			name:   "explicit valid values are preserved",
			input:  PageInput{Page: 2, Size: 15},
			page:   2,
			size:   15,
			offset: 15,
			limit:  15,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.input.GetPage(); got != tc.page {
				t.Fatalf("GetPage() = %d, want %d", got, tc.page)
			}
			if got := tc.input.GetSize(); got != tc.size {
				t.Fatalf("GetSize() = %d, want %d", got, tc.size)
			}
			if got := tc.input.Offset(); got != tc.offset {
				t.Fatalf("Offset() = %d, want %d", got, tc.offset)
			}
			if got := tc.input.Limit(); got != tc.limit {
				t.Fatalf("Limit() = %d, want %d", got, tc.limit)
			}
		})
	}
}

func TestNewPageZeroTotalHasZeroPages(t *testing.T) {
	t.Parallel()

	page := NewPage([]int{}, 0, PageInput{Page: 1, Size: 20})
	if page.Pages != 0 {
		t.Fatalf("expected zero pages for zero total, got %d", page.Pages)
	}
}

func TestNewPageRoundsUpAndNormalizesNilItems(t *testing.T) {
	t.Parallel()

	page := NewPage[int](nil, 21, PageInput{})

	if page.Items == nil {
		t.Fatal("expected nil items to be normalized to an empty slice")
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected empty items slice, got %d item(s)", len(page.Items))
	}
	if page.Page != DefaultPage {
		t.Fatalf("expected default page %d, got %d", DefaultPage, page.Page)
	}
	if page.Size != DefaultSize {
		t.Fatalf("expected default size %d, got %d", DefaultSize, page.Size)
	}
	if page.Pages != 2 {
		t.Fatalf("expected 2 pages, got %d", page.Pages)
	}
}
