// Package admin provides an explicit, metadata-driven admin API for GORM models.
package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
type FieldPermissionChecker func(*ninja.Context, *Resource, *FieldMeta)
type BeforeCreateHook func(*ninja.Context, map[string]any) error
type AfterCreateHook func(*ninja.Context, any) error
type BeforeUpdateHook func(*ninja.Context, any, map[string]any) error
type AfterUpdateHook func(*ninja.Context, any) error
type BeforeDeleteHook func(*ninja.Context, any) error
type AfterDeleteHook func(*ninja.Context, any) error

type RowPermissionChecker interface {
	Scope(*ninja.Context, Action, *Resource, *gorm.DB) *gorm.DB
}

type RowPermissionFunc func(*ninja.Context, Action, *Resource, *gorm.DB) *gorm.DB

func (f RowPermissionFunc) Scope(ctx *ninja.Context, action Action, resource *Resource, db *gorm.DB) *gorm.DB {
	return f(ctx, action, resource, db)
}

type RelationOptions struct {
	Resource     string
	ValueField   string
	LabelField   string
	SearchFields []string
}

type RelationMeta struct {
	Resource     string   `json:"resource"`
	ValueField   string   `json:"value_field"`
	LabelField   string   `json:"label_field"`
	SearchFields []string `json:"search_fields,omitempty"`
}

type FieldOptions struct {
	Label      string
	Component  string
	Enum       []any
	Relation   *RelationOptions
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
	Name             string
	Label            string
	Path             string
	Model            any
	ListFields       []string
	DetailFields     []string
	CreateFields     []string
	UpdateFields     []string
	FilterFields     []string
	SortFields       []string
	SearchFields     []string
	FieldOptions     map[string]FieldOptions
	Permissions      PermissionChecker
	QueryScope       QueryScope
	RowPermissions   RowPermissionChecker
	FieldPermissions FieldPermissionChecker
	BeforeCreate     BeforeCreateHook
	AfterCreate      AfterCreateHook
	BeforeUpdate     BeforeUpdateHook
	AfterUpdate      AfterUpdateHook
	BeforeDelete     BeforeDeleteHook
	AfterDelete      AfterDeleteHook

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
	Name        string        `json:"name"`
	Label       string        `json:"label"`
	Type        string        `json:"type"`
	Component   string        `json:"component"`
	Column      string        `json:"column"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required"`
	Unique      bool          `json:"unique"`
	ReadOnly    bool          `json:"read_only"`
	List        bool          `json:"list"`
	Detail      bool          `json:"detail"`
	Create      bool          `json:"create"`
	Update      bool          `json:"update"`
	Filterable  bool          `json:"filterable"`
	Sortable    bool          `json:"sortable"`
	Searchable  bool          `json:"searchable"`
	Default     any           `json:"default,omitempty"`
	Enum        []any         `json:"enum,omitempty"`
	Relation    *RelationMeta `json:"relation,omitempty"`
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

type RelationOption struct {
	Value any            `json:"value"`
	Label string         `json:"label"`
	Item  map[string]any `json:"item,omitempty"`
}

type RelationOptionsOutput struct {
	Items []RelationOption `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
	Pages int              `json:"pages"`
}

type listInput struct {
	pagination.PageInput
	Search string `form:"search"`
	Sort   string `form:"sort"`
}

type relationOptionsInput struct {
	pagination.PageInput
	Search string `form:"search"`
	Field  string `path:"field" binding:"required"`
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
		ninja.Get(router, base+"/fields/:field/options", resource.handleRelationOptions(s),
			ninja.Summary("List admin relation selector options"),
			ninja.Description("Returns paginated selector options for relation-backed admin fields."))
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
		if err := s.authorize(ctx, ActionList, resource); err != nil {
			if isVisibilityDenied(err) {
				if errors.Is(err, ninja.UnauthorizedError()) {
					return nil, err
				}
				continue
			}
			return nil, err
		}
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

func isVisibilityDenied(err error) bool {
	return errors.Is(err, ninja.UnauthorizedError()) || errors.Is(err, ninja.ForbiddenError())
}

func (r *Resource) handleMetadata(site *Site) func(*ninja.Context, *struct{}) (*ResourceMetadata, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*ResourceMetadata, error) {
		if err := site.authorize(ctx, ActionDetail, r); err != nil {
			return nil, err
		}
		view := r.resolved(ctx)
		meta := view.metadata
		meta.Actions = make([]Action, 0, len(r.metadata.Actions))
		for _, action := range r.metadata.Actions {
			if err := site.authorize(ctx, action, r); err != nil {
				if isVisibilityDenied(err) {
					continue
				}
				return nil, err
			}
			meta.Actions = append(meta.Actions, action)
		}
		return &meta, nil
	}
}

func (r *Resource) handleList(site *Site) func(*ninja.Context, *listInput) (*ResourceListOutput, error) {
	return func(ctx *ninja.Context, in *listInput) (*ResourceListOutput, error) {
		if err := site.authorize(ctx, ActionList, r); err != nil {
			return nil, err
		}
		view := r.resolved(ctx)
		db := r.scopedDB(ctx, ActionList, orm.WithContext(ctx.Context))
		query, err := r.applyListQueryFor(view, db.Model(r.newModel()), ctx.Request.URL.Query(), in)
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
			items = append(items, r.serializeFor(view, itemsPtr.Elem().Index(i), fieldModeList))
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
		view := r.resolved(ctx)
		model, err := r.findByID(r.scopedDB(ctx, ActionDetail, orm.WithContext(ctx.Context)), in.ID)
		if err != nil {
			return nil, err
		}
		return &ResourceRecordOutput{Item: r.serializeFor(view, reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleCreate(site *Site) func(*ninja.Context, *struct{}) (*ResourceRecordOutput, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*ResourceRecordOutput, error) {
		if err := site.authorize(ctx, ActionCreate, r); err != nil {
			return nil, err
		}
		view := r.resolved(ctx)

		values, err := r.decodeWritePayloadFor(view, ctx, fieldModeCreate)
		if err != nil {
			return nil, err
		}
		if r.BeforeCreate != nil {
			if err := r.BeforeCreate(ctx, values); err != nil {
				return nil, err
			}
		}
		if err := r.validateRequiredFor(view, values, fieldModeCreate); err != nil {
			return nil, err
		}

		model := r.newModel()
		if err := r.applyValuesFor(view, reflect.ValueOf(model).Elem(), values); err != nil {
			return nil, err
		}
		if err := orm.WithContext(ctx.Context).Create(model).Error; err != nil {
			return nil, r.normalizeWriteError(ctx, ActionCreate, reflect.ValueOf(model).Elem(), nil, err)
		}
		if r.AfterCreate != nil {
			if err := r.AfterCreate(ctx, model); err != nil {
				return nil, err
			}
		}
		return &ResourceRecordOutput{Item: r.serializeFor(view, reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleUpdate(site *Site) func(*ninja.Context, *pathIDInput) (*ResourceRecordOutput, error) {
	return func(ctx *ninja.Context, in *pathIDInput) (*ResourceRecordOutput, error) {
		if err := site.authorize(ctx, ActionUpdate, r); err != nil {
			return nil, err
		}
		view := r.resolved(ctx)

		scopedDB := r.scopedDB(ctx, ActionUpdate, orm.WithContext(ctx.Context))
		model, err := r.findByID(scopedDB, in.ID)
		if err != nil {
			return nil, err
		}

		values, err := r.decodeWritePayloadFor(view, ctx, fieldModeUpdate)
		if err != nil {
			return nil, err
		}
		if r.BeforeUpdate != nil {
			if err := r.BeforeUpdate(ctx, model, values); err != nil {
				return nil, err
			}
		}

		updates, err := r.updateColumnsFor(view, values)
		if err != nil {
			return nil, err
		}
		if len(updates) > 0 {
			desired := reflect.New(r.modelType).Elem()
			desired.Set(reflect.ValueOf(model).Elem())
			if err := r.applyValuesFor(view, desired, values); err != nil {
				return nil, err
			}
			if err := orm.WithContext(ctx.Context).Model(model).Updates(updates).Error; err != nil {
				return nil, r.normalizeWriteError(ctx, ActionUpdate, desired, r.primaryKeyValue(reflect.ValueOf(model).Elem()), err)
			}
		}
		if err := orm.WithContext(ctx.Context).First(model, r.primaryKeyValue(reflect.ValueOf(model).Elem())).Error; err != nil {
			return nil, err
		}
		if r.AfterUpdate != nil {
			if err := r.AfterUpdate(ctx, model); err != nil {
				return nil, err
			}
		}
		return &ResourceRecordOutput{Item: r.serializeFor(view, reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) handleDelete(site *Site) func(*ninja.Context, *pathIDInput) error {
	return func(ctx *ninja.Context, in *pathIDInput) error {
		if err := site.authorize(ctx, ActionDelete, r); err != nil {
			return err
		}

		model, err := r.findByID(r.scopedDB(ctx, ActionDelete, orm.WithContext(ctx.Context)), in.ID)
		if err != nil {
			return err
		}
		if r.BeforeDelete != nil {
			if err := r.BeforeDelete(ctx, model); err != nil {
				return err
			}
		}
		if err := orm.WithContext(ctx.Context).Delete(model).Error; err != nil {
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

		allowedIDs := make([]any, 0, len(ids))
		for _, id := range ids {
			model := r.newModel()
			if err := r.scopedDB(ctx, ActionBulkDelete, orm.WithContext(ctx.Context)).First(model, id).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return nil, err
			}
			allowedIDs = append(allowedIDs, id)
		}
		if len(allowedIDs) == 0 {
			return &BulkDeleteOutput{Deleted: 0}, nil
		}

		result := orm.WithContext(ctx.Context).Delete(r.newModel(), allowedIDs)
		if result.Error != nil {
			return nil, result.Error
		}
		return &BulkDeleteOutput{Deleted: result.RowsAffected}, nil
	}
}

func (r *Resource) findByID(db *gorm.DB, raw string) (any, error) {
	value, err := r.primaryKey.parseString(raw)
	if err != nil {
		return nil, ninja.NewErrorWithCode(http.StatusBadRequest, "BAD_PATH_PARAM", fmt.Sprintf("id: %s", err.Error()))
	}
	model := r.newModel()
	if err := db.First(model, value).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
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

func (r *Resource) normalizeWriteError(ctx *ninja.Context, action Action, desired reflect.Value, currentID any, err error) error {
	if !isDuplicateKeyError(err) {
		return err
	}
	if fields := r.softDeletedConflictFields(ctx, action, desired, currentID); len(fields) > 0 {
		names := make([]string, 0, len(fields))
		for _, field := range fields {
			names = append(names, field.Meta.Name)
		}
		return ninja.NewErrorWithCode(http.StatusConflict, "SOFT_DELETED_CONFLICT", fmt.Sprintf("a soft-deleted record with the same value for %s already exists; restore or permanently remove it before saving", strings.Join(names, ", ")))
	}
	return ninja.ConflictError()
}

func (r *Resource) softDeletedConflictFields(ctx *ninja.Context, action Action, desired reflect.Value, currentID any) []*fieldMeta {
	softDeleteField := r.softDeleteField()
	if softDeleteField == nil || !desired.IsValid() {
		return nil
	}

	var matches []*fieldMeta
	for _, field := range r.fields {
		if field == nil || !field.Meta.Unique {
			continue
		}
		value, ok := r.fieldValue(desired, field)
		if !ok {
			continue
		}
		query := r.scopedDB(ctx, action, orm.WithContext(ctx.Context)).
			Model(r.newModel()).
			Unscoped().
			Where(clause.Eq{Column: clause.Column{Name: field.Meta.Column}, Value: value})
		if currentID != nil && r.primaryKey != nil {
			query = query.Where(clause.Neq{Column: clause.Column{Name: r.primaryKey.Meta.Column}, Value: currentID})
		}

		var activeCount int64
		if err := query.Session(&gorm.Session{}).
			Where(clause.Eq{Column: clause.Column{Name: softDeleteField.Meta.Column}, Value: nil}).
			Count(&activeCount).Error; err != nil {
			return nil
		}
		if activeCount > 0 {
			return nil
		}

		var deletedCount int64
		if err := query.Session(&gorm.Session{}).
			Where(clause.Neq{Column: clause.Column{Name: softDeleteField.Meta.Column}, Value: nil}).
			Count(&deletedCount).Error; err != nil {
			return nil
		}
		if deletedCount > 0 {
			matches = append(matches, field)
		}
	}
	return matches
}

func (r *Resource) softDeleteField() *fieldMeta {
	for _, field := range r.fields {
		if field == nil {
			continue
		}
		if field.fieldType == reflect.TypeOf(gorm.DeletedAt{}) {
			return field
		}
	}
	return nil
}

func (r *Resource) fieldValue(v reflect.Value, field *fieldMeta) (any, bool) {
	if field == nil || !v.IsValid() {
		return nil, false
	}
	current := v
	for _, index := range field.index {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, true
			}
			current = current.Elem()
		}
		current = current.Field(index)
	}
	if current.Kind() == reflect.Ptr {
		if current.IsNil() {
			return nil, true
		}
		current = current.Elem()
	}
	return current.Interface(), true
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "duplicate entry") ||
		strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "violates unique constraint")
}

func (r *Resource) newModel() any {
	return reflect.New(r.modelType).Interface()
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
		column := queryColumn(field)
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
			db = db.Where(column+" = ?", value)
		case "ne":
			db = db.Where(column+" <> ?", value)
		case "gt":
			db = db.Where(column+" > ?", value)
		case "gte":
			db = db.Where(column+" >= ?", value)
		case "lt":
			db = db.Where(column+" < ?", value)
		case "lte":
			db = db.Where(column+" <= ?", value)
		case "like":
			db = db.Where(column+" LIKE ?", value)
		case "in":
			db = db.Where(column+" IN ?", value)
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
