package orm

import (
	"errors"
	"strconv"
	"testing"

	"github.com/shijl0925/gin-ninja/pagination"
)

type cursorTestItem struct {
	ID   int
	Name string
}

func TestSelectCursorPageAscending(t *testing.T) {
	db := testDB(t)
	if err := db.Migrator().DropTable(&cursorTestItem{}); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if err := db.AutoMigrate(&cursorTestItem{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create([]cursorTestItem{
		{ID: 1, Name: "one"},
		{ID: 2, Name: "two"},
		{ID: 3, Name: "three"},
		{ID: 4, Name: "four"},
	}).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	items, nextCursor, err := SelectCursorPage(
		db,
		pagination.CursorPagination{Cursor: "1", Size: 2},
		"id",
		strconv.Atoi,
		func(item cursorTestItem) string { return strconv.Itoa(item.ID) },
	)
	if err != nil {
		t.Fatalf("SelectCursorPage: %v", err)
	}
	if len(items) != 2 || items[0].ID != 2 || items[1].ID != 3 {
		t.Fatalf("unexpected cursor page items: %+v", items)
	}
	if nextCursor != "3" {
		t.Fatalf("nextCursor = %q, want %q", nextCursor, "3")
	}
}

func TestSelectCursorPageDescendingLastPage(t *testing.T) {
	db := testDB(t)
	if err := db.Migrator().DropTable(&cursorTestItem{}); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if err := db.AutoMigrate(&cursorTestItem{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create([]cursorTestItem{
		{ID: 1, Name: "one"},
		{ID: 2, Name: "two"},
		{ID: 3, Name: "three"},
	}).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	items, nextCursor, err := SelectCursorPage(
		db,
		pagination.CursorPagination{Cursor: "2", Size: 2},
		"id",
		strconv.Atoi,
		func(item cursorTestItem) string { return strconv.Itoa(item.ID) },
		CursorPageDesc(),
	)
	if err != nil {
		t.Fatalf("SelectCursorPage: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("unexpected cursor page items: %+v", items)
	}
	if nextCursor != "" {
		t.Fatalf("nextCursor = %q, want empty", nextCursor)
	}
}

func TestSelectCursorPageValidatesInputs(t *testing.T) {
	db := testDB(t)

	if _, _, err := SelectCursorPage[cursorTestItem](nil, pagination.CursorPagination{}, "id", strconv.Atoi, func(item cursorTestItem) string { return "" }); err == nil {
		t.Fatal("expected nil database error")
	}
	if _, _, err := SelectCursorPage(db, pagination.CursorPagination{}, "", strconv.Atoi, func(item cursorTestItem) string { return "" }); err == nil {
		t.Fatal("expected missing cursor column error")
	}
	if _, _, err := SelectCursorPage[cursorTestItem, int](db, pagination.CursorPagination{}, "id", nil, func(item cursorTestItem) string { return "" }); err == nil {
		t.Fatal("expected missing cursor decoder error")
	}
	if _, _, err := SelectCursorPage[cursorTestItem](db, pagination.CursorPagination{}, "id", strconv.Atoi, nil); err == nil {
		t.Fatal("expected missing cursor extractor error")
	}
}

func TestSelectCursorPageDecodesOnlyNonEmptyCursor(t *testing.T) {
	db := testDB(t)
	if err := db.Migrator().DropTable(&cursorTestItem{}); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if err := db.AutoMigrate(&cursorTestItem{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create([]cursorTestItem{
		{ID: 1, Name: "one"},
		{ID: 2, Name: "two"},
		{ID: 3, Name: "three"},
	}).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	decodeCalls := 0
	items, nextCursor, err := SelectCursorPage(
		db,
		pagination.CursorPagination{Size: 2},
		"id",
		func(cursor string) (int, error) {
			decodeCalls++
			return strconv.Atoi(cursor)
		},
		func(item cursorTestItem) string { return strconv.Itoa(item.ID) },
	)
	if err != nil {
		t.Fatalf("SelectCursorPage: %v", err)
	}
	if decodeCalls != 0 {
		t.Fatalf("expected empty cursor to skip decoding, got %d call(s)", decodeCalls)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 {
		t.Fatalf("unexpected cursor page items: %+v", items)
	}
	if nextCursor != "2" {
		t.Fatalf("nextCursor = %q, want %q", nextCursor, "2")
	}
}

func TestSelectCursorPageReturnsDecodeError(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("bad cursor")

	_, _, err := SelectCursorPage(
		db,
		pagination.CursorPagination{Cursor: "bad", Size: 2},
		"id",
		func(string) (int, error) { return 0, wantErr },
		func(item cursorTestItem) string { return strconv.Itoa(item.ID) },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected decode error %v, got %v", wantErr, err)
	}
}
