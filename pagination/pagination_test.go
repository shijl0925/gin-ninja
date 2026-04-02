package pagination_test

import (
	"testing"

	"github.com/shijl0925/gin-ninja/pagination"
)

func TestPageInput_Defaults(t *testing.T) {
	p := pagination.PageInput{}
	if p.GetPage() != pagination.DefaultPage {
		t.Errorf("expected default page %d, got %d", pagination.DefaultPage, p.GetPage())
	}
	if p.GetSize() != pagination.DefaultSize {
		t.Errorf("expected default size %d, got %d", pagination.DefaultSize, p.GetSize())
	}
}

func TestPageInput_Clamps(t *testing.T) {
	p := pagination.PageInput{Page: -1, Size: 9999}
	if p.GetPage() != pagination.DefaultPage {
		t.Errorf("expected clamped page %d, got %d", pagination.DefaultPage, p.GetPage())
	}
	if p.GetSize() != pagination.MaxSize {
		t.Errorf("expected clamped size %d, got %d", pagination.MaxSize, p.GetSize())
	}
}

func TestPageInput_Offset(t *testing.T) {
	p := pagination.PageInput{Page: 3, Size: 10}
	if p.Offset() != 20 {
		t.Errorf("expected offset 20, got %d", p.Offset())
	}
}

func TestNewPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	input := pagination.PageInput{Page: 2, Size: 3}
	page := pagination.NewPage(items, 10, input)

	if page.Total != 10 {
		t.Errorf("expected total 10, got %d", page.Total)
	}
	if page.Page != 2 {
		t.Errorf("expected page 2, got %d", page.Page)
	}
	if page.Size != 3 {
		t.Errorf("expected size 3, got %d", page.Size)
	}
	if page.Pages != 4 { // ceil(10/3) = 4
		t.Errorf("expected pages 4, got %d", page.Pages)
	}
	if len(page.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(page.Items))
	}
}
