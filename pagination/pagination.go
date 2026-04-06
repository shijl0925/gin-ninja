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
	"reflect"
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
	// Deprecated: prefer a struct level `order:"..."` tag with ApplyOrder/ResolveOrder.
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
	return ParseSort(p.Sort)
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

// ParseSort parses a raw comma-separated sort string.
func ParseSort(raw string) []SortField {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
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

// ResolveOrder resolves declarative sort clauses from `order:"..."` tags.
func ResolveOrder(input any) ([]SortField, error) {
	var resolved []SortField
	if err := resolveOrderInto(reflect.ValueOf(input), &resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}

// ApplyOrder validates and applies declarative sorting to a gormx query.
func ApplyOrder[T any](query *gormx.Query[T], input any) error {
	if query == nil {
		return nil
	}
	fields, err := ResolveOrder(input)
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

func resolveOrderInto(value reflect.Value, resolved *[]SortField) error {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("order input must be a struct or pointer to struct")
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		tag := strings.TrimSpace(field.Tag.Get("order"))
		if tag != "" && tag != "-" {
			current, ok, err := resolveTaggedOrder(field, fieldValue, tag)
			if err != nil {
				return err
			}
			if ok {
				*resolved = append(*resolved, current...)
			}
		}

		if field.Anonymous {
			if err := resolveOrderInto(fieldValue, resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveTaggedOrder(field reflect.StructField, value reflect.Value, tag string) ([]SortField, bool, error) {
	raw, ok, err := extractOrderRawValue(field, value)
	if err != nil || !ok {
		return nil, ok, err
	}

	schema, err := parseOrderTagSchema(tag, field)
	if err != nil {
		return nil, false, err
	}

	parsed := ParseSort(raw)
	if len(parsed) == 0 {
		return nil, false, nil
	}

	resolved := make([]SortField, 0, len(parsed))
	for _, sortField := range parsed {
		column, exists := schema.allowed[sortField.Name]
		if !exists {
			return nil, false, fmt.Errorf("unsupported sort field %q", sortField.Name)
		}
		resolved = append(resolved, SortField{
			Name: column,
			Desc: sortField.Desc,
		})
	}
	return resolved, true, nil
}

func extractOrderRawValue(field reflect.StructField, value reflect.Value) (string, bool, error) {
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return "", false, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.String:
		raw := strings.TrimSpace(value.String())
		if raw == "" {
			return "", false, nil
		}
		return raw, true, nil
	case reflect.Struct:
		sortField := value.FieldByName("Sort")
		sortStructField, ok := value.Type().FieldByName("Sort")
		if !ok || !sortStructField.IsExported() || !sortField.IsValid() || sortField.Kind() != reflect.String {
			return "", false, fmt.Errorf("order tag on %s requires a string field or struct with an exported Sort string field", field.Name)
		}
		raw := strings.TrimSpace(sortField.String())
		if raw == "" {
			return "", false, nil
		}
		return raw, true, nil
	default:
		return "", false, fmt.Errorf("order tag on %s requires a string field or struct with an exported Sort string field, got %s", field.Name, value.Kind())
	}
}

func parseOrderTagSchema(tag string, field reflect.StructField) (*SortSchema, error) {
	schema := &SortSchema{allowed: map[string]string{}}
	parts := strings.Split(tag, "|")
	for _, part := range parts {
		spec := strings.TrimSpace(part)
		if spec == "" {
			return nil, fmt.Errorf("order tag on %s contains an empty field name", field.Name)
		}

		alias, column := parseAliasAndColumn(spec)
		if alias == "" || column == "" {
			return nil, fmt.Errorf("order tag on %s contains an empty field name", field.Name)
		}
		schema.allowed[alias] = column
	}
	return schema, nil
}

func parseAliasAndColumn(spec string) (alias string, column string) {
	for _, separator := range []string{":", "="} {
		if alias, column, ok := strings.Cut(spec, separator); ok {
			alias = strings.TrimSpace(alias)
			column = strings.TrimSpace(column)
			if alias == "" || column == "" {
				return "", ""
			}
			return alias, column
		}
	}
	return spec, spec
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
	return &Page[T]{
		Items: items,
		Total: total,
		Page:  input.GetPage(),
		Size:  size,
		Pages: pages,
	}
}
