package ninja

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ModelSchemaOption customizes how a model is serialized.
type ModelSchemaOption func(*modelSchemaFilter)

// ModelSchemaMode controls which model fields are selected by convention.
type ModelSchemaMode string

const (
	// ModelSchemaModeRead includes fields that are safe to return in JSON.
	ModelSchemaModeRead ModelSchemaMode = "read"
	// ModelSchemaModeList includes readable scalar fields for list responses.
	ModelSchemaModeList ModelSchemaMode = "list"
	// ModelSchemaModeDetail includes readable fields for detail responses.
	ModelSchemaModeDetail ModelSchemaMode = "detail"
	// ModelSchemaModeCreate includes fields writable during create operations.
	ModelSchemaModeCreate ModelSchemaMode = "create"
	// ModelSchemaModeUpdate includes fields writable during update operations.
	ModelSchemaModeUpdate ModelSchemaMode = "update"
)

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

// SchemaMode applies a conventional field-selection mode.
func SchemaMode(mode ModelSchemaMode) ModelSchemaOption {
	return func(filter *modelSchemaFilter) {
		filter.mode = normalizeModelSchemaMode(mode)
	}
}

// Depth controls how many nested model levels are serialized through the same schema mode.
func Depth(depth int) ModelSchemaOption {
	return func(filter *modelSchemaFilter) {
		filter.depth = normalizeModelSchemaDepth(depth)
	}
}

// ModelSchemaDescriptor describes how a model type should be serialized.
type ModelSchemaDescriptor[T any] struct {
	filter        modelSchemaFilter
	componentName string
}

// ModelSchemaOf creates a reusable model schema descriptor.
func ModelSchemaOf[T any](opts ...ModelSchemaOption) ModelSchemaDescriptor[T] {
	descriptor := ModelSchemaDescriptor[T]{}
	for _, opt := range opts {
		opt(&descriptor.filter)
	}
	descriptor.filter = descriptor.filter.normalized()
	return descriptor
}

// Fields returns a copy of the descriptor limited to the provided field names.
func (d ModelSchemaDescriptor[T]) Fields(fields ...string) ModelSchemaDescriptor[T] {
	d.filter.fields = normalizeModelSchemaNames(fields)
	return d
}

// Exclude returns a copy of the descriptor excluding the provided field names.
func (d ModelSchemaDescriptor[T]) Exclude(fields ...string) ModelSchemaDescriptor[T] {
	d.filter.exclude = normalizeModelSchemaNames(fields)
	return d
}

// Mode returns a copy of the descriptor with a conventional field-selection mode.
func (d ModelSchemaDescriptor[T]) Mode(mode ModelSchemaMode) ModelSchemaDescriptor[T] {
	d.filter.mode = normalizeModelSchemaMode(mode)
	return d
}

// Read returns a copy of the descriptor using read-mode field selection.
func (d ModelSchemaDescriptor[T]) Read() ModelSchemaDescriptor[T] {
	return d.Mode(ModelSchemaModeRead)
}

// List returns a copy of the descriptor using list-mode field selection.
func (d ModelSchemaDescriptor[T]) List() ModelSchemaDescriptor[T] {
	return d.Mode(ModelSchemaModeList)
}

// Detail returns a copy of the descriptor using detail-mode field selection.
func (d ModelSchemaDescriptor[T]) Detail() ModelSchemaDescriptor[T] {
	return d.Mode(ModelSchemaModeDetail)
}

// Create returns a copy of the descriptor using create-mode field selection.
func (d ModelSchemaDescriptor[T]) Create() ModelSchemaDescriptor[T] {
	return d.Mode(ModelSchemaModeCreate)
}

// Update returns a copy of the descriptor using update-mode field selection.
func (d ModelSchemaDescriptor[T]) Update() ModelSchemaDescriptor[T] {
	return d.Mode(ModelSchemaModeUpdate)
}

// Depth returns a copy of the descriptor with nested model serialization depth.
func (d ModelSchemaDescriptor[T]) Depth(depth int) ModelSchemaDescriptor[T] {
	d.filter.depth = normalizeModelSchemaDepth(depth)
	return d
}

// Preloads returns GORM association preload paths implied by the descriptor depth.
func (d ModelSchemaDescriptor[T]) Preloads() []string {
	descriptor := d.schemaDescriptor()
	return modelSchemaPreloads(descriptor.target, descriptor.filter)
}

// ComponentName returns a copy of the descriptor with a fixed OpenAPI component name.
func (d ModelSchemaDescriptor[T]) ComponentName(name string) ModelSchemaDescriptor[T] {
	d.componentName = sanitizeComponentName(name)
	return d
}

// Wrap serializes model with the descriptor's field filters.
func (d ModelSchemaDescriptor[T]) Wrap(model T) *ModelSchema[T] {
	return &ModelSchema[T]{
		Model:   model,
		Fields:  append([]string(nil), d.filter.fields...),
		Exclude: append([]string(nil), d.filter.exclude...),
		Mode:    d.filter.mode,
		Depth:   d.filter.depth,
	}
}

func (d ModelSchemaDescriptor[T]) schemaDescriptor() modelSchemaDescriptor {
	return modelSchemaDescriptor{
		target:        reflect.TypeOf((*T)(nil)).Elem(),
		filter:        d.filter.normalized(),
		componentName: d.componentName,
	}
}

// ModelSchema wraps a model value and serializes only the allowed fields.
type ModelSchema[T any] struct {
	Model   T               `json:"-"`
	Fields  []string        `json:"-"`
	Exclude []string        `json:"-"`
	Mode    ModelSchemaMode `json:"-"`
	Depth   int             `json:"-"`
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
		Mode:    filter.mode,
		Depth:   filter.depth,
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
	field.FieldByName("Mode").Set(reflect.ValueOf(fieldInfo.filter.mode))
	field.FieldByName("Depth").Set(reflect.ValueOf(fieldInfo.filter.depth))

	out := value.Addr().Interface().(*TSchema)
	return out, nil
}

func bindModelSchemaForType(schemaType reflect.Type, model any) (any, bool, error) {
	t := deref(schemaType)
	if t.Kind() != reflect.Struct {
		return nil, false, nil
	}

	value := reflect.New(t).Elem()
	field, filter, ok := modelSchemaBindingField(value, t)
	if !ok {
		return nil, false, nil
	}
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}

	if err := assignModelSchemaModel(field.FieldByName("Model"), reflect.ValueOf(model)); err != nil {
		return nil, true, err
	}
	field.FieldByName("Fields").Set(reflect.ValueOf(append([]string(nil), filter.fields...)))
	field.FieldByName("Exclude").Set(reflect.ValueOf(append([]string(nil), filter.exclude...)))
	field.FieldByName("Mode").Set(reflect.ValueOf(filter.mode))
	field.FieldByName("Depth").Set(reflect.ValueOf(filter.depth))
	return value.Addr().Interface(), true, nil
}

func modelSchemaBindingField(value reflect.Value, t reflect.Type) (reflect.Value, modelSchemaFilter, bool) {
	if hasDirectModelSchemaFields(t) {
		return value, modelSchemaFilter{}, true
	}

	embedded, ok := findEmbeddedModelSchemaField(t)
	if !ok {
		return reflect.Value{}, modelSchemaFilter{}, false
	}
	return value.FieldByIndex(embedded.index), embedded.filter, true
}

func hasDirectModelSchemaFields(t reflect.Type) bool {
	fields := map[string]bool{
		"Model":   false,
		"Fields":  false,
		"Exclude": false,
		"Mode":    false,
		"Depth":   false,
	}
	for i := 0; i < t.NumField(); i++ {
		if _, ok := fields[t.Field(i).Name]; ok {
			fields[t.Field(i).Name] = true
		}
	}
	for _, ok := range fields {
		if !ok {
			return false
		}
	}
	return true
}

func (m ModelSchema[T]) MarshalJSON() ([]byte, error) {
	filter := newModelSchemaFilter(m.Fields, m.Exclude)
	filter.mode = normalizeModelSchemaMode(m.Mode)
	filter.depth = normalizeModelSchemaDepth(m.Depth)
	filtered, err := serializeModelSchemaValue(reflect.ValueOf(m.Model), filter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(filtered)
}

type modelSchemaFilter struct {
	fields  []string
	exclude []string
	mode    ModelSchemaMode
	depth   int
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
	filter := newModelSchemaFilter(m.Fields, m.Exclude)
	filter.mode = normalizeModelSchemaMode(m.Mode)
	filter.depth = normalizeModelSchemaDepth(m.Depth)
	return modelSchemaDescriptor{
		target:        reflect.TypeOf(zero),
		filter:        filter,
		componentName: sanitizeComponentName(typeName(reflect.TypeOf(m))),
	}
}

func newModelSchemaFilter(fields, exclude []string) modelSchemaFilter {
	return modelSchemaFilter{
		fields:  normalizeModelSchemaNames(fields),
		exclude: normalizeModelSchemaNames(exclude),
	}
}

func (f modelSchemaFilter) normalized() modelSchemaFilter {
	return modelSchemaFilter{
		fields:  normalizeModelSchemaNames(f.fields),
		exclude: normalizeModelSchemaNames(f.exclude),
		mode:    normalizeModelSchemaMode(f.mode),
		depth:   normalizeModelSchemaDepth(f.depth),
	}
}

func (f modelSchemaFilter) isZero() bool {
	return len(f.fields) == 0 && len(f.exclude) == 0 && f.mode == "" && f.depth == 0
}

func (f modelSchemaFilter) includes(field reflect.StructField, jsonName string) bool {
	if len(f.fields) > 0 && !containsModelSchemaName(f.fields, field.Name, jsonName) {
		return false
	}
	if containsModelSchemaName(f.exclude, field.Name, jsonName) {
		return false
	}
	return modelSchemaModeIncludes(f.mode, field, jsonName)
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
				index: field.Index,
				filter: newModelSchemaFilter(
					parseModelSchemaTag(field.Tag.Get("fields")),
					parseModelSchemaTag(field.Tag.Get("exclude")),
				).withMode(parseModelSchemaModeTag(field.Tag.Get("mode"))).
					withDepth(parseModelSchemaDepthTag(field.Tag.Get("depth"))),
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

func parseModelSchemaModeTag(raw string) ModelSchemaMode {
	return normalizeModelSchemaMode(ModelSchemaMode(raw))
}

func parseModelSchemaDepthTag(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	depth, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return normalizeModelSchemaDepth(depth)
}

func normalizeModelSchemaMode(mode ModelSchemaMode) ModelSchemaMode {
	value := strings.TrimSpace(strings.ToLower(string(mode)))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "", "none":
		return ""
	case "read", "output":
		return ModelSchemaModeRead
	case "list":
		return ModelSchemaModeList
	case "detail", "details":
		return ModelSchemaModeDetail
	case "create", "creation", "insert":
		return ModelSchemaModeCreate
	case "update", "patch":
		return ModelSchemaModeUpdate
	default:
		return ModelSchemaMode(value)
	}
}

func normalizeModelSchemaDepth(depth int) int {
	if depth < 0 {
		return 0
	}
	return depth
}

func (f modelSchemaFilter) withMode(mode ModelSchemaMode) modelSchemaFilter {
	f.mode = normalizeModelSchemaMode(mode)
	return f
}

func (f modelSchemaFilter) withDepth(depth int) modelSchemaFilter {
	f.depth = normalizeModelSchemaDepth(depth)
	return f
}

func (f modelSchemaFilter) child() modelSchemaFilter {
	if f.depth <= 0 {
		return modelSchemaFilter{}
	}
	return modelSchemaFilter{
		mode:  normalizeModelSchemaMode(f.mode),
		depth: f.depth - 1,
	}
}

func modelSchemaModeIncludes(mode ModelSchemaMode, field reflect.StructField, jsonName string) bool {
	switch normalizeModelSchemaMode(mode) {
	case "":
		return true
	case ModelSchemaModeRead, ModelSchemaModeDetail:
		return modelSchemaReadableField(field)
	case ModelSchemaModeList:
		return modelSchemaReadableField(field) && modelSchemaListField(field.Type)
	case ModelSchemaModeCreate:
		return modelSchemaWritableField(field, jsonName, ModelSchemaModeCreate)
	case ModelSchemaModeUpdate:
		return modelSchemaWritableField(field, jsonName, ModelSchemaModeUpdate)
	default:
		return true
	}
}

type modelSchemaFieldAccess struct {
	writeOnly  bool
	createOnly bool
	updateOnly bool
}

func modelSchemaReadableField(field reflect.StructField) bool {
	access := modelSchemaResolveFieldAccess(field.Tag)
	return !access.writeOnly
}

func modelSchemaWritableField(field reflect.StructField, jsonName string, mode ModelSchemaMode) bool {
	if modelSchemaReadOnlyField(field, jsonName) {
		return false
	}
	gormTag := field.Tag.Get("gorm")
	if mode == ModelSchemaModeCreate && modelSchemaHasGORMToken(gormTag, "<-:update") {
		return false
	}
	if mode == ModelSchemaModeUpdate && modelSchemaHasGORMToken(gormTag, "<-:create") {
		return false
	}
	access := modelSchemaResolveFieldAccess(field.Tag)
	switch normalizeModelSchemaMode(mode) {
	case ModelSchemaModeCreate:
		return !modelSchemaHasWriteModeOverride(access) || access.createOnly
	case ModelSchemaModeUpdate:
		return !modelSchemaHasWriteModeOverride(access) || access.updateOnly
	default:
		return true
	}
}

func modelSchemaHasWriteModeOverride(access modelSchemaFieldAccess) bool {
	return access.createOnly || access.updateOnly
}

func modelSchemaResolveFieldAccess(tag reflect.StructTag) modelSchemaFieldAccess {
	var access modelSchemaFieldAccess
	modelSchemaApplyFieldAccessTag(&access, tag.Get("ninja"))
	modelSchemaApplyFieldAccessTag(&access, tag.Get("crud"))
	return access
}

func modelSchemaApplyFieldAccessTag(access *modelSchemaFieldAccess, raw string) {
	for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	}) {
		switch modelSchemaNormalizeAccessToken(token) {
		case "writeonly":
			access.writeOnly = true
		case "createonly":
			access.createOnly = true
		case "updateonly":
			access.updateOnly = true
		}
	}
}

func modelSchemaNormalizeAccessToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	token = strings.ReplaceAll(token, "_", "")
	token = strings.ReplaceAll(token, "-", "")
	return token
}

func modelSchemaReadOnlyField(field reflect.StructField, jsonName string) bool {
	gormTag := field.Tag.Get("gorm")
	if modelSchemaHasGORMToken(gormTag, "-") ||
		modelSchemaHasGORMToken(gormTag, "primarykey") ||
		modelSchemaHasGORMToken(gormTag, "primary_key") ||
		modelSchemaHasGORMToken(gormTag, "->") ||
		modelSchemaHasGORMToken(gormTag, "<-:false") ||
		modelSchemaHasGORMToken(gormTag, "autocreatetime") ||
		modelSchemaHasGORMToken(gormTag, "autoupdatetime") {
		return true
	}
	if field.Name == "ID" || jsonName == "id" {
		return true
	}
	switch field.Name {
	case "CreatedAt", "UpdatedAt", "DeletedAt":
		return true
	default:
		return false
	}
}

func modelSchemaHasGORMToken(raw, token string) bool {
	token = modelSchemaNormalizeGORMToken(token)
	for _, part := range strings.Split(raw, ";") {
		if modelSchemaNormalizeGORMToken(part) == token {
			return true
		}
	}
	return false
}

func modelSchemaHasGORMSetting(raw, key string) bool {
	key = modelSchemaNormalizeGORMToken(key)
	for _, part := range strings.Split(raw, ";") {
		name, _, _ := strings.Cut(part, ":")
		if modelSchemaNormalizeGORMToken(name) == key {
			return true
		}
	}
	return false
}

func modelSchemaNormalizeGORMToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	token = strings.ReplaceAll(token, "_", "")
	return token
}

func modelSchemaListField(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if modelSchemaImplementsMarshaler(t) {
		return true
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}

func modelSchemaImplementsMarshaler(t reflect.Type) bool {
	jsonMarshalerType := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType := reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	if t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType) {
		return true
	}
	if t.Kind() != reflect.Ptr {
		ptr := reflect.PointerTo(t)
		return ptr.Implements(jsonMarshalerType) || ptr.Implements(textMarshalerType)
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
		// Values copied out of interfaces are not addressable, but pointer-receiver
		// marshalers still need an addressable value to preserve their JSON encoding.
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
		if filter.depth > 0 && modelSchemaNestedField(value.Type()) {
			nested, err := serializeModelSchemaValue(value, filter.child())
			if err != nil {
				return nil, err
			}
			out[name] = nested
			continue
		}
		out[name] = value.Interface()
	}
	return out, nil
}

func modelSchemaNestedField(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		return !modelSchemaImplementsMarshaler(t)
	case reflect.Slice, reflect.Array:
		return modelSchemaNestedField(t.Elem())
	default:
		return false
	}
}

func modelSchemaPreloads(t reflect.Type, filter modelSchemaFilter) []string {
	filter = filter.normalized()
	if filter.depth <= 0 {
		return nil
	}
	var paths []string
	collectModelSchemaPreloads(deref(t), filter, "", &paths)
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func collectModelSchemaPreloads(t reflect.Type, filter modelSchemaFilter, prefix string, paths *[]string) {
	t = modelSchemaRelationType(t)
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous {
			collectModelSchemaPreloads(field.Type, filter, prefix, paths)
			continue
		}

		name := jsonFieldName(field)
		if name == "-" || !filter.includes(field, name) || !modelSchemaPreloadField(field) {
			continue
		}

		path := field.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		*paths = appendModelSchemaPreloadPath(*paths, path)

		child := filter.child()
		if child.depth > 0 {
			collectModelSchemaPreloads(field.Type, child, path, paths)
		}
	}
}

func modelSchemaPreloadField(field reflect.StructField) bool {
	if !modelSchemaNestedField(field.Type) {
		return false
	}
	gormTag := field.Tag.Get("gorm")
	return !modelSchemaHasGORMToken(gormTag, "-") &&
		!modelSchemaHasGORMSetting(gormTag, "embedded") &&
		!modelSchemaHasGORMSetting(gormTag, "embeddedprefix")
}

func modelSchemaRelationType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	return t
}

func appendModelSchemaPreloadPath(paths []string, path string) []string {
	if path == "" {
		return paths
	}
	for _, existing := range paths {
		if existing == path {
			return paths
		}
	}
	return append(paths, path)
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
