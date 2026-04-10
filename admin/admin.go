// Package admin provides an explicit, metadata-driven admin API for GORM models.
package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/order"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/gorm"
)

type Action string

const (
	ActionList       Action = "list"
	ActionDetail     Action = "detail"
	ActionCreate     Action = "create"
	ActionUpdate     Action = "update"
	ActionDelete     Action = "delete"
	ActionBulkDelete Action = "bulk_delete"
)

type PermissionChecker func(*ninja.Context, Action, *Resource) error
type QueryScope func(*ninja.Context, *gorm.DB) *gorm.DB
type BeforeCreateHook func(*ninja.Context, map[string]any) error
type AfterCreateHook func(*ninja.Context, any) error
type BeforeUpdateHook func(*ninja.Context, any, map[string]any) error
type AfterUpdateHook func(*ninja.Context, any) error
type BeforeDeleteHook func(*ninja.Context, any) error
type AfterDeleteHook func(*ninja.Context, any) error

type FieldOptions struct {
	Label      string
	Component  string
	Enum       []any
	Hidden     *bool
	ReadOnly   *bool
	List       *bool
	Detail     *bool
	Create     *bool
	Update     *bool
	Filterable *bool
	Sortable   *bool
	Searchable *bool
}

type Resource struct {
	Name         string
	Label        string
	Path         string
	Model        any
	ListFields   []string
	DetailFields []string
	CreateFields []string
	UpdateFields []string
	FilterFields []string
	SortFields   []string
	SearchFields []string
	FieldOptions map[string]FieldOptions
	Permissions  PermissionChecker
	QueryScope   QueryScope
	BeforeCreate BeforeCreateHook
	AfterCreate  AfterCreateHook
	BeforeUpdate BeforeUpdateHook
	AfterUpdate  AfterUpdateHook
	BeforeDelete BeforeDeleteHook
	AfterDelete  AfterDeleteHook

	modelType    reflect.Type
	metadata     ResourceMetadata
	fields       []*fieldMeta
	fieldByName  map[string]*fieldMeta
	primaryKey   *fieldMeta
	allowedQuery map[string]struct{}
}

type Site struct {
	checker   PermissionChecker
	resources []*Resource
	byName    map[string]*Resource
}

type Option func(*Site)

func WithPermissionChecker(checker PermissionChecker) Option {
	return func(site *Site) {
		site.checker = checker
	}
}

type ResourceSummary struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type ResourceIndex struct {
	Resources []ResourceSummary `json:"resources"`
}

type FieldMeta struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Component   string `json:"component"`
	Column      string `json:"column"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Unique      bool   `json:"unique"`
	ReadOnly    bool   `json:"read_only"`
	List        bool   `json:"list"`
	Detail      bool   `json:"detail"`
	Create      bool   `json:"create"`
	Update      bool   `json:"update"`
	Filterable  bool   `json:"filterable"`
	Sortable    bool   `json:"sortable"`
	Searchable  bool   `json:"searchable"`
	Default     any    `json:"default,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
}

type ResourceMetadata struct {
	Name         string      `json:"name"`
	Label        string      `json:"label"`
	Path         string      `json:"path"`
	Fields       []FieldMeta `json:"fields"`
	ListFields   []string    `json:"list_fields"`
	DetailFields []string    `json:"detail_fields"`
	CreateFields []string    `json:"create_fields"`
	UpdateFields []string    `json:"update_fields"`
	FilterFields []string    `json:"filter_fields"`
	SortFields   []string    `json:"sort_fields"`
	SearchFields []string    `json:"search_fields"`
	Actions      []Action    `json:"actions"`
}

type ResourceListOutput struct {
	Items []map[string]any `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
	Pages int              `json:"pages"`
}

type ResourceRecordOutput struct {
	Item map[string]any `json:"item"`
}

type BulkDeleteOutput struct {
	Deleted int64 `json:"deleted"`
}

type listInput struct {
	pagination.PageInput
	Search string `form:"search"`
	Sort   string `form:"sort"`
}

type pathIDInput struct {
	ID string `path:"id" binding:"required"`
}

func NewSite(opts ...Option) *Site {
	site := &Site{byName: map[string]*Resource{}}
	for _, opt := range opts {
		opt(site)
	}
	return site
}

func (s *Site) Register(resource *Resource) error {
	if resource == nil {
		return fmt.Errorf("admin resource must not be nil")
	}
	if err := resource.prepare(); err != nil {
		return err
	}
	if _, exists := s.byName[resource.Name]; exists {
		return fmt.Errorf("admin resource %q already registered", resource.Name)
	}
	s.resources = append(s.resources, resource)
	s.byName[resource.Name] = resource
	return nil
}

func (s *Site) MustRegister(resource *Resource) {
	if err := s.Register(resource); err != nil {
		panic(err)
	}
}

func (s *Site) Mount(router *ninja.Router) {
	if router == nil {
		panic("admin router must not be nil")
	}

	ninja.Get(router, "/resources", s.listResources,
		ninja.Summary("List admin resources"),
		ninja.Description("Returns the registered admin resources used to build navigation menus."))

	for _, resource := range s.resources {
		base := "/resources" + resource.Path
		ninja.Get(router, base+"/meta", resource.handleMetadata(s),
			ninja.Summary("Get admin resource metadata"),
			ninja.Description("Returns resource field metadata, form hints, list fields, and supported actions."))
		ninja.Get(router, base, resource.handleList(s),
			ninja.Summary("List admin resource records"),
			ninja.Description("Returns paginated admin records with safe search, filter, and sort support."))
		ninja.Get(router, base+"/:id", resource.handleDetail(s),
			ninja.Summary("Get admin resource record"),
			ninja.Description("Returns one admin record by primary key."))
		ninja.Post(router, base, resource.handleCreate(s),
			ninja.Summary("Create admin resource record"),
			ninja.Description("Creates one admin record from a JSON payload."),
			ninja.WithTransaction())
		ninja.Put(router, base+"/:id", resource.handleUpdate(s),
			ninja.Summary("Update admin resource record"),
			ninja.Description("Updates one admin record from a partial JSON payload."),
			ninja.WithTransaction())
		ninja.Delete(router, base+"/:id", resource.handleDelete(s),
			ninja.Summary("Delete admin resource record"),
			ninja.Description("Deletes one admin record by primary key."),
			ninja.WithTransaction())
		ninja.Post(router, base+"/bulk-delete", resource.handleBulkDelete(s),
			ninja.Summary("Bulk delete admin resource records"),
			ninja.Description("Deletes multiple admin records by primary key."),
			ninja.WithTransaction())
	}
}

func (s *Site) listResources(ctx *ninja.Context, _ *struct{}) (*ResourceIndex, error) {
	items := make([]ResourceSummary, 0, len(s.resources))
	for _, resource := range s.resources {
		items = append(items, ResourceSummary{
			Name:  resource.metadata.Name,
			Label: resource.metadata.Label,
			Path:  resource.metadata.Path,
		})
	}
	return &ResourceIndex{Resources: items}, nil
}

func (s *Site) authorize(ctx *ninja.Context, action Action, resource *Resource) error {
	if s != nil && s.checker != nil {
		if err := s.checker(ctx, action, resource); err != nil {
			return err
		}
	}
	if resource != nil && resource.Permissions != nil {
		if err := resource.Permissions(ctx, action, resource); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) handleMetadata(site *Site) func(*ninja.Context, *struct{}) (*ResourceMetadata, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*ResourceMetadata, error) {
		if err := site.authorize(ctx, ActionDetail, r); err != nil {
			return nil, err
		}
		meta := r.metadata
		meta.Fields = append([]FieldMeta(nil), meta.Fields...)
		meta.ListFields = append([]string(nil), meta.ListFields...)
		meta.DetailFields = append([]string(nil), meta.DetailFields...)
		meta.CreateFields = append([]string(nil), meta.CreateFields...)
		meta.UpdateFields = append([]string(nil), meta.UpdateFields...)
		meta.FilterFields = append([]string(nil), meta.FilterFields...)
		meta.SortFields = append([]string(nil), meta.SortFields...)
		meta.SearchFields = append([]string(nil), meta.SearchFields...)
		meta.Actions = append([]Action(nil), meta.Actions...)
		return &meta, nil
	}
}

func (r *Resource) handleList(site *Site) func(*ninja.Context, *listInput) (*ResourceListOutput, error) {
	return func(ctx *ninja.Context, in *listInput) (*ResourceListOutput, error) {
		if err := site.authorize(ctx, ActionList, r); err != nil {
			return nil, err
		}

		db := r.scopedDB(ctx, orm.WithContext(ctx.Context))
		query, err := r.applyListQuery(db.Model(r.newModel()), ctx.Request.URL.Query(), in)
		if err != nil {
			return nil, err
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			return nil, err
		}

		itemsPtr := reflect.New(reflect.SliceOf(r.modelType))
		page := pagination.NewPage([]map[string]any{}, total, in.PageInput)
		if err := query.Offset(in.PageInput.Offset()).Limit(in.PageInput.Limit()).Find(itemsPtr.Interface()).Error; err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, itemsPtr.Elem().Len())
		for i := 0; i < itemsPtr.Elem().Len(); i++ {
			items = append(items, r.serialize(itemsPtr.Elem().Index(i), fieldModeList))
		}
		page.Items = items

		return &ResourceListOutput{
			Items: page.Items,
			Total: page.Total,
			Page:  page.Page,
			Size:  page.Size,
			Pages: page.Pages,
		}, nil
	}
}

func (r *Resource) handleDetail(site *Site) func(*ninja.Context, *pathIDInput) (*ResourceRecordOutput, error) {
	return func(ctx *ninja.Context, in *pathIDInput) (*ResourceRecordOutput, error) {
		if err := site.authorize(ctx, ActionDetail, r); err != nil {
			return nil, err
		}
		model, err := r.findByID(r.scopedDB(ctx, orm.WithContext(ctx.Context)), in.ID)
		if err != nil {
			return nil, err
		}
		return &ResourceRecordOutput{Item: r.serialize(reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleCreate(site *Site) func(*ninja.Context, *struct{}) (*ResourceRecordOutput, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*ResourceRecordOutput, error) {
		if err := site.authorize(ctx, ActionCreate, r); err != nil {
			return nil, err
		}

		values, err := r.decodeWritePayload(ctx, fieldModeCreate)
		if err != nil {
			return nil, err
		}
		if r.BeforeCreate != nil {
			if err := r.BeforeCreate(ctx, values); err != nil {
				return nil, err
			}
		}
		if err := r.validateRequired(values, fieldModeCreate); err != nil {
			return nil, err
		}

		model := r.newModel()
		if err := r.applyValues(reflect.ValueOf(model).Elem(), values); err != nil {
			return nil, err
		}
		if err := orm.WithContext(ctx.Context).Create(model).Error; err != nil {
			return nil, err
		}
		if r.AfterCreate != nil {
			if err := r.AfterCreate(ctx, model); err != nil {
				return nil, err
			}
		}
		return &ResourceRecordOutput{Item: r.serialize(reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleUpdate(site *Site) func(*ninja.Context, *pathIDInput) (*ResourceRecordOutput, error) {
	return func(ctx *ninja.Context, in *pathIDInput) (*ResourceRecordOutput, error) {
		if err := site.authorize(ctx, ActionUpdate, r); err != nil {
			return nil, err
		}

		db := r.scopedDB(ctx, orm.WithContext(ctx.Context))
		model, err := r.findByID(db, in.ID)
		if err != nil {
			return nil, err
		}

		values, err := r.decodeWritePayload(ctx, fieldModeUpdate)
		if err != nil {
			return nil, err
		}
		if r.BeforeUpdate != nil {
			if err := r.BeforeUpdate(ctx, model, values); err != nil {
				return nil, err
			}
		}

		updates, err := r.updateColumns(values)
		if err != nil {
			return nil, err
		}
		if len(updates) > 0 {
			if err := db.Model(model).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		if err := db.First(model, r.primaryKeyValue(reflect.ValueOf(model).Elem())).Error; err != nil {
			return nil, err
		}
		if r.AfterUpdate != nil {
			if err := r.AfterUpdate(ctx, model); err != nil {
				return nil, err
			}
		}
		return &ResourceRecordOutput{Item: r.serialize(reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleDelete(site *Site) func(*ninja.Context, *pathIDInput) error {
	return func(ctx *ninja.Context, in *pathIDInput) error {
		if err := site.authorize(ctx, ActionDelete, r); err != nil {
			return err
		}

		db := r.scopedDB(ctx, orm.WithContext(ctx.Context))
		model, err := r.findByID(db, in.ID)
		if err != nil {
			return err
		}
		if r.BeforeDelete != nil {
			if err := r.BeforeDelete(ctx, model); err != nil {
				return err
			}
		}
		if err := db.Delete(model).Error; err != nil {
			return err
		}
		if r.AfterDelete != nil {
			if err := r.AfterDelete(ctx, model); err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *Resource) handleBulkDelete(site *Site) func(*ninja.Context, *struct{}) (*BulkDeleteOutput, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*BulkDeleteOutput, error) {
		if err := site.authorize(ctx, ActionBulkDelete, r); err != nil {
			return nil, err
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			return nil, err
		}
		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

		var payload struct {
			IDs []json.RawMessage `json:"ids"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "INVALID_JSON", err.Error())
		}
		if len(payload.IDs) == 0 {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", "ids must not be empty")
		}

		ids := make([]any, 0, len(payload.IDs))
		for _, raw := range payload.IDs {
			value, err := r.parsePrimaryKeyJSON(raw)
			if err != nil {
				return nil, err
			}
			ids = append(ids, value)
		}

		result := r.scopedDB(ctx, orm.WithContext(ctx.Context)).Delete(r.newModel(), ids)
		if result.Error != nil {
			return nil, result.Error
		}
		return &BulkDeleteOutput{Deleted: result.RowsAffected}, nil
	}
}

func (r *Resource) decodeWritePayload(ctx *ninja.Context, mode fieldMode) (map[string]any, error) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, err
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "INVALID_JSON", err.Error())
	}

	values := make(map[string]any, len(payload))
	for name, raw := range payload {
		field, ok := r.fieldByName[name]
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

func (r *Resource) validateRequired(values map[string]any, mode fieldMode) error {
	if mode != fieldModeCreate {
		return nil
	}
	for _, field := range r.fields {
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

func (r *Resource) applyValues(target reflect.Value, values map[string]any) error {
	for name, value := range values {
		field := r.fieldByName[name]
		if field == nil {
			continue
		}
		if err := field.setValue(target, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) updateColumns(values map[string]any) (map[string]any, error) {
	updates := make(map[string]any, len(values))
	for name, value := range values {
		field := r.fieldByName[name]
		if field == nil {
			continue
		}
		updates[field.Meta.Column] = value
	}
	return updates, nil
}

func (r *Resource) findByID(db *gorm.DB, raw string) (any, error) {
	value, err := r.primaryKey.parseString(raw)
	if err != nil {
		return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_PATH_PARAM", fmt.Sprintf("id: %s", err.Error()))
	}
	model := r.newModel()
	if err := db.First(model, value).Error; err != nil {
		return nil, err
	}
	return model, nil
}

func (r *Resource) parsePrimaryKeyJSON(raw json.RawMessage) (any, error) {
	value, err := r.primaryKey.decodeJSON(raw)
	if err != nil {
		return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("id: %s", err.Error()))
	}
	return value, nil
}

func (r *Resource) applyListQuery(db *gorm.DB, query url.Values, in *listInput) (*gorm.DB, error) {
	if term := strings.TrimSpace(in.Search); term != "" {
		if len(r.metadata.SearchFields) == 0 {
			return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_SEARCH", "search is not enabled for this resource")
		}
		parts := make([]string, 0, len(r.metadata.SearchFields))
		args := make([]any, 0, len(r.metadata.SearchFields))
		for _, name := range r.metadata.SearchFields {
			field := r.fieldByName[name]
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

	for _, name := range r.metadata.FilterFields {
		field := r.fieldByName[name]
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
		allowed := make(map[string]*fieldMeta, len(r.metadata.SortFields))
		for _, name := range r.metadata.SortFields {
			if field := r.fieldByName[name]; field != nil {
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

func (r *Resource) newModel() any {
	return reflect.New(r.modelType).Interface()
}

func (r *Resource) scopedDB(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
	if r.QueryScope != nil {
		return r.QueryScope(ctx, db)
	}
	return db
}

func (r *Resource) primaryKeyValue(v reflect.Value) any {
	if r.primaryKey == nil {
		return nil
	}
	current := v
	for _, index := range r.primaryKey.index {
		current = current.Field(index)
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
	}
	return current.Interface()
}

func (r *Resource) serialize(v reflect.Value, mode fieldMode) map[string]any {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	out := map[string]any{}
	for _, field := range r.fields {
		if !field.allowed(mode) {
			continue
		}
		value := field.value(v)
		out[field.Meta.Name] = value
	}
	return out
}

func boolPtr(v bool) *bool { return &v }

func applyBoolOverride(target *bool, override *bool) {
	if override != nil {
		*target = *override
	}
}

func cloneSlice[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	return append([]T(nil), in...)
}

func containsName(set []string, name string) bool {
	for _, current := range set {
		if current == name {
			return true
		}
	}
	return false
}

func applyFieldSet(fields []*fieldMeta, names []string, mode fieldMode) error {
	if len(names) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, field := range fields {
		seen[field.Meta.Name] = true
		enabled := containsName(names, field.Meta.Name)
		switch mode {
		case fieldModeList:
			field.Meta.List = enabled
		case fieldModeDetail:
			field.Meta.Detail = enabled
		case fieldModeCreate:
			field.Meta.Create = enabled
		case fieldModeUpdate:
			field.Meta.Update = enabled
		case fieldModeFilter:
			field.Meta.Filterable = enabled
		case fieldModeSort:
			field.Meta.Sortable = enabled
		case fieldModeSearch:
			field.Meta.Searchable = enabled
		}
	}
	for _, name := range names {
		if !seen[name] {
			return fmt.Errorf("unknown admin field %q", name)
		}
	}
	return nil
}

func visibleFields(fields []*fieldMeta, mode fieldMode) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.allowed(mode) {
			out = append(out, field.Meta.Name)
		}
	}
	return out
}

func appendAction(actions []Action, action Action, enabled bool) []Action {
	if enabled {
		return append(actions, action)
	}
	return actions
}

func anyWritable(fields []*fieldMeta, mode fieldMode) bool {
	for _, field := range fields {
		if field.allowed(mode) {
			return true
		}
	}
	return false
}

func applyFilter(db *gorm.DB, query url.Values, field *fieldMeta) (*gorm.DB, error) {
	for _, candidate := range []struct {
		Suffix string
		Op     string
	}{
		{"", "eq"},
		{"__eq", "eq"},
		{"__ne", "ne"},
		{"__gt", "gt"},
		{"__gte", "gte"},
		{"__lt", "lt"},
		{"__lte", "lte"},
		{"__like", "like"},
		{"__in", "in"},
		{"__from", "gte"},
		{"__to", "lte"},
	} {
		key := field.Meta.Name + candidate.Suffix
		raw := strings.TrimSpace(query.Get(key))
		if raw == "" {
			continue
		}
		var (
			value any
			err   error
		)
		switch candidate.Op {
		case "in":
			parts := strings.Split(raw, ",")
			values := make([]any, 0, len(parts))
			for _, part := range parts {
				parsed, parseErr := field.parseString(strings.TrimSpace(part))
				if parseErr != nil {
					return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_FILTER", fmt.Sprintf("field %q: %s", field.Meta.Name, parseErr.Error()))
				}
				values = append(values, parsed)
			}
			value = values
		case "like":
			value = "%" + raw + "%"
		default:
			value, err = field.parseString(raw)
			if err != nil {
				return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_FILTER", fmt.Sprintf("field %q: %s", field.Meta.Name, err.Error()))
			}
		}
		switch candidate.Op {
		case "eq":
			db = db.Where(field.Meta.Column+" = ?", value)
		case "ne":
			db = db.Where(field.Meta.Column+" <> ?", value)
		case "gt":
			db = db.Where(field.Meta.Column+" > ?", value)
		case "gte":
			db = db.Where(field.Meta.Column+" >= ?", value)
		case "lt":
			db = db.Where(field.Meta.Column+" < ?", value)
		case "lte":
			db = db.Where(field.Meta.Column+" <= ?", value)
		case "like":
			db = db.Where(field.Meta.Column+" LIKE ?", value)
		case "in":
			db = db.Where(field.Meta.Column+" IN ?", value)
		}
	}
	return db, nil
}

func parseFlexibleTime(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", raw)
}

func sortStrings(in []string) []string {
	out := append([]string(nil), in...)
	slices.Sort(out)
	return out
}
