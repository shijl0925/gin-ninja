package ninja

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// JSON Schema generation
// ---------------------------------------------------------------------------

// Schema represents a JSON Schema object (OpenAPI 3.0 compatible subset).
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Default     interface{}        `json:"default,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Nullable    bool               `json:"nullable,omitempty"`
	Minimum     *float64           `json:"minimum,omitempty"`
	Maximum     *float64           `json:"maximum,omitempty"`
	MinLength   *int               `json:"minLength,omitempty"`
	MaxLength   *int               `json:"maxLength,omitempty"`
	Example     interface{}        `json:"example,omitempty"`
}

// schemaRegistry accumulates reusable component schemas to avoid duplication
// in the generated OpenAPI spec.
type schemaRegistry struct {
	schemas map[string]*Schema
}

func newSchemaRegistry() *schemaRegistry {
	return &schemaRegistry{schemas: make(map[string]*Schema)}
}

// schemaForType returns the JSON Schema for the given reflect.Type and, for
// named struct types, registers the schema in the registry so it can be
// referenced via $ref.
func (r *schemaRegistry) schemaForType(t reflect.Type) *Schema {
	return r.schemaForTypeWithFilter(t, modelSchemaFilter{})
}

func (r *schemaRegistry) schemaForTypeWithFilter(t reflect.Type, filter modelSchemaFilter) *Schema {
	if descriptor, ok := resolveModelSchemaDescriptor(t); ok {
		return r.schemaForDescriptor(descriptor)
	}

	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Invalid:
		return &Schema{}

	case reflect.Bool:
		return &Schema{Type: "boolean"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer", Format: intFormat(t.Kind())}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer", Format: uintFormat(t.Kind())}

	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}

	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}

	case reflect.String:
		return &Schema{Type: "string"}

	case reflect.Slice, reflect.Array:
		items := r.schemaForTypeWithFilter(t.Elem(), filter)
		return &Schema{Type: "array", Items: items}

	case reflect.Map:
		return &Schema{Type: "object"}

	case reflect.Struct:
		if isUploadedFileType(t) {
			return &Schema{Type: "string", Format: "binary"}
		}
		return r.schemaForDescriptor(modelSchemaDescriptor{
			target: t,
			filter: filter,
		})

	default:
		return &Schema{Type: "string"}
	}
}

func (r *schemaRegistry) schemaForDescriptor(descriptor modelSchemaDescriptor) *Schema {
	t := deref(descriptor.target)
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return r.schemaForTypeWithFilter(t, descriptor.filter)
	}
	name := descriptor.componentName
	if name == "" {
		name = modelSchemaComponentName(t, descriptor.filter)
	}
	if _, ok := r.schemas[name]; !ok {
		r.schemas[name] = &Schema{Type: "object"}
		r.schemas[name] = r.buildStructSchemaWithFilter(t, descriptor.filter)
	}
	return &Schema{Ref: fmt.Sprintf("#/components/schemas/%s", name)}
}

// buildStructSchema constructs the full Schema object for a struct type.
func (r *schemaRegistry) buildStructSchema(t reflect.Type) *Schema {
	return r.buildStructSchemaWithFilter(t, modelSchemaFilter{})
}

func (r *schemaRegistry) buildStructSchemaWithFilter(t reflect.Type, filter modelSchemaFilter) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Flatten embedded / anonymous structs.
		if f.Anonymous {
			embedded := r.buildStructSchemaWithFilter(deref(f.Type), filter)
			for k, v := range embedded.Properties {
				s.Properties[k] = v
			}
			s.Required = append(s.Required, embedded.Required...)
			continue
		}

		fieldName := jsonFieldName(f)
		if fieldName == "-" || !filter.includes(f, fieldName) {
			continue
		}

		fieldSchema := r.schemaForType(f.Type)

		// Copy so we can annotate without mutating the shared instance.
		s.Properties[fieldName] = annotateSchema(fieldSchema, f)

		// Mark as required if binding tag says so.
		if isRequired(f) {
			s.Required = append(s.Required, fieldName)
		}
	}

	return s
}

func modelSchemaComponentName(t reflect.Type, filter modelSchemaFilter) string {
	name := typeName(t)
	if filter.isZero() {
		return name
	}
	parts := []string{name}
	if len(filter.fields) > 0 {
		parts = append(parts, "fields", strings.Join(filter.fields, "_"))
	}
	if len(filter.exclude) > 0 {
		parts = append(parts, "exclude", strings.Join(filter.exclude, "_"))
	}
	return sanitizeComponentName(strings.Join(parts, "__"))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// typeName returns a clean, unique name suitable for use as an OpenAPI
// component schema key.
func typeName(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	if name == "" {
		name = t.String()
	}
	return sanitizeComponentName(name)
}

// jsonFieldName resolves the JSON key for a struct field using the `json` tag,
// falling back to the field name.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return defaultJSONFieldName(f.Name)
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return defaultJSONFieldName(f.Name)
	}
	return parts[0]
}

func defaultJSONFieldName(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	prefix := 1
	for prefix < len(runes) && unicode.IsUpper(runes[prefix]) {
		if prefix+1 < len(runes) && unicode.IsLower(runes[prefix+1]) {
			break
		}
		prefix++
	}
	for i := 0; i < prefix; i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

// isRequired returns true when the field has a `binding:"required"` constraint.
func isRequired(f reflect.StructField) bool {
	binding := f.Tag.Get("binding")
	for _, part := range strings.Split(binding, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}

// deref follows pointer indirections.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// intFormat returns the OpenAPI format string for integer kinds.
func intFormat(k reflect.Kind) string {
	switch k {
	case reflect.Int32, reflect.Uint32:
		return "int32"
	default:
		return "int64"
	}
}

func uintFormat(k reflect.Kind) string {
	switch k {
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	default:
		return "uint64"
	}
}

func annotateSchema(schema *Schema, f reflect.StructField) *Schema {
	annotated := *schema
	if desc := f.Tag.Get("description"); desc != "" {
		annotated.Description = desc
	}
	if example := f.Tag.Get("example"); example != "" {
		annotated.Example = example
	}
	if def, ok := defaultValueForField(f); ok {
		annotated.Default = def
	}
	return &annotated
}

func defaultValueForField(f reflect.StructField) (interface{}, bool) {
	raw := f.Tag.Get("default")
	if raw == "" {
		return nil, false
	}
	return defaultValueForType(deref(f.Type), raw)
}

func defaultValueForType(t reflect.Type, raw string) (interface{}, bool) {
	switch t.Kind() {
	case reflect.String:
		return raw, true
	case reflect.Bool:
		switch strings.ToLower(raw) {
		case "true":
			return true, true
		case "false":
			return false, true
		default:
			return nil, false
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var v int64
		if _, err := fmt.Sscan(raw, &v); err != nil {
			return nil, false
		}
		return v, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var v uint64
		if _, err := fmt.Sscan(raw, &v); err != nil {
			return nil, false
		}
		return v, true
	case reflect.Float32, reflect.Float64:
		var v float64
		if _, err := fmt.Sscan(raw, &v); err != nil {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}

func paginatedSchema(itemSchema *Schema) *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"items": {Type: "array", Items: itemSchema},
			"total": {Type: "integer", Format: "int64"},
			"page":  {Type: "integer", Format: "int64"},
			"size":  {Type: "integer", Format: "int64"},
			"pages": {Type: "integer", Format: "int64"},
		},
		Required: []string{"items", "total", "page", "size", "pages"},
	}
}

var invalidComponentNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeComponentName(name string) string {
	name = invalidComponentNameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_.-")
	if name == "" {
		return "Schema"
	}
	return name
}
