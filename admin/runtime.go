package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/internal/sqlident"
	"github.com/shijl0925/gin-ninja/order"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/gorm"
)

type resolvedResource struct {
	fields      []*fieldMeta
	fieldByName map[string]*fieldMeta
	metadata    ResourceMetadata
	primaryKey  *fieldMeta
	fieldMeta   map[*fieldMeta]FieldMeta
}

func (r *Resource) resolved(ctx *ninja.Context) *resolvedResource {
	if r.FieldPermissions == nil {
		return r.resolvedView
	}
	view := &resolvedResource{
		fields:      r.fields,
		fieldByName: r.fieldByName,
		metadata: ResourceMetadata{
			Name:        r.metadata.Name,
			Label:       r.metadata.Label,
			Path:        r.metadata.Path,
			Icon:        r.metadata.Icon,
			Group:       r.metadata.Group,
			Description: r.metadata.Description,
			Order:       r.metadata.Order,
		},
		primaryKey: r.primaryKey,
		fieldMeta:  make(map[*fieldMeta]FieldMeta, len(r.fields)),
	}
	for _, field := range r.fields {
		meta := cloneFieldMetaValue(field.Meta)
		r.FieldPermissions(ctx, r, &meta)
		normalizeResolvedField(&meta)
		view.fieldMeta[field] = meta
		if includeFieldMetaInMetadata(meta) {
			view.metadata.Fields = append(view.metadata.Fields, cloneFieldMetaValue(meta))
		}
	}
	view.metadata.ListFields = view.visibleFields(fieldModeList)
	view.metadata.DetailFields = view.visibleFields(fieldModeDetail)
	view.metadata.CreateFields = view.visibleFields(fieldModeCreate)
	view.metadata.UpdateFields = view.visibleFields(fieldModeUpdate)
	view.metadata.FilterFields = view.visibleFields(fieldModeFilter)
	view.metadata.SortFields = view.visibleFields(fieldModeSort)
	view.metadata.SearchFields = view.visibleFields(fieldModeSearch)
	view.metadata.Actions = append([]Action(nil), r.metadata.Actions...)
	return view
}

func (view *resolvedResource) meta(field *fieldMeta) FieldMeta {
	if field == nil {
		return FieldMeta{}
	}
	if view != nil && view.fieldMeta != nil {
		if meta, ok := view.fieldMeta[field]; ok {
			return meta
		}
	}
	return field.Meta
}

func (view *resolvedResource) allowed(field *fieldMeta, mode fieldMode) bool {
	meta := view.meta(field)
	switch mode {
	case fieldModeList:
		return meta.List
	case fieldModeDetail:
		return meta.Detail
	case fieldModeCreate:
		return meta.Create
	case fieldModeUpdate:
		return meta.Update
	case fieldModeFilter:
		return meta.Filterable
	case fieldModeSort:
		return meta.Sortable
	case fieldModeSearch:
		return meta.Searchable
	default:
		return false
	}
}

func (view *resolvedResource) visibleFields(mode fieldMode) []string {
	out := make([]string, 0, len(view.fields))
	for _, field := range view.fields {
		if view.allowed(field, mode) {
			out = append(out, view.meta(field).Name)
		}
	}
	return out
}

func cloneFieldMetaValue(meta FieldMeta) FieldMeta {
	meta.Enum = cloneSlice(meta.Enum)
	if meta.Relation != nil {
		relation := *meta.Relation
		relation.SearchFields = cloneSlice(relation.SearchFields)
		meta.Relation = &relation
	}
	return meta
}

func normalizeResolvedField(meta *FieldMeta) {
	if meta.Relation != nil && strings.TrimSpace(meta.Component) == "" {
		meta.Component = "select"
	}
}

func includeFieldMetaInMetadata(meta FieldMeta) bool {
	return meta.List || meta.Detail || meta.Create || meta.Update || meta.Filterable || meta.Sortable || meta.Searchable
}

func includeFieldInMetadata(field *fieldMeta) bool {
	if field == nil {
		return false
	}
	return includeFieldMetaInMetadata(field.Meta)
}

func (r *Resource) scopedDB(ctx *ninja.Context, action Action, db *gorm.DB) *gorm.DB {
	if r.RowPermissions != nil {
		if scoped := r.RowPermissions.Scope(ctx, action, r, db); scoped != nil {
			db = scoped
		}
	}
	if r.QueryScope != nil {
		if scoped := r.QueryScope(ctx, db); scoped != nil {
			db = scoped
		}
	}
	return db
}

func (r *Resource) decodeWritePayloadFor(view *resolvedResource, ctx *ninja.Context, mode fieldMode) (map[string]any, error) {
	body, err := readAndRestoreRequestBody(ctx)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ninja.NewError(http.StatusBadRequest, "invalid request body")
	}

	values := make(map[string]any, len(payload))
	for name, raw := range payload {
		field, ok := view.fieldByName[name]
		if !ok {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("unknown field %q", name))
		}
		if !view.allowed(field, mode) {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q is not writable", name))
		}
		decoded, err := field.decodeJSON(raw)
		if err != nil {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q: %s", name, err.Error()))
		}
		values[name] = decoded
	}
	return values, nil
}

func readAndRestoreRequestBody(ctx *ninja.Context) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return nil, err
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// queryColumn keeps primary-key lookups on the canonical `id` column name
// because the default field-name to snake-case conversion would turn `ID`
// into the invalid SQLite column `i_d`.
func queryColumn(field *fieldMeta) string {
	if field == nil {
		return ""
	}
	return queryColumnFor(field, field.Meta)
}

func queryColumnFor(field *fieldMeta, meta FieldMeta) string {
	if field == nil {
		return ""
	}
	if meta.Name == "id" {
		return meta.Name
	}
	return meta.Column
}

func safeQueryColumnFor(field *fieldMeta, meta FieldMeta) (string, error) {
	column := queryColumnFor(field, meta)
	if !sqlident.IsSafeFieldName(column) {
		return "", fmt.Errorf("admin field %q uses unsafe column %q", meta.Name, column)
	}
	return column, nil
}

func (r *Resource) validateRequiredFor(view *resolvedResource, values map[string]any, mode fieldMode) error {
	if mode != fieldModeCreate {
		return nil
	}
	for _, field := range view.fields {
		meta := view.meta(field)
		if !meta.Required || !view.allowed(field, mode) {
			continue
		}
		if _, ok := values[meta.Name]; ok {
			continue
		}
		return ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q is required", meta.Name))
	}
	return nil
}

func (r *Resource) applyValuesFor(view *resolvedResource, target reflect.Value, values map[string]any) error {
	for name, value := range values {
		field := view.fieldByName[name]
		if field == nil {
			continue
		}
		if err := field.setValue(target, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) updateColumnsFor(view *resolvedResource, before, after reflect.Value) ([]string, error) {
	columns := make([]string, 0, len(view.fields))
	seen := make(map[string]struct{}, len(view.fields))
	for _, field := range view.fields {
		if field == nil || !field.persisted || field.primaryKey {
			continue
		}
		column := strings.TrimSpace(view.meta(field).Column)
		if column == "" {
			continue
		}
		if reflect.DeepEqual(field.value(before), field.value(after)) {
			continue
		}
		if _, ok := seen[column]; ok {
			continue
		}
		seen[column] = struct{}{}
		columns = append(columns, column)
	}
	return columns, nil
}

func (r *Resource) persistedColumnsFor(view *resolvedResource) []string {
	columns := make([]string, 0, len(view.fields))
	seen := make(map[string]struct{}, len(view.fields))
	for _, field := range view.fields {
		if field == nil || !field.persisted || field.primaryKey {
			continue
		}
		column := strings.TrimSpace(view.meta(field).Column)
		if column == "" {
			continue
		}
		if _, ok := seen[column]; ok {
			continue
		}
		seen[column] = struct{}{}
		columns = append(columns, column)
	}
	return columns
}

func (r *Resource) hasNonPersistedValues(view *resolvedResource, values map[string]any) bool {
	for name := range values {
		field := view.fieldByName[name]
		if field != nil && !field.persisted {
			return true
		}
	}
	return false
}

func (r *Resource) applyListQueryFor(view *resolvedResource, db *gorm.DB, query url.Values, in *listInput) (*gorm.DB, error) {
	if term := strings.TrimSpace(in.Search); term != "" {
		if len(view.metadata.SearchFields) == 0 {
			return nil, ninja.NewError(http.StatusBadRequest, "search is not enabled for this resource")
		}
		parts := make([]string, 0, len(view.metadata.SearchFields))
		args := make([]any, 0, len(view.metadata.SearchFields))
		for _, name := range view.metadata.SearchFields {
			field := view.fieldByName[name]
			if field == nil {
				continue
			}
			column, err := safeQueryColumnFor(field, view.meta(field))
			if err != nil {
				return nil, err
			}
			parts = append(parts, column+" LIKE ?")
			args = append(args, "%"+term+"%")
		}
		if len(parts) > 0 {
			db = db.Where(strings.Join(parts, " OR "), args...)
		}
	}

	for _, name := range view.metadata.FilterFields {
		field := view.fieldByName[name]
		if field == nil {
			continue
		}
		next, err := applyFilter(db, query, field, view.meta(field))
		if err != nil {
			return nil, err
		}
		db = next
	}

	if strings.TrimSpace(in.Sort) != "" {
		allowed := make(map[string]*fieldMeta, len(view.metadata.SortFields))
		for _, name := range view.metadata.SortFields {
			if field := view.fieldByName[name]; field != nil {
				allowed[name] = field
			}
		}
		for _, sortField := range order.ParseSort(in.Sort) {
			field := allowed[sortField.Name]
			if field == nil {
				return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("unsupported sort field %q", sortField.Name))
			}
			direction := "ASC"
			if sortField.Desc {
				direction = "DESC"
			}
			column, err := safeQueryColumnFor(field, view.meta(field))
			if err != nil {
				return nil, err
			}
			db = db.Order(column + " " + direction)
		}
	}

	return db, nil
}

func (r *Resource) serializeFor(view *resolvedResource, v reflect.Value, mode fieldMode) map[string]any {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	out := map[string]any{}
	for _, field := range view.fields {
		if !view.allowed(field, mode) {
			continue
		}
		out[view.meta(field).Name] = field.value(v)
	}
	return out
}

func (r *Resource) handleRelationOptions(site *Site) func(*ninja.Context, *relationOptionsInput) (*RelationOptionsOutput, error) {
	return func(ctx *ninja.Context, in *relationOptionsInput) (*RelationOptionsOutput, error) {
		if err := site.authorize(ctx, ActionDetail, r); err != nil {
			return nil, err
		}

		view := r.resolved(ctx)
		field := view.fieldByName[in.Field]
		fieldMeta := view.meta(field)
		if field == nil || fieldMeta.Relation == nil {
			return nil, ninja.NotFoundError()
		}

		target := site.byName[fieldMeta.Relation.Resource]
		if target == nil {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("relation resource %q is not registered", fieldMeta.Relation.Resource))
		}
		if err := site.authorize(ctx, ActionList, target); err != nil {
			return nil, err
		}

		targetView := target.resolved(ctx)
		valueField := targetView.fieldByName[fieldMeta.Relation.ValueField]
		labelField := targetView.fieldByName[fieldMeta.Relation.LabelField]
		if valueField == nil || labelField == nil {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("relation fields %q/%q are not available", fieldMeta.Relation.ValueField, fieldMeta.Relation.LabelField))
		}

		if ctx == nil || ctx.Context == nil {
			return nil, ninja.InternalError()
		}
		baseDB := orm.WithContext(ctx.Context)
		if baseDB == nil {
			return nil, ninja.InternalError()
		}
		db := target.scopedDB(ctx, ActionList, baseDB).Model(target.newModel())
		if term := strings.TrimSpace(in.Search); term != "" {
			names := cloneSlice(fieldMeta.Relation.SearchFields)
			if len(names) == 0 {
				names = []string{fieldMeta.Relation.LabelField}
			}
			parts := make([]string, 0, len(names)+1)
			args := make([]any, 0, len(names)+1)
			if value, err := valueField.parseString(term); err == nil {
				column, err := safeQueryColumnFor(valueField, targetView.meta(valueField))
				if err != nil {
					return nil, err
				}
				parts = append(parts, column+" = ?")
				args = append(args, value)
			}
			for _, name := range names {
				searchField := targetView.fieldByName[name]
				if searchField == nil {
					continue
				}
				column, err := safeQueryColumnFor(searchField, targetView.meta(searchField))
				if err != nil {
					return nil, err
				}
				parts = append(parts, column+" LIKE ?")
				args = append(args, "%"+term+"%")
			}
			if len(parts) > 0 {
				db = db.Where(strings.Join(parts, " OR "), args...)
			}
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			return nil, err
		}

		itemsPtr := reflect.New(reflect.SliceOf(target.modelType))
		page := pagination.NewPage([]RelationOption{}, total, in.PageInput)
		if err := db.Offset(in.PageInput.Offset()).Limit(in.PageInput.Limit()).Find(itemsPtr.Interface()).Error; err != nil {
			return nil, err
		}

		items := make([]RelationOption, 0, itemsPtr.Elem().Len())
		for i := 0; i < itemsPtr.Elem().Len(); i++ {
			itemValue := itemsPtr.Elem().Index(i)
			items = append(items, RelationOption{
				Value: valueField.value(itemValue),
				Label: fmt.Sprint(labelField.value(itemValue)),
				Item:  target.serializeFor(targetView, itemValue, fieldModeList),
			})
		}
		page.Items = items
		return &RelationOptionsOutput{
			Items: page.Items,
			Total: page.Total,
			Page:  page.Page,
			Size:  page.Size,
			Pages: page.Pages,
		}, nil
	}
}
