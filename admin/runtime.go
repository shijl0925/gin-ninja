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
}

func (r *Resource) resolved(ctx *ninja.Context) *resolvedResource {
	view := &resolvedResource{
		fields:      make([]*fieldMeta, 0, len(r.fields)),
		fieldByName: make(map[string]*fieldMeta, len(r.fieldByName)),
		metadata: ResourceMetadata{
			Name:  r.metadata.Name,
			Label: r.metadata.Label,
			Path:  r.metadata.Path,
		},
	}
	for _, field := range r.fields {
		cloned := cloneField(field)
		if r.FieldPermissions != nil {
			r.FieldPermissions(ctx, r, &cloned.Meta)
		}
		normalizeResolvedField(&cloned.Meta)
		view.fields = append(view.fields, cloned)
		view.fieldByName[cloned.Meta.Name] = cloned
		if r.primaryKey != nil && cloned.Meta.Name == r.primaryKey.Meta.Name {
			view.primaryKey = cloned
		}
		if includeFieldInMetadata(cloned) {
			view.metadata.Fields = append(view.metadata.Fields, cloneFieldMetaValue(cloned.Meta))
		}
	}
	view.metadata.ListFields = visibleFields(view.fields, fieldModeList)
	view.metadata.DetailFields = visibleFields(view.fields, fieldModeDetail)
	view.metadata.CreateFields = visibleFields(view.fields, fieldModeCreate)
	view.metadata.UpdateFields = visibleFields(view.fields, fieldModeUpdate)
	view.metadata.FilterFields = visibleFields(view.fields, fieldModeFilter)
	view.metadata.SortFields = visibleFields(view.fields, fieldModeSort)
	view.metadata.SearchFields = visibleFields(view.fields, fieldModeSearch)
	view.metadata.Actions = append([]Action(nil), r.metadata.Actions...)
	return view
}

func cloneField(field *fieldMeta) *fieldMeta {
	if field == nil {
		return nil
	}
	cloned := *field
	cloned.Meta = cloneFieldMetaValue(field.Meta)
	cloned.index = append([]int(nil), field.index...)
	return &cloned
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
	if meta == nil {
		return
	}
	if meta.Relation != nil && strings.TrimSpace(meta.Component) == "" {
		meta.Component = "select"
	}
}

func includeFieldInMetadata(field *fieldMeta) bool {
	if field == nil {
		return false
	}
	meta := field.Meta
	return meta.List || meta.Detail || meta.Create || meta.Update || meta.Filterable || meta.Sortable || meta.Searchable
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
		return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "INVALID_JSON", err.Error())
	}

	values := make(map[string]any, len(payload))
	for name, raw := range payload {
		field, ok := view.fieldByName[name]
		if !ok {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("unknown field %q", name))
		}
		if !field.allowed(mode) {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("field %q is not writable", name))
		}
		decoded, err := field.decodeJSON(raw)
		if err != nil {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("field %q: %s", name, err.Error()))
		}
		values[name] = decoded
	}
	return values, nil
}

func readAndRestoreRequestBody(ctx *ninja.Context) ([]byte, error) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, err
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func (r *Resource) validateRequiredFor(view *resolvedResource, values map[string]any, mode fieldMode) error {
	if mode != fieldModeCreate {
		return nil
	}
	for _, field := range view.fields {
		if !field.Meta.Required || !field.allowed(mode) {
			continue
		}
		if _, ok := values[field.Meta.Name]; ok {
			continue
		}
		return ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("field %q is required", field.Meta.Name))
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

func (r *Resource) updateColumnsFor(view *resolvedResource, values map[string]any) (map[string]any, error) {
	updates := make(map[string]any, len(values))
	for name, value := range values {
		field := view.fieldByName[name]
		if field == nil {
			continue
		}
		updates[field.Meta.Column] = value
	}
	return updates, nil
}

func (r *Resource) applyListQueryFor(view *resolvedResource, db *gorm.DB, query url.Values, in *listInput) (*gorm.DB, error) {
	if term := strings.TrimSpace(in.Search); term != "" {
		if len(view.metadata.SearchFields) == 0 {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_SEARCH", "search is not enabled for this resource")
		}
		parts := make([]string, 0, len(view.metadata.SearchFields))
		args := make([]any, 0, len(view.metadata.SearchFields))
		for _, name := range view.metadata.SearchFields {
			field := view.fieldByName[name]
			if field == nil {
				continue
			}
			parts = append(parts, field.Meta.Column+" LIKE ?")
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
		next, err := applyFilter(db, query, field)
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
				return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_SORT", fmt.Sprintf("unsupported sort field %q", sortField.Name))
			}
			direction := "ASC"
			if sortField.Desc {
				direction = "DESC"
			}
			db = db.Order(field.Meta.Column + " " + direction)
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
		if !field.allowed(mode) {
			continue
		}
		out[field.Meta.Name] = field.value(v)
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
		if field == nil || field.Meta.Relation == nil {
			return nil, ninja.NotFoundError()
		}

		target := site.byName[field.Meta.Relation.Resource]
		if target == nil {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("relation resource %q is not registered", field.Meta.Relation.Resource))
		}
		if err := site.authorize(ctx, ActionList, target); err != nil {
			return nil, err
		}

		targetView := target.resolved(ctx)
		valueField := targetView.fieldByName[field.Meta.Relation.ValueField]
		labelField := targetView.fieldByName[field.Meta.Relation.LabelField]
		if valueField == nil || labelField == nil {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("relation fields %q/%q are not available", field.Meta.Relation.ValueField, field.Meta.Relation.LabelField))
		}

		db := target.scopedDB(ctx, ActionList, orm.WithContext(ctx.Context)).Model(target.newModel())
		if term := strings.TrimSpace(in.Search); term != "" {
			names := cloneSlice(field.Meta.Relation.SearchFields)
			if len(names) == 0 {
				names = []string{field.Meta.Relation.LabelField}
			}
			parts := make([]string, 0, len(names))
			args := make([]any, 0, len(names))
			for _, name := range names {
				searchField := targetView.fieldByName[name]
				if searchField == nil {
					continue
				}
				parts = append(parts, searchField.Meta.Column+" LIKE ?")
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
