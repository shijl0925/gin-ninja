package ninja

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ModelSchemaOption customizes how a model is serialized.
type ModelSchemaOption func(*modelSchemaFilter)

// Fields limits serialization to the provided field names.
func Fields(fields ...string) ModelSchemaOption {
	return func(filter *modelSchemaFilter) {
		filter.fields = normalizeModelSchemaNames(fields)
	}
}

// Exclude removes the provided field names from serialization.
func Exclude(fields ...string) ModelSchemaOption {
	return func(filter *modelSchemaFilter) {
		filter.exclude = normalizeModelSchemaNames(fields)
	}
}

// ModelSchema wraps a model value and serializes only the allowed fields.
type ModelSchema[T any] struct {
	Model   T        `json:"-"`
	Fields  []string `json:"-"`
	Exclude []string `json:"-"`
}

// NewModelSchema wraps a model with optional field filters.
func NewModelSchema[T any](model T, opts ...ModelSchemaOption) *ModelSchema[T] {
	filter := modelSchemaFilter{}
	for _, opt := range opts {
		opt(&filter)
	}
	return &ModelSchema[T]{
		Model:   model,
		Fields:  append([]string(nil), filter.fields...),
		Exclude: append([]string(nil), filter.exclude...),
	}
}

// BindModelSchema creates a user-defined schema value from a model.
func BindModelSchema[TSchema any](model any) (*TSchema, error) {
	var zero TSchema
	t := reflect.TypeOf(zero)
	if t == nil {
		return nil, fmt.Errorf("model schema target type is nil")
	}
	if t.Kind() == reflect.Ptr {
		return nil, fmt.Errorf("model schema target must be a non-pointer struct")
	}
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model schema target must be a struct")
	}

	fieldInfo, ok := findEmbeddedModelSchemaField(t)
	if !ok {
		return nil, fmt.Errorf("%s does not embed ninja.ModelSchema", t.Name())
	}

	value := reflect.New(t).Elem()
	field := value.FieldByIndex(fieldInfo.index)
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}

	if err := assignModelSchemaModel(field.FieldByName("Model"), reflect.ValueOf(model)); err != nil {
		return nil, err
	}
	field.FieldByName("Fields").Set(reflect.ValueOf(append([]string(nil), fieldInfo.filter.fields...)))
	field.FieldByName("Exclude").Set(reflect.ValueOf(append([]string(nil), fieldInfo.filter.exclude...)))

	out := value.Addr().Interface().(*TSchema)
	return out, nil
}

func (m ModelSchema[T]) MarshalJSON() ([]byte, error) {
	filter := newModelSchemaFilter(m.Fields, m.Exclude)
	filtered, err := serializeModelSchemaValue(reflect.ValueOf(m.Model), filter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(filtered)
}

type modelSchemaFilter struct {
	fields  []string
	exclude []string
}

type modelSchemaDescriptor struct {
	target        reflect.Type
	filter        modelSchemaFilter
	componentName string
}

type embeddedModelSchemaField struct {
	index  []int
	filter modelSchemaFilter
}

type modelSchemaCarrier interface {
	modelSchemaDescriptor() modelSchemaDescriptor
}

func (m ModelSchema[T]) modelSchemaDescriptor() modelSchemaDescriptor {
	var zero T
	return modelSchemaDescriptor{
		target:        reflect.TypeOf(zero),
		filter:        newModelSchemaFilter(m.Fields, m.Exclude),
		componentName: sanitizeComponentName(typeName(reflect.TypeOf(m))),
	}
}

func newModelSchemaFilter(fields, exclude []string) modelSchemaFilter {
	return modelSchemaFilter{
		fields:  normalizeModelSchemaNames(fields),
		exclude: normalizeModelSchemaNames(exclude),
	}
}

func (f modelSchemaFilter) isZero() bool {
	return len(f.fields) == 0 && len(f.exclude) == 0
}

func (f modelSchemaFilter) includes(field reflect.StructField, jsonName string) bool {
	if !f.isZero() {
		if len(f.fields) > 0 && !containsModelSchemaName(f.fields, field.Name, jsonName) {
			return false
		}
		if containsModelSchemaName(f.exclude, field.Name, jsonName) {
			return false
		}
	}
	return true
}

func resolveModelSchemaDescriptor(t reflect.Type) (modelSchemaDescriptor, bool) {
	if t == nil {
		return modelSchemaDescriptor{}, false
	}
	s := deref(t)
	if s.Kind() == reflect.Struct {
		if embedded, ok := findEmbeddedModelSchemaField(s); ok {
			field := s.FieldByIndex(embedded.index)
			if descriptor, ok := directModelSchemaDescriptor(field.Type); ok {
				descriptor.filter = embedded.filter
				descriptor.componentName = sanitizeComponentName(typeName(s))
				return descriptor, true
			}
		}
	}
	return directModelSchemaDescriptor(t)
}

func directModelSchemaDescriptor(t reflect.Type) (modelSchemaDescriptor, bool) {
	for _, candidate := range modelSchemaCandidates(t) {
		if carrier, ok := candidate.(modelSchemaCarrier); ok {
			descriptor := carrier.modelSchemaDescriptor()
			if descriptor.target != nil {
				return descriptor, true
			}
		}
	}
	return modelSchemaDescriptor{}, false
}

func modelSchemaCandidates(t reflect.Type) []any {
	var candidates []any
	if t.Kind() == reflect.Ptr {
		candidates = append(candidates, reflect.New(t.Elem()).Interface())
		return candidates
	}
	candidates = append(candidates, reflect.New(t).Interface())
	if zero := reflect.Zero(t); zero.IsValid() && zero.CanInterface() {
		candidates = append(candidates, zero.Interface())
	}
	return candidates
}

func findEmbeddedModelSchemaField(t reflect.Type) (embeddedModelSchemaField, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			continue
		}
		if _, ok := directModelSchemaDescriptor(field.Type); ok {
			return embeddedModelSchemaField{
				index:  field.Index,
				filter: newModelSchemaFilter(parseModelSchemaTag(field.Tag.Get("fields")), parseModelSchemaTag(field.Tag.Get("exclude"))),
			}, true
		}
	}
	return embeddedModelSchemaField{}, false
}

func parseModelSchemaTag(raw string) []string {
	if raw == "" {
		return nil
	}
	return normalizeModelSchemaNames(strings.Split(raw, ","))
}

func normalizeModelSchemaNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func containsModelSchemaName(names []string, candidates ...string) bool {
	if len(names) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		index := sort.SearchStrings(names, candidate)
		if index < len(names) && names[index] == candidate {
			return true
		}
	}
	return false
}

func serializeModelSchemaValue(v reflect.Value, filter modelSchemaFilter) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}
	if marshaled, ok := preserveCustomJSONValue(v); ok {
		return marshaled, nil
	}
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}
		return serializeModelSchemaValue(v.Elem(), filter)
	case reflect.Slice, reflect.Array:
		items := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			item, err := serializeModelSchemaElement(v.Index(i), filter)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	case reflect.Struct:
		return serializeModelSchemaStruct(v, filter)
	default:
		return v.Interface(), nil
	}
}

func serializeModelSchemaElement(v reflect.Value, filter modelSchemaFilter) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}
	if marshaled, ok := preserveCustomJSONValue(v); ok {
		return marshaled, nil
	}
	if v.Kind() == reflect.Struct {
		return serializeModelSchemaStruct(v, filter)
	}
	return v.Interface(), nil
}

func preserveCustomJSONValue(v reflect.Value) (any, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if candidate, ok := customJSONValue(v); ok {
		return candidate, true
	}
	if v.Kind() != reflect.Ptr && v.CanAddr() {
		if candidate, ok := customJSONValue(v.Addr()); ok {
			return candidate, true
		}
	}
	if v.Kind() != reflect.Ptr && !v.CanAddr() {
		copy := reflect.New(v.Type()).Elem()
		copy.Set(v)
		if candidate, ok := customJSONValue(copy.Addr()); ok {
			return candidate, true
		}
	}
	return nil, false
}

func customJSONValue(v reflect.Value) (any, bool) {
	if !v.IsValid() || !v.CanInterface() {
		return nil, false
	}
	value := v.Interface()
	switch value.(type) {
	case json.Marshaler, encoding.TextMarshaler:
		return value, true
	default:
		return nil, false
	}
}

func serializeModelSchemaStruct(v reflect.Value, filter modelSchemaFilter) (map[string]any, error) {
	out := make(map[string]any)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		if field.Anonymous {
			embeddedValue := v.Field(i)
			for embeddedValue.Kind() == reflect.Ptr {
				if embeddedValue.IsNil() {
					embeddedValue = reflect.Value{}
					break
				}
				embeddedValue = embeddedValue.Elem()
			}
			if embeddedValue.IsValid() && embeddedValue.Kind() == reflect.Struct {
				embedded, err := serializeModelSchemaStruct(embeddedValue, filter)
				if err != nil {
					return nil, err
				}
				for key, value := range embedded {
					out[key] = value
				}
			}
			continue
		}

		name := jsonFieldName(field)
		if name == "-" || !filter.includes(field, name) {
			continue
		}

		value := v.Field(i)
		if isJSONOmitEmpty(field) && value.IsZero() {
			continue
		}
		if marshaled, ok := preserveCustomJSONValue(value); ok {
			out[name] = marshaled
			continue
		}
		out[name] = value.Interface()
	}
	return out, nil
}

func isJSONOmitEmpty(field reflect.StructField) bool {
	tag := field.Tag.Get("json")
	if tag == "" {
		return false
	}
	parts := strings.Split(tag, ",")
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			return true
		}
	}
	return false
}

func assignModelSchemaModel(dst, src reflect.Value) error {
	if !src.IsValid() {
		return fmt.Errorf("model value is invalid")
	}
	if src.Type().AssignableTo(dst.Type()) {
		dst.Set(src)
		return nil
	}
	if src.Type().ConvertibleTo(dst.Type()) {
		dst.Set(src.Convert(dst.Type()))
		return nil
	}
	if src.Kind() == reflect.Ptr && !src.IsNil() {
		elem := src.Elem()
		if elem.Type().AssignableTo(dst.Type()) {
			dst.Set(elem)
			return nil
		}
		if elem.Type().ConvertibleTo(dst.Type()) {
			dst.Set(elem.Convert(dst.Type()))
			return nil
		}
	}
	if dst.Kind() == reflect.Ptr {
		target := dst.Type().Elem()
		if src.Type().AssignableTo(target) || src.Type().ConvertibleTo(target) {
			ptr := reflect.New(target)
			if src.Type().AssignableTo(target) {
				ptr.Elem().Set(src)
			} else {
				ptr.Elem().Set(src.Convert(target))
			}
			dst.Set(ptr)
			return nil
		}
	}
	return fmt.Errorf("cannot assign model type %s to schema model type %s", src.Type(), dst.Type())
}
