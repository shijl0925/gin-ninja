// Package pagination provides reusable pagination types for gin-ninja handlers.
//
// Usage:
//
//	type ListUsersInput struct {
//	    pagination.PageInput
//	    Search string `form:"search"`
//	}
//
//	func listUsers(ctx *ninja.Context, input *ListUsersInput) (*pagination.Page[UserOut], error) {
//	    // service.Page(input.GetPage(), input.GetSize(), ...)
//	}
package pagination

import (
	"fmt"
	"math"
	"strings"

	"github.com/shijl0925/go-toolkits/gormx"
)

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
//	    Name string `form:"name"`
//	}
type PageInput struct {
	// Page is the 1-based page number (default: 1).
	Page int `form:"page" binding:"omitempty,min=1" json:"-"`
	// Size is the number of items per page (default: 20, max: 100).
	Size int `form:"size" binding:"omitempty,min=1,max=100" json:"-"`
	// Sort is a comma-separated list of fields. Prefix with "-" for descending.
	Sort string `form:"sort" json:"-"`
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
	return (p.GetPage() - 1) * p.GetSize()
}

// Limit returns the SQL LIMIT value (identical to GetSize).
func (p PageInput) Limit() int {
	return p.GetSize()
}

// SortField is a single parsed sort field.
type SortField struct {
	Name string
	Desc bool
}

// SortSchema defines the allowlist of accepted sort aliases.
type SortSchema struct {
	allowed map[string]string
}

// NewSortSchema creates a sort schema and allows each field as its own alias.
func NewSortSchema(fields ...string) *SortSchema {
	schema := &SortSchema{allowed: map[string]string{}}
	for _, field := range fields {
		schema.Allow(field)
	}
	return schema
}

// Allow adds a sort alias. If column is omitted, the alias itself is used.
func (s *SortSchema) Allow(alias string, column ...string) *SortSchema {
	if s == nil {
		return nil
	}
	if s.allowed == nil {
		s.allowed = map[string]string{}
	}
	target := alias
	if len(column) > 0 {
		trimmed := strings.TrimSpace(column[0])
		if trimmed != "" {
			target = trimmed
		}
	}
	s.allowed[strings.TrimSpace(alias)] = target
	return s
}

// GetSortFields parses the raw sort string.
func (p PageInput) GetSortFields() []SortField {
	if strings.TrimSpace(p.Sort) == "" {
		return nil
	}

	parts := strings.Split(p.Sort, ",")
	fields := make([]SortField, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}

		field := SortField{Name: part}
		prefix := part[0]
		switch prefix {
		case '-':
			field.Name = strings.TrimSpace(part[1:])
			field.Desc = true
		case '+':
			field.Name = strings.TrimSpace(part[1:])
		}
		if field.Name == "" {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

// ResolveSort validates the sort string against a schema.
func (p PageInput) ResolveSort(schema *SortSchema) ([]SortField, error) {
	fields := p.GetSortFields()
	if len(fields) == 0 {
		return nil, nil
	}
	if schema == nil || len(schema.allowed) == 0 {
		return nil, fmt.Errorf("sort schema is required when sort is provided")
	}

	resolved := make([]SortField, 0, len(fields))
	for _, field := range fields {
		column, ok := schema.allowed[field.Name]
		if !ok {
			return nil, fmt.Errorf("unsupported sort field %q", field.Name)
		}
		resolved = append(resolved, SortField{
			Name: column,
			Desc: field.Desc,
		})
	}
	return resolved, nil
}

// ApplySort validates and applies the sort string to a gormx query.
func ApplySort[T any](query *gormx.Query[T], input PageInput, schema *SortSchema) error {
	if query == nil {
		return nil
	}
	fields, err := input.ResolveSort(schema)
	if err != nil {
		return err
	}
	for _, field := range fields {
		if field.Desc {
			query.OrderDesc(field.Name)
			continue
		}
		query.OrderAsc(field.Name)
	}
	return nil
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
	if pages < 1 {
		pages = 1
	}
	return &Page[T]{
		Items: items,
		Total: total,
		Page:  input.GetPage(),
		Size:  size,
		Pages: pages,
	}
}
