// Package pagination provides reusable pagination types for gin-ninja handlers.
//
// Usage:
//
//	type ListUsersInput struct {
//	    pagination.PageInput
//	    Search string `query:"search"`
//	}
//
//	func listUsers(ctx *ninja.Context, input *ListUsersInput) (*pagination.Page[UserOut], error) {
//	    // service.Page(input.GetPage(), input.GetSize(), ...)
//	}
package pagination

import "math"

const maxInt = int(^uint(0) >> 1)

// DefaultPage is the default page number when not specified.
const DefaultPage = 1

// DefaultSize is the default page size when not specified.
const DefaultSize = 20

// MaxSize is the maximum allowed page size.
const MaxSize = 100

// PageInput is an embeddable struct that captures pagination query parameters.
//
//	type ListUsersInput struct {
//	    pagination.PageInput
//	    Name string `query:"name"`
//	}
type PageInput struct {
	// Page is the 1-based page number (default: 1).
	Page int `query:"page" binding:"omitempty,min=1" json:"-"`
	// Size is the number of items per page (default: 20, max: 100).
	Size int `query:"size" binding:"omitempty,min=1,max=100" json:"-"`
}

// CursorPagination is an embeddable struct that captures cursor pagination
// query parameters. Cursor values are opaque to gin-ninja; handlers should
// encode the stable ordering values needed by their storage layer.
//
//	type ListEventsInput struct {
//	    pagination.CursorPagination
//	}
type CursorPagination struct {
	// Cursor is the opaque cursor returned by a previous page.
	Cursor string `query:"cursor" binding:"omitempty,max=512" json:"-"`
	// Size is the number of items per page (default: 20, max: 100).
	Size int `query:"size" binding:"omitempty,min=1,max=100" json:"-"`
}

// GetPage returns the effective page number (at least 1).
func (p PageInput) GetPage() int {
	if p.Page < 1 {
		return DefaultPage
	}
	return p.Page
}

// GetSize returns the effective page size (between 1 and MaxSize).
func (p PageInput) GetSize() int {
	switch {
	case p.Size < 1:
		return DefaultSize
	case p.Size > MaxSize:
		return MaxSize
	default:
		return p.Size
	}
}

// Offset computes the SQL OFFSET value from Page and Size.
func (p PageInput) Offset() int {
	page := p.GetPage()
	if page <= 1 {
		return 0
	}

	size := p.GetSize()
	offset := page - 1
	if offset > maxInt/size {
		return maxInt
	}
	return offset * size
}

// Limit returns the SQL LIMIT value (identical to GetSize).
func (p PageInput) Limit() int {
	return p.GetSize()
}

// GetCursor returns the requested opaque cursor.
func (p CursorPagination) GetCursor() string {
	return p.Cursor
}

// GetSize returns the effective cursor page size (between 1 and MaxSize).
func (p CursorPagination) GetSize() int {
	switch {
	case p.Size < 1:
		return DefaultSize
	case p.Size > MaxSize:
		return MaxSize
	default:
		return p.Size
	}
}

// Limit returns the SQL LIMIT value for cursor pagination.
func (p CursorPagination) Limit() int {
	return p.GetSize()
}

// Page is the paginated response envelope returned by list endpoints.
//
//	func listUsers(...) (*pagination.Page[UserOut], error) {
//	    items, total, err := svc.Page(input.GetPage(), input.GetSize())
//	    return pagination.NewPage(items, total, input.PageInput), err
//	}
type Page[T any] struct {
	// Items contains the records for the current page.
	Items []T `json:"items"`
	// Total is the total number of matching records across all pages.
	Total int64 `json:"total"`
	// Page is the current 1-based page number.
	Page int `json:"page"`
	// Size is the number of items per page that was requested.
	Size int `json:"size"`
	// Pages is the total number of pages.
	Pages int `json:"pages"`
}

// NewPage constructs a Page response from a slice of items, the total count,
// and the original page input.
func NewPage[T any](items []T, total int64, input PageInput) *Page[T] {
	size := input.GetSize()
	pages := int(math.Ceil(float64(total) / float64(size)))
	if total > 0 && pages < 1 {
		pages = 1
	}
	if items == nil {
		items = []T{}
	}
	return &Page[T]{
		Items: items,
		Total: total,
		Page:  input.GetPage(),
		Size:  size,
		Pages: pages,
	}
}

// CursorPage is the cursor-paginated response envelope returned by list endpoints.
//
//	func listEvents(...) (*pagination.CursorPage[EventOut], error) {
//	    return pagination.NewCursorPage(items, input.CursorPagination, nextCursor), nil
//	}
type CursorPage[T any] struct {
	// Items contains the records for the current page.
	Items []T `json:"items"`
	// NextCursor is the opaque cursor clients can pass to fetch the next page.
	NextCursor string `json:"next_cursor,omitempty"`
	// PreviousCursor is the opaque cursor clients can pass to fetch the previous page.
	PreviousCursor string `json:"previous_cursor,omitempty"`
	// Size is the number of items per page that was requested.
	Size int `json:"size"`
	// HasNext indicates whether a next page is available.
	HasNext bool `json:"has_next"`
	// HasPrevious indicates whether a previous page is available.
	HasPrevious bool `json:"has_previous"`
}

// NewCursorPage constructs a cursor-paginated response from a slice of items,
// the original cursor input, and the next cursor. Pass previousCursor when
// backward pagination is supported.
func NewCursorPage[T any](items []T, input CursorPagination, nextCursor string, previousCursor ...string) *CursorPage[T] {
	if items == nil {
		items = []T{}
	}
	prev := ""
	if len(previousCursor) > 0 {
		prev = previousCursor[0]
	}
	return &CursorPage[T]{
		Items:          items,
		NextCursor:     nextCursor,
		PreviousCursor: prev,
		Size:           input.GetSize(),
		HasNext:        nextCursor != "",
		HasPrevious:    prev != "",
	}
}
