package pagination

import "testing"

func TestResolveSort(t *testing.T) {
	input := PageInput{Sort: "name,-created_at"}
	schema := NewSortSchema("name").Allow("created_at")

	fields, err := input.ResolveSort(schema)
	if err != nil {
		t.Fatalf("ResolveSort: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Name != "name" || fields[0].Desc {
		t.Fatalf("unexpected first field: %+v", fields[0])
	}
	if fields[1].Name != "created_at" || !fields[1].Desc {
		t.Fatalf("unexpected second field: %+v", fields[1])
	}
}

func TestResolveSortRejectsUnknownField(t *testing.T) {
	_, err := (PageInput{Sort: "password"}).ResolveSort(NewSortSchema("name"))
	if err == nil {
		t.Fatal("expected sort validation error")
	}
}

func TestNewPageZeroTotalHasZeroPages(t *testing.T) {
	page := NewPage([]int{}, 0, PageInput{Page: 1, Size: 20})
	if page.Pages != 0 {
		t.Fatalf("expected zero pages for zero total, got %d", page.Pages)
	}
}
