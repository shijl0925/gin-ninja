package admin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jinzhu/inflection"
	"gorm.io/gorm"
)

type fieldMode string

const (
	fieldModeList   fieldMode = "list"
	fieldModeDetail fieldMode = "detail"
	fieldModeCreate fieldMode = "create"
	fieldModeUpdate fieldMode = "update"
	fieldModeFilter fieldMode = "filter"
	fieldModeSort   fieldMode = "sort"
	fieldModeSearch fieldMode = "search"
)

type fieldMeta struct {
	Meta              FieldMeta
	index             []int
	fieldType         reflect.Type
	timeField         bool
	persisted         bool
	primaryKey        bool
	componentExplicit bool
	autoRelation      *autoRelationMeta
}

type autoRelationMeta struct {
	targetType reflect.Type
}

type fieldAccess struct {
	WriteOnly  bool
	CreateOnly bool
	UpdateOnly bool
}

func (r *Resource) prepare() error {
	if r.Model == nil {
		return fmt.Errorf("admin resource model must not be nil")
	}

	r.modelType = reflect.TypeOf(r.Model)
	for r.modelType.Kind() == reflect.Ptr {
		r.modelType = r.modelType.Elem()
	}
	if r.modelType.Kind() != reflect.Struct {
		return fmt.Errorf("admin resource %q model must be a struct or pointer to struct", r.Name)
	}
	r.Name = firstNonEmpty(r.Name, inferResourceName(r.modelType))
	if r.Name == "" {
		return fmt.Errorf("admin resource name must not be empty")
	}

	r.Label = firstNonEmpty(r.Label, humanize(r.Name))
	r.Path = normalizePath(firstNonEmpty(r.Path, "/"+toKebab(r.Name)))
	r.fields = collectFields(r.modelType, nil, r.FieldOptions)
	if len(r.fields) == 0 {
		return fmt.Errorf("admin resource %q has no exported fields", r.Name)
	}
	r.fieldByName = map[string]*fieldMeta{}
	for _, field := range r.fields {
		r.fieldByName[field.Meta.Name] = field
		if r.primaryKey == nil && isPrimaryKeyField(field) {
			r.primaryKey = field
		}
	}
	if r.primaryKey == nil {
		return fmt.Errorf("admin resource %q requires an id-compatible primary key field", r.Name)
	}

	if err := applyFieldSet(r.fields, r.ListFields, fieldModeList); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.DetailFields, fieldModeDetail); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.CreateFields, fieldModeCreate); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.UpdateFields, fieldModeUpdate); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.FilterFields, fieldModeFilter); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.SortFields, fieldModeSort); err != nil {
		return err
	}
	if err := applyFieldSet(r.fields, r.SearchFields, fieldModeSearch); err != nil {
		return err
	}

	r.metadata = ResourceMetadata{
		Name:         r.Name,
		Label:        r.Label,
		Path:         r.Path,
		Fields:       make([]FieldMeta, 0, len(r.fields)),
		ListFields:   visibleFields(r.fields, fieldModeList),
		DetailFields: visibleFields(r.fields, fieldModeDetail),
		CreateFields: visibleFields(r.fields, fieldModeCreate),
		UpdateFields: visibleFields(r.fields, fieldModeUpdate),
		FilterFields: visibleFields(r.fields, fieldModeFilter),
		SortFields:   visibleFields(r.fields, fieldModeSort),
		SearchFields: visibleFields(r.fields, fieldModeSearch),
	}
	r.syncMetadataFields()

	actions := []Action{ActionList, ActionDetail}
	actions = appendAction(actions, ActionCreate, anyWritable(r.fields, fieldModeCreate))
	actions = appendAction(actions, ActionUpdate, anyWritable(r.fields, fieldModeUpdate))
	actions = appendAction(actions, ActionDelete, r.primaryKey != nil)
	actions = appendAction(actions, ActionBulkDelete, r.primaryKey != nil)
	r.metadata.Actions = actions
	return nil
}

func (r *Resource) syncMetadataFields() {
	if r == nil {
		return
	}
	r.metadata.Fields = r.metadata.Fields[:0]
	for _, field := range r.fields {
		r.metadata.Fields = append(r.metadata.Fields, cloneFieldMetaValue(field.Meta))
	}
}

func collectFields(t reflect.Type, prefix []int, overrides map[string]FieldOptions) []*fieldMeta {
	var out []*fieldMeta
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() && !field.Anonymous {
			continue
		}
		index := append(append([]int(nil), prefix...), i)

		derefType := field.Type
		for derefType.Kind() == reflect.Ptr {
			derefType = derefType.Elem()
		}
		if field.Anonymous && derefType.Kind() == reflect.Struct && derefType != reflect.TypeOf(time.Time{}) {
			out = append(out, collectFields(derefType, index, overrides)...)
			continue
		}

		meta := buildFieldMeta(field, index)
		if meta == nil {
			continue
		}
		applyFieldOptions(meta, overrides[meta.Meta.Name])
		if meta.Meta.Relation == nil {
			inferAutoRelation(meta, field, t)
		}
		out = append(out, meta)
	}
	return out
}

func buildFieldMeta(field reflect.StructField, index []int) *fieldMeta {
	name, hiddenByJSON := adminFieldName(field)
	if name == "" {
		return nil
	}

	gormTag := parseTagSettings(field.Tag.Get("gorm"))
	adminTag := parseTagSettings(field.Tag.Get("admin"))
	access := resolveFieldAccess(field.Tag)
	description := strings.TrimSpace(field.Tag.Get("description"))
	fieldType := indirectType(field.Type)
	typeName, component := inferFieldType(field, fieldType)
	readOnly := hiddenByJSON || isReadOnlyField(field, gormTag)
	hiddenFromRead := hiddenByJSON || access.WriteOnly
	sensitive := isSensitiveField(field, hiddenFromRead)
	writable := !readOnly && !hiddenByJSON && isWritableField(fieldType)

	meta := &fieldMeta{
		Meta: FieldMeta{
			Name:        name,
			Label:       firstNonEmpty(adminTag["label"], humanize(name)),
			Type:        typeName,
			Component:   firstNonEmpty(adminTag["component"], component),
			Column:      firstNonEmpty(gormTag["column"], toSnake(field.Name)),
			Description: description,
			Required:    isRequired(field, gormTag, readOnly),
			Unique:      hasTagFlag(gormTag, "unique") || hasTagFlag(gormTag, "uniqueindex"),
			ReadOnly:    readOnly,
			List:        !hiddenFromRead && !sensitive && isScalarField(fieldType),
			Detail:      !hiddenFromRead && !sensitive,
			Create:      writable && allowCreateField(access),
			Update:      writable && allowUpdateField(access),
			Filterable:  !hiddenFromRead && !sensitive && isFilterableField(fieldType),
			Sortable:    !hiddenFromRead && !sensitive && isSortableField(fieldType),
			Searchable:  !hiddenFromRead && !sensitive && fieldType.Kind() == reflect.String,
			Default:     strings.TrimSpace(gormTag["default"]),
		},
		index:             index,
		fieldType:         fieldType,
		timeField:         fieldType == reflect.TypeOf(time.Time{}),
		persisted:         !hasTagFlag(gormTag, "-"),
		primaryKey:        hasTagFlag(gormTag, "primarykey") || field.Name == "ID",
		componentExplicit: strings.TrimSpace(adminTag["component"]) != "",
	}

	if meta.Meta.Name == "deletedAt" {
		meta.Meta.Detail = false
		meta.Meta.List = false
		meta.Meta.Filterable = false
		meta.Meta.Sortable = false
	}
	if meta.Meta.Type == "boolean" {
		meta.Meta.Enum = []any{true, false}
	}
	applyAdminTag(meta, adminTag)
	meta.Meta.Required = isRequired(field, gormTag, meta.Meta.ReadOnly)
	return meta
}

func isPrimaryKeyField(field *fieldMeta) bool {
	if field == nil {
		return false
	}
	if field.primaryKey {
		return true
	}
	return field.Meta.ReadOnly && (field.Meta.Name == "id" || strings.EqualFold(field.Meta.Column, "id"))
}

func resolveFieldAccess(tag reflect.StructTag) fieldAccess {
	var access fieldAccess
	applyFieldAccessTag(&access, tag.Get("ninja"))
	applyFieldAccessTag(&access, tag.Get("crud"))
	return access
}

func applyFieldAccessTag(access *fieldAccess, raw string) {
	for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	}) {
		switch normalizeFieldAccessToken(token) {
		case "writeonly":
			access.WriteOnly = true
		case "createonly":
			access.CreateOnly = true
		case "updateonly":
			access.UpdateOnly = true
		}
	}
}

func normalizeFieldAccessToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	token = strings.ReplaceAll(token, "_", "")
	token = strings.ReplaceAll(token, "-", "")
	return token
}

func allowCreateField(access fieldAccess) bool {
	return !hasWriteModeOverride(access) || access.CreateOnly
}

func allowUpdateField(access fieldAccess) bool {
	return !hasWriteModeOverride(access) || access.UpdateOnly
}

func hasWriteModeOverride(access fieldAccess) bool {
	return access.CreateOnly || access.UpdateOnly
}

func applyFieldOptions(meta *fieldMeta, opts FieldOptions) {
	if meta == nil {
		return
	}
	if opts.Label != "" {
		meta.Meta.Label = opts.Label
	}
	if opts.Component != "" {
		meta.Meta.Component = opts.Component
		meta.componentExplicit = true
	}
	if len(opts.Enum) > 0 {
		meta.Meta.Enum = cloneSlice(opts.Enum)
	}
	if opts.Relation != nil {
		valueField := strings.TrimSpace(firstNonEmpty(opts.Relation.ValueField, meta.Meta.Name))
		meta.Meta.Relation = &RelationMeta{
			Resource:     strings.TrimSpace(opts.Relation.Resource),
			ValueField:   valueField,
			LabelField:   strings.TrimSpace(firstNonEmpty(opts.Relation.LabelField, valueField)),
			SearchFields: cloneSlice(opts.Relation.SearchFields),
		}
		if opts.Component == "" {
			meta.Meta.Component = "select"
		}
	}
	applyBoolOverride(&meta.Meta.ReadOnly, opts.ReadOnly)
	applyBoolOverride(&meta.Meta.List, opts.List)
	applyBoolOverride(&meta.Meta.Detail, opts.Detail)
	applyBoolOverride(&meta.Meta.Create, opts.Create)
	applyBoolOverride(&meta.Meta.Update, opts.Update)
	applyBoolOverride(&meta.Meta.Filterable, opts.Filterable)
	applyBoolOverride(&meta.Meta.Sortable, opts.Sortable)
	applyBoolOverride(&meta.Meta.Searchable, opts.Searchable)
	if opts.Hidden != nil && *opts.Hidden {
		meta.Meta.List = false
		meta.Meta.Detail = false
		meta.Meta.Create = false
		meta.Meta.Update = false
		meta.Meta.Filterable = false
		meta.Meta.Sortable = false
		meta.Meta.Searchable = false
	}
}

func inferAutoRelation(meta *fieldMeta, field reflect.StructField, owner reflect.Type) {
	if meta == nil || strings.TrimSpace(field.Name) == "" {
		return
	}
	baseName, ok := relationFieldBaseName(field.Name)
	if !ok {
		return
	}
	relatedField, found := owner.FieldByName(baseName)
	if !found || relatedField.Anonymous {
		return
	}
	relatedType := indirectType(relatedField.Type)
	if !isRelationEligibleType(relatedType) {
		return
	}
	meta.autoRelation = &autoRelationMeta{targetType: relatedType}
	meta.Meta.Relation = &RelationMeta{ValueField: "id"}
	if !meta.componentExplicit {
		meta.Meta.Component = "select"
	}
}

func isRelationEligibleType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{}) && t != reflect.TypeOf(gorm.DeletedAt{})
}

func relationFieldBaseName(name string) (string, bool) {
	for _, suffix := range []string{"ID", "Id"} {
		if baseName, ok := strings.CutSuffix(name, suffix); ok && strings.TrimSpace(baseName) != "" {
			return baseName, true
		}
	}
	return "", false
}

func inferRelationLabelField(resource *Resource) string {
	if resource == nil {
		return "id"
	}
	for _, name := range []string{"name", "title", "code", "email"} {
		if field := resource.fieldByName[name]; field != nil && field.fieldType.Kind() == reflect.String {
			return name
		}
	}
	for _, field := range resource.fields {
		if field != nil && field.fieldType.Kind() == reflect.String {
			return field.Meta.Name
		}
	}
	if resource.primaryKey != nil {
		return resource.primaryKey.Meta.Name
	}
	return "id"
}

func inferRelationSearchFields(resource *Resource, labelField string) []string {
	if resource == nil {
		return nil
	}
	var names []string
	added := map[string]struct{}{}
	add := func(name string) {
		if strings.TrimSpace(name) == "" {
			return
		}
		field := resource.fieldByName[name]
		if field == nil || field.fieldType.Kind() != reflect.String {
			return
		}
		if _, exists := added[name]; exists {
			return
		}
		added[name] = struct{}{}
		names = append(names, name)
	}
	add(labelField)
	for _, name := range []string{"name", "title", "code", "email"} {
		add(name)
	}
	return names
}

func applyAdminTag(meta *fieldMeta, settings map[string]string) {
	if meta == nil || len(settings) == 0 {
		return
	}
	if relation := relationMetaFromTag(settings); relation != nil {
		meta.Meta.Relation = relation
		if !meta.componentExplicit {
			meta.Meta.Component = "select"
		}
	}
	if hasTagFlag(settings, "hidden") || settings["-"] != "" || hasTagFlag(settings, "omit") {
		hidden := true
		applyFieldOptions(meta, FieldOptions{Hidden: &hidden})
	}
	if value, ok := settings["readonly"]; ok && value != "" {
		readOnly := value != "false"
		meta.Meta.ReadOnly = readOnly
	}
	for key, target := range map[string]*bool{
		"list":       &meta.Meta.List,
		"detail":     &meta.Meta.Detail,
		"create":     &meta.Meta.Create,
		"update":     &meta.Meta.Update,
		"filter":     &meta.Meta.Filterable,
		"sortable":   &meta.Meta.Sortable,
		"search":     &meta.Meta.Searchable,
		"filterable": &meta.Meta.Filterable,
		"sort":       &meta.Meta.Sortable,
	} {
		if value, ok := settings[key]; ok {
			*target = value != "false"
		}
	}
}

func relationMetaFromTag(settings map[string]string) *RelationMeta {
	resource := strings.TrimSpace(firstNonEmpty(settings["relation"], settings["relation_resource"]))
	if resource == "" || resource == "true" {
		return nil
	}
	valueField := strings.TrimSpace(firstNonEmpty(settings["relation_value"], settings["relation_value_field"], "id"))
	labelField := strings.TrimSpace(firstNonEmpty(settings["relation_label"], settings["relation_label_field"]))
	return &RelationMeta{
		Resource:     resource,
		ValueField:   valueField,
		LabelField:   labelField,
		SearchFields: splitTagList(firstNonEmpty(settings["relation_search"], settings["relation_search_fields"])),
	}
}

func splitTagList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (f *fieldMeta) allowed(mode fieldMode) bool {
	switch mode {
	case fieldModeList:
		return f.Meta.List
	case fieldModeDetail:
		return f.Meta.Detail
	case fieldModeCreate:
		return f.Meta.Create
	case fieldModeUpdate:
		return f.Meta.Update
	case fieldModeFilter:
		return f.Meta.Filterable
	case fieldModeSort:
		return f.Meta.Sortable
	case fieldModeSearch:
		return f.Meta.Searchable
	default:
		return false
	}
}

func (f *fieldMeta) value(v reflect.Value) any {
	current := v
	for _, index := range f.index {
		current = current.Field(index)
	}
	for current.Kind() == reflect.Ptr {
		if current.IsNil() {
			return nil
		}
		current = current.Elem()
	}
	return current.Interface()
}

func (f *fieldMeta) setValue(target reflect.Value, value any) error {
	current := target
	for i, index := range f.index {
		current = current.Field(index)
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}
		if i == len(f.index)-1 {
			if value == nil {
				current.Set(reflect.Zero(current.Type()))
				return nil
			}
			converted := reflect.ValueOf(value)
			if !converted.IsValid() {
				current.Set(reflect.Zero(current.Type()))
				return nil
			}
			if converted.Type().AssignableTo(current.Type()) {
				current.Set(converted)
				return nil
			}
			if converted.Type().ConvertibleTo(current.Type()) {
				current.Set(converted.Convert(current.Type()))
				return nil
			}
			return fmt.Errorf("cannot assign %T to %s", value, current.Type())
		}
	}
	return nil
}

func (f *fieldMeta) decodeJSON(raw json.RawMessage) (any, error) {
	if bytesEqualFoldSpace(raw, []byte("null")) {
		return nil, nil
	}
	holder := reflect.New(f.fieldType)
	if err := json.Unmarshal(raw, holder.Interface()); err != nil {
		return nil, err
	}
	return holder.Elem().Interface(), nil
}

func (f *fieldMeta) parseString(raw string) (any, error) {
	if f.timeField {
		return parseFlexibleTime(raw)
	}
	switch f.fieldType.Kind() {
	case reflect.String:
		return raw, nil
	case reflect.Bool:
		return strconv.ParseBool(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		out := reflect.New(f.fieldType).Elem()
		out.SetInt(value)
		return out.Interface(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		out := reflect.New(f.fieldType).Elem()
		out.SetUint(value)
		return out.Interface(), nil
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, err
		}
		out := reflect.New(f.fieldType).Elem()
		out.SetFloat(value)
		return out.Interface(), nil
	default:
		return nil, fmt.Errorf("unsupported filter type %s", f.fieldType)
	}
}

func adminFieldName(field reflect.StructField) (string, bool) {
	jsonTag := strings.TrimSpace(field.Tag.Get("json"))
	if jsonTag == "" {
		return defaultJSONFieldName(field.Name), false
	}
	parts := strings.Split(jsonTag, ",")
	if parts[0] == "-" {
		return defaultJSONFieldName(field.Name), true
	}
	if parts[0] == "" {
		return defaultJSONFieldName(field.Name), false
	}
	return parts[0], false
}

func parseTagSettings(raw string) map[string]string {
	settings := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			settings[strings.ToLower(part)] = "true"
			continue
		}
		settings[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return settings
}

func hasTagFlag(settings map[string]string, key string) bool {
	value, ok := settings[strings.ToLower(key)]
	return ok && value != "false"
}

func inferFieldType(field reflect.StructField, t reflect.Type) (string, string) {
	if t == reflect.TypeOf(time.Time{}) {
		return "datetime", "datetime"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean", "checkbox"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer", "number"
	case reflect.Float32, reflect.Float64:
		return "number", "number"
	case reflect.String:
		lower := strings.ToLower(field.Name)
		gormType := strings.ToLower(parseTagSettings(field.Tag.Get("gorm"))["type"])
		switch {
		case strings.Contains(lower, "email") || strings.Contains(field.Tag.Get("binding"), "email"):
			return "string", "email"
		case isSensitiveField(field, false):
			return "string", "password"
		case strings.Contains(gormType, "text"):
			return "string", "textarea"
		default:
			return "string", "text"
		}
	case reflect.Slice:
		return "array", "array"
	default:
		return "object", "text"
	}
}

func isReadOnlyField(field reflect.StructField, gormTag map[string]string) bool {
	name := field.Name
	return hasTagFlag(gormTag, "primarykey") || name == "ID" || name == "CreatedAt" || name == "UpdatedAt" || name == "DeletedAt"
}

func isRequired(field reflect.StructField, gormTag map[string]string, readOnly bool) bool {
	if readOnly {
		return false
	}
	binding := strings.Split(field.Tag.Get("binding"), ",")
	for _, part := range binding {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return hasTagFlag(gormTag, "not null")
}

func isSensitiveField(field reflect.StructField, hiddenByJSON bool) bool {
	if hiddenByJSON {
		return true
	}
	name := strings.ToLower(field.Name)
	for _, token := range []string{"password", "secret", "token", "key"} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func isScalarField(t reflect.Type) bool {
	return isFilterableField(t)
}

func isWritableField(t reflect.Type) bool {
	if t == reflect.TypeOf(time.Time{}) {
		return false
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Map, reflect.Interface, reflect.Func:
		return false
	default:
		return true
	}
}

func isFilterableField(t reflect.Type) bool {
	if t == reflect.TypeOf(time.Time{}) {
		return true
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isSortableField(t reflect.Type) bool {
	return isFilterableField(t)
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func inferResourceName(t reflect.Type) string {
	t = indirectType(t)
	name := strings.TrimSpace(t.Name())
	if name == "" {
		return ""
	}
	jsonName := defaultJSONFieldName(name)
	if jsonName == "" {
		return ""
	}
	return toKebab(inflection.Plural(jsonName))
}

func cloneFieldOptionsMap(in map[string]FieldOptions) map[string]FieldOptions {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FieldOptions, len(in))
	for key, value := range in {
		cloned := value
		cloned.Enum = cloneSlice(value.Enum)
		if value.Relation != nil {
			relation := *value.Relation
			relation.SearchFields = cloneSlice(value.Relation.SearchFields)
			cloned.Relation = &relation
		}
		out[key] = cloned
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func humanize(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	var out []rune
	lastLower := false
	for _, r := range name {
		if unicode.IsUpper(r) && lastLower {
			out = append(out, ' ')
		}
		out = append(out, r)
		lastLower = unicode.IsLower(r)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
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

func toSnake(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	out := make([]rune, 0, len(runes)+4)
	for i, r := range runes {
		if shouldInsertSnakeUnderscore(runes, i, r) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

func shouldInsertSnakeUnderscore(runes []rune, index int, current rune) bool {
	if index == 0 || !unicode.IsUpper(current) {
		return false
	}
	if unicode.IsLower(runes[index-1]) {
		return true
	}
	return unicode.IsUpper(runes[index-1]) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
}

func toKebab(name string) string {
	return strings.ReplaceAll(toSnake(name), "_", "-")
}

func bytesEqualFoldSpace(left, right []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(left)), strings.TrimSpace(string(right)))
}