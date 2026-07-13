// Package admin provides an explicit, metadata-driven admin API for GORM models.
package admin

import (
	"bytes"
	"encoding/csv"
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

const exportMaxRows = 10000

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
	Label       string
	Component   string
	Placeholder string
	Help        string
	Width       string
	Format      string
	Enum        []any
	Relation    *RelationOptions
	Hidden      *bool
	ReadOnly    *bool
	List        *bool
	Detail      *bool
	Create      *bool
	Update      *bool
	Filterable  *bool
	Sortable    *bool
	Searchable  *bool
}

type Resource struct {
	Name             string
	Label            string
	Path             string
	Icon             string
	Group            string
	Description      string
	Order            int
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
	resolvedView *resolvedResource
}

type Site struct {
	checker         PermissionChecker
	resources       []*Resource
	byName          map[string]*Resource
	byModel         map[reflect.Type]*Resource
	ambiguousModels map[reflect.Type]struct{}
}

type Option func(*Site)

func WithPermissionChecker(checker PermissionChecker) Option {
	return func(site *Site) {
		site.checker = checker
	}
}

type ResourceSummary struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Path        string `json:"path"`
	Icon        string `json:"icon,omitempty"`
	Group       string `json:"group,omitempty"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order,omitempty"`
}

type ResourceIndex struct {
	Resources []ResourceSummary `json:"resources"`
}

type ResourceStat struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Path  string `json:"path"`
	Total int64  `json:"total"`
}

type ResourceStatsOutput struct {
	Resources []ResourceStat `json:"resources"`
	Total     int64          `json:"total"`
}

type SearchResultItem struct {
	ID    any            `json:"id"`
	Label string         `json:"label"`
	Item  map[string]any `json:"item"`
}

type SearchResourceResult struct {
	Resource ResourceSummary    `json:"resource"`
	Items    []SearchResultItem `json:"items"`
	Total    int64              `json:"total"`
}

type SearchOutput struct {
	Query   string                 `json:"query"`
	Results []SearchResourceResult `json:"results"`
	Total   int64                  `json:"total"`
	Size    int                    `json:"size"`
}

type FieldMeta struct {
	Name        string        `json:"name"`
	Label       string        `json:"label"`
	Type        string        `json:"type"`
	Component   string        `json:"component"`
	Column      string        `json:"column"`
	Description string        `json:"description,omitempty"`
	Placeholder string        `json:"placeholder,omitempty"`
	Help        string        `json:"help,omitempty"`
	Width       string        `json:"width,omitempty"`
	Format      string        `json:"format,omitempty"`
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
	Icon         string      `json:"icon,omitempty"`
	Group        string      `json:"group,omitempty"`
	Description  string      `json:"description,omitempty"`
	Order        int         `json:"order,omitempty"`
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
	Search string `query:"search"`
	Sort   string `query:"sort"`
}

type searchInput struct {
	Q    string `query:"q"`
	Size int    `query:"size" binding:"omitempty,min=1,max=20"`
}

type relationOptionsInput struct {
	pagination.PageInput
	Search string `query:"search"`
	Field  string `path:"field" binding:"required"`
}

type pathIDInput struct {
	ID string `path:"id" binding:"required"`
}

func NewSite(opts ...Option) *Site {
	site := &Site{
		byName:          map[string]*Resource{},
		byModel:         map[reflect.Type]*Resource{},
		ambiguousModels: map[reflect.Type]struct{}{},
	}
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
	s.registerModel(resource)
	s.resolveAutoRelations()
	return nil
}

func (s *Site) MustRegister(resource *Resource) {
	if err := s.Register(resource); err != nil {
		panic(err)
	}
}

type ModelResource struct {
	Name             string
	Label            string
	Path             string
	Icon             string
	Group            string
	Description      string
	Order            int
	Model            any
	Preloads         []string
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
}

func (r *ModelResource) Resource() *Resource {
	if r == nil {
		return nil
	}
	queryScope := composeQueryScope(r.Preloads, r.QueryScope)
	return &Resource{
		Name:             r.Name,
		Label:            r.Label,
		Path:             r.Path,
		Icon:             r.Icon,
		Group:            r.Group,
		Description:      r.Description,
		Order:            r.Order,
		Model:            r.Model,
		QueryScope:       queryScope,
		ListFields:       cloneSlice(r.ListFields),
		DetailFields:     cloneSlice(r.DetailFields),
		CreateFields:     cloneSlice(r.CreateFields),
		UpdateFields:     cloneSlice(r.UpdateFields),
		FilterFields:     cloneSlice(r.FilterFields),
		SortFields:       cloneSlice(r.SortFields),
		SearchFields:     cloneSlice(r.SearchFields),
		FieldOptions:     cloneFieldOptionsMap(r.FieldOptions),
		Permissions:      r.Permissions,
		RowPermissions:   r.RowPermissions,
		FieldPermissions: r.FieldPermissions,
		BeforeCreate:     r.BeforeCreate,
		AfterCreate:      r.AfterCreate,
		BeforeUpdate:     r.BeforeUpdate,
		AfterUpdate:      r.AfterUpdate,
		BeforeDelete:     r.BeforeDelete,
		AfterDelete:      r.AfterDelete,
	}
}

func composeQueryScope(preloads []string, queryScope QueryScope) QueryScope {
	if len(preloads) == 0 {
		return queryScope
	}
	clonedPreloads := cloneSlice(preloads)
	return func(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
		for _, preload := range clonedPreloads {
			if strings.TrimSpace(preload) == "" {
				continue
			}
			db = db.Preload(preload)
		}
		if queryScope != nil {
			if scoped := queryScope(ctx, db); scoped != nil {
				db = scoped
			}
		}
		return db
	}
}

func (s *Site) RegisterModel(resource *ModelResource) error {
	if resource == nil {
		return fmt.Errorf("admin model resource must not be nil")
	}
	return s.Register(resource.Resource())
}

func (s *Site) MustRegisterModel(resource *ModelResource) {
	if err := s.RegisterModel(resource); err != nil {
		panic(err)
	}
}

func (s *Site) registerModel(resource *Resource) {
	if resource == nil || resource.modelType == nil {
		return
	}
	modelType := resource.modelType
	if _, ambiguous := s.ambiguousModels[modelType]; ambiguous {
		return
	}
	if _, exists := s.byModel[modelType]; exists {
		delete(s.byModel, modelType)
		s.ambiguousModels[modelType] = struct{}{}
		return
	}
	s.byModel[modelType] = resource
}

func (s *Site) resolveAutoRelations() {
	for _, resource := range s.resources {
		if resource == nil {
			continue
		}
		changed := false
		for _, field := range resource.fields {
			if field == nil || field.Meta.Relation == nil {
				continue
			}
			relation := cloneRelationMeta(field.Meta.Relation)
			var target *Resource
			switch {
			case field.autoRelation != nil:
				target = s.byModel[field.autoRelation.targetType]
				if target == nil {
					resetAutoRelation(field)
					changed = true
					continue
				}
				if strings.TrimSpace(relation.Resource) == "" {
					relation.Resource = target.metadata.Name
				}
			case strings.TrimSpace(relation.Resource) != "":
				target = s.byName[relation.Resource]
			}
			if strings.TrimSpace(relation.ValueField) == "" {
				relation.ValueField = "id"
			}
			if target == nil {
				field.Meta.Relation = relation
				changed = true
				continue
			}
			if strings.TrimSpace(relation.LabelField) == "" {
				relation.LabelField = inferRelationLabelField(target)
			}
			if len(relation.SearchFields) == 0 {
				relation.SearchFields = inferRelationSearchFields(target, relation.LabelField)
			}
			field.Meta.Relation = relation
			if !field.componentExplicit {
				field.Meta.Component = "select"
			}
			changed = true
		}
		if changed {
			resource.syncMetadataFields()
		}
	}
}

func cloneRelationMeta(meta *RelationMeta) *RelationMeta {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.SearchFields = cloneSlice(meta.SearchFields)
	return &cloned
}

func resetAutoRelation(field *fieldMeta) {
	field.Meta.Relation = &RelationMeta{ValueField: "id"}
	if !field.componentExplicit {
		field.Meta.Component = "select"
	}
}

func (s *Site) Mount(router *ninja.Router) {
	if router == nil {
		panic("admin router must not be nil")
	}

	ninja.Get(router, "/resources", s.listResources,
		ninja.Summary("List admin resources"),
		ninja.Description("Returns the registered admin resources used to build navigation menus."))
	ninja.Get(router, "/resources/stats", s.resourceStats,
		ninja.Summary("Get admin resource stats"),
		ninja.Description("Returns count summaries for visible admin resources."))
	ninja.Get(router, "/search", s.searchResources,
		ninja.Summary("Search admin resources"),
		ninja.Description("Searches across visible searchable resources and returns grouped results."))

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
		ninja.Get(router, base+"/export", resource.handleExport(s),
			ninja.Summary("Export admin resource records"),
			ninja.Description("Exports filtered admin records as CSV using visible list fields."))
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
			Name:        resource.metadata.Name,
			Label:       resource.metadata.Label,
			Path:        resource.metadata.Path,
			Icon:        resource.metadata.Icon,
			Group:       resource.metadata.Group,
			Description: resource.metadata.Description,
			Order:       resource.metadata.Order,
		})
	}
	return &ResourceIndex{Resources: items}, nil
}

func (s *Site) resourceStats(ctx *ninja.Context, _ *struct{}) (*ResourceStatsOutput, error) {
	items := make([]ResourceStat, 0, len(s.resources))
	var grandTotal int64
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
		db := resource.scopedDB(ctx, ActionList, orm.WithContext(ctx.Context)).Model(resource.newModel())
		var total int64
		if err := db.Count(&total).Error; err != nil {
			return nil, err
		}
		grandTotal += total
		items = append(items, ResourceStat{
			Name:  resource.metadata.Name,
			Label: resource.metadata.Label,
			Path:  resource.metadata.Path,
			Total: total,
		})
	}
	return &ResourceStatsOutput{Resources: items, Total: grandTotal}, nil
}

func (s *Site) searchResources(ctx *ninja.Context, in *searchInput) (*SearchOutput, error) {
	query := strings.TrimSpace(in.Q)
	size := in.Size
	if size < 1 {
		size = 5
	}
	if size > 20 {
		size = 20
	}
	out := &SearchOutput{
		Query:   query,
		Results: []SearchResourceResult{},
		Size:    size,
	}
	if len(query) < 2 {
		return out, nil
	}

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
		view := resource.resolved(ctx)
		if len(view.metadata.SearchFields) == 0 {
			continue
		}

		db := resource.scopedDB(ctx, ActionList, orm.WithContext(ctx.Context)).Model(resource.newModel())
		listQuery, err := resource.applyListQueryFor(view, db, url.Values{}, &listInput{
			PageInput: pagination.PageInput{Page: 1, Size: size},
			Search:    query,
		})
		if err != nil {
			return nil, err
		}

		var total int64
		if err := listQuery.Count(&total).Error; err != nil {
			return nil, err
		}
		if total == 0 {
			continue
		}

		itemsPtr := reflect.New(reflect.SliceOf(resource.modelType))
		if err := listQuery.Limit(size).Find(itemsPtr.Interface()).Error; err != nil {
			return nil, err
		}
		items := make([]SearchResultItem, 0, itemsPtr.Elem().Len())
		for i := 0; i < itemsPtr.Elem().Len(); i++ {
			itemValue := itemsPtr.Elem().Index(i)
			var id any
			if resource.primaryKey != nil {
				id = resource.primaryKey.value(itemValue)
			}
			items = append(items, SearchResultItem{
				ID:    id,
				Label: resource.searchLabelFor(view, itemValue),
				Item:  resource.serializeFor(view, itemValue, fieldModeList),
			})
		}
		out.Total += total
		out.Results = append(out.Results, SearchResourceResult{
			Resource: resource.summary(),
			Items:    items,
			Total:    total,
		})
	}
	return out, nil
}

func (r *Resource) summary() ResourceSummary {
	if r == nil {
		return ResourceSummary{}
	}
	return ResourceSummary{
		Name:        r.metadata.Name,
		Label:       r.metadata.Label,
		Path:        r.metadata.Path,
		Icon:        r.metadata.Icon,
		Group:       r.metadata.Group,
		Description: r.metadata.Description,
		Order:       r.metadata.Order,
	}
}

func (r *Resource) searchLabelFor(view *resolvedResource, value reflect.Value) string {
	if r == nil || view == nil {
		return ""
	}
	names := append(cloneSlice(view.metadata.SearchFields), view.metadata.ListFields...)
	seen := map[string]struct{}{}
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		field := view.fieldByName[name]
		if field == nil || field.fieldType.Kind() != reflect.String {
			continue
		}
		label := strings.TrimSpace(fmt.Sprint(field.value(value)))
		if label != "" {
			return label
		}
	}
	if r.primaryKey != nil {
		return fmt.Sprint(r.primaryKey.value(value))
	}
	return ""
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

func (r *Resource) handleExport(site *Site) func(*ninja.Context, *listInput) (*ninja.Download, error) {
	return func(ctx *ninja.Context, in *listInput) (*ninja.Download, error) {
		if err := site.authorize(ctx, ActionList, r); err != nil {
			return nil, err
		}
		view := r.resolved(ctx)
		fields, err := exportFields(view, requestedExportFieldNames(ctx.Request.URL.Query()))
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			return nil, ninja.NewError(http.StatusBadRequest, "no list fields are available for export")
		}

		db := r.scopedDB(ctx, ActionList, orm.WithContext(ctx.Context))
		query, err := r.applyListQueryFor(view, db.Model(r.newModel()), ctx.Request.URL.Query(), in)
		if err != nil {
			return nil, err
		}

		itemsPtr := reflect.New(reflect.SliceOf(r.modelType))
		if err := query.Limit(exportMaxRows).Find(itemsPtr.Interface()).Error; err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		buf.WriteString("\ufeff")
		writer := csv.NewWriter(&buf)
		header := make([]string, 0, len(fields))
		for _, field := range fields {
			header = append(header, view.meta(field).Label)
		}
		if err := writer.Write(header); err != nil {
			return nil, err
		}
		for i := 0; i < itemsPtr.Elem().Len(); i++ {
			row := make([]string, 0, len(fields))
			itemValue := itemsPtr.Elem().Index(i)
			for _, field := range fields {
				row = append(row, csvCellValue(field.value(itemValue)))
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, err
		}

		filename := exportFilename(r.metadata.Name, time.Now())
		return ninja.NewDownload(filename, "text/csv; charset=utf-8", buf.Bytes()), nil
	}
}

func requestedExportFieldNames(query url.Values) []string {
	var names []string
	for _, raw := range query["fields"] {
		for _, part := range strings.Split(raw, ",") {
			name := strings.TrimSpace(part)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func exportFields(view *resolvedResource, requested []string) ([]*fieldMeta, error) {
	if view == nil {
		return nil, nil
	}
	allowed := make(map[string]*fieldMeta, len(view.metadata.ListFields))
	for _, name := range view.metadata.ListFields {
		field := view.fieldByName[name]
		if field != nil && view.allowed(field, fieldModeList) {
			allowed[name] = field
		}
	}
	if len(requested) == 0 {
		fields := make([]*fieldMeta, 0, len(view.metadata.ListFields))
		for _, name := range view.metadata.ListFields {
			if field := allowed[name]; field != nil {
				fields = append(fields, field)
			}
		}
		return fields, nil
	}
	fields := make([]*fieldMeta, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		if _, ok := seen[name]; ok {
			continue
		}
		field := allowed[name]
		if field == nil {
			return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q is not available for export", name))
		}
		fields = append(fields, field)
		seen[name] = struct{}{}
	}
	return fields, nil
}

func csvCellValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case time.Time:
		if typed.IsZero() {
			return ""
		}
		return typed.Format(time.RFC3339)
	case fmt.Stringer:
		return typed.String()
	case []byte:
		return string(typed)
	}
	if reflect.TypeOf(value).Kind() == reflect.Slice || reflect.TypeOf(value).Kind() == reflect.Map || reflect.TypeOf(value).Kind() == reflect.Struct {
		data, err := json.Marshal(value)
		if err == nil {
			return string(data)
		}
	}
	return fmt.Sprint(value)
}

func exportFilename(resourceName string, now time.Time) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(resourceName) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_':
			b.WriteRune(r)
			lastDash = r == '-'
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "resource"
	}
	return name + "-" + now.Format("20060102-150405") + ".csv"
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

		scopedDB := r.scopedDB(ctx, ActionCreate, orm.WithContext(ctx.Context))
		model := r.newModel()
		if err := r.applyValuesFor(view, reflect.ValueOf(model).Elem(), values); err != nil {
			return nil, err
		}
		if err := scopedDB.Create(model).Error; err != nil {
			return nil, r.normalizeWriteError(ctx, ActionCreate, reflect.ValueOf(model).Elem(), nil, err)
		}
		if err := r.ensureScopedWriteVisible(scopedDB, model); err != nil {
			return nil, err
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
		desired := reflect.ValueOf(model).Elem()
		original := reflect.New(desired.Type()).Elem()
		original.Set(desired)

		values, err := r.decodeWritePayloadFor(view, ctx, fieldModeUpdate)
		if err != nil {
			return nil, err
		}
		if r.BeforeUpdate != nil {
			if err := r.BeforeUpdate(ctx, model, values); err != nil {
				return nil, err
			}
		}

		if len(values) > 0 {
			if err := r.applyValuesFor(view, desired, values); err != nil {
				return nil, err
			}
		}
		columns, err := r.updateColumnsFor(view, original, desired)
		if err != nil {
			return nil, err
		}
		if len(columns) == 0 && len(values) > 0 && r.hasNonPersistedValues(view, values) {
			columns = r.persistedColumnsFor(view)
		}
		if len(columns) > 0 {
			probe := r.newModel()
			if err := scopedDB.Select(queryColumn(r.primaryKey)).First(probe, r.primaryKeyValue(reflect.ValueOf(model).Elem())).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, ninja.NotFoundError()
				}
				return nil, err
			}
			currentID := r.primaryKeyValue(reflect.ValueOf(model).Elem())
			saveErr := orm.WithContext(ctx.Context).
				Model(model).
				Where(queryColumn(r.primaryKey)+" IN (?)", r.scopedPrimaryKeySubquery(scopedDB, currentID)).
				Select(columns).
				Updates(model).Error
			if saveErr != nil {
				return nil, r.normalizeWriteError(ctx, ActionUpdate, desired, currentID, saveErr)
			}
			if err := r.ensureScopedWriteVisible(scopedDB, model); err != nil {
				return nil, err
			}
		}
		if r.AfterUpdate != nil {
			if err := r.AfterUpdate(ctx, model); err != nil {
				return nil, err
			}
		}
		return &ResourceRecordOutput{Item: r.serializeFor(view, reflect.ValueOf(model).Elem(), fieldModeDetail)}, nil
	}
}

func (r *Resource) deleteModelWithHooks(ctx *ninja.Context, scopedDB *gorm.DB, model any) (bool, error) {
	if model == nil {
		return false, nil
	}
	currentID := r.primaryKeyValue(reflect.ValueOf(model).Elem())
	if r.BeforeDelete != nil {
		if err := r.BeforeDelete(ctx, model); err != nil {
			return false, err
		}
	}
	probe := r.newModel()
	if err := scopedDB.Select(queryColumn(r.primaryKey)).First(probe, currentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	result := orm.WithContext(ctx.Context).
		Where(queryColumn(r.primaryKey)+" IN (?)", r.scopedPrimaryKeySubquery(scopedDB, currentID)).
		Delete(model)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	if r.AfterDelete != nil {
		if err := r.AfterDelete(ctx, model); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Resource) ensureScopedWriteVisible(scopedDB *gorm.DB, model any) error {
	if err := r.reloadScopedWrite(scopedDB, model); err != nil {
		if errors.Is(err, ninja.NotFoundError()) {
			return ninja.ForbiddenError()
		}
		return err
	}
	return nil
}

func (r *Resource) reloadScopedWrite(scopedDB *gorm.DB, model any) error {
	if scopedDB == nil || model == nil {
		return ninja.InternalError()
	}
	value := reflect.ValueOf(model)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return ninja.InternalError()
	}
	currentID := r.primaryKeyValue(value.Elem())
	if err := scopedDB.First(model, currentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NotFoundError()
		}
		return err
	}
	return nil
}

func (r *Resource) scopedPrimaryKeySubquery(scopedDB *gorm.DB, currentID any) *gorm.DB {
	return scopedDB.Session(&gorm.Session{}).
		Model(r.newModel()).
		Select(queryColumn(r.primaryKey)).
		Where(clause.Eq{Column: clause.Column{Name: queryColumn(r.primaryKey)}, Value: currentID})
}

func (r *Resource) handleDelete(site *Site) func(*ninja.Context, *pathIDInput) error {
	return func(ctx *ninja.Context, in *pathIDInput) error {
		if err := site.authorize(ctx, ActionDelete, r); err != nil {
			return err
		}

		scopedDB := r.scopedDB(ctx, ActionDelete, orm.WithContext(ctx.Context))
		model, err := r.findByID(scopedDB, in.ID)
		if err != nil {
			return err
		}
		deleted, err := r.deleteModelWithHooks(ctx, scopedDB, model)
		if err != nil {
			return err
		}
		if !deleted {
			return ninja.NotFoundError()
		}
		return nil
	}
}

func (r *Resource) handleBulkDelete(site *Site) func(*ninja.Context, *struct{}) (*BulkDeleteOutput, error) {
	return func(ctx *ninja.Context, _ *struct{}) (*BulkDeleteOutput, error) {
		if err := site.authorize(ctx, ActionBulkDelete, r); err != nil {
			return nil, err
		}

		body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 1<<20)) // 1 MB limit
		if err != nil {
			return nil, err
		}
		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

		var payload struct {
			IDs []json.RawMessage `json:"ids"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, ninja.NewError(http.StatusBadRequest, "invalid request body")
		}
		if len(payload.IDs) == 0 {
			return nil, ninja.NewError(http.StatusBadRequest, "ids must not be empty")
		}

		ids := make([]any, 0, len(payload.IDs))
		for _, raw := range payload.IDs {
			value, err := r.parsePrimaryKeyJSON(raw)
			if err != nil {
				return nil, err
			}
			ids = append(ids, value)
		}

		scopedDB := r.scopedDB(ctx, ActionBulkDelete, orm.WithContext(ctx.Context))
		itemsPtr := reflect.New(reflect.SliceOf(r.modelType))
		if err := scopedDB.Where(clause.IN{Column: clause.Column{Name: queryColumn(r.primaryKey)}, Values: ids}).Find(itemsPtr.Interface()).Error; err != nil {
			return nil, err
		}

		var deleted int64
		for i := 0; i < itemsPtr.Elem().Len(); i++ {
			model := itemsPtr.Elem().Index(i).Addr().Interface()
			removed, err := r.deleteModelWithHooks(ctx, scopedDB, model)
			if err != nil {
				return nil, err
			}
			if removed {
				deleted++
			}
		}
		return &BulkDeleteOutput{Deleted: deleted}, nil
	}
}

func (r *Resource) findByID(db *gorm.DB, raw string) (any, error) {
	value, err := r.primaryKey.parseString(raw)
	if err != nil {
		return nil, ninja.NewError(http.StatusBadRequest, "invalid id")
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
		return nil, ninja.NewError(http.StatusBadRequest, "invalid id value")
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
		return ninja.NewError(http.StatusConflict, fmt.Sprintf("a soft-deleted record with the same value for field(s): %s already exists; restore or permanently remove it before saving", strings.Join(names, ", ")))
	}
	return ninja.ConflictError()
}

func (r *Resource) softDeletedConflictFields(ctx *ninja.Context, action Action, desired reflect.Value, currentID any) []*fieldMeta {
	softDeleteField := r.softDeleteField()
	if softDeleteField == nil || !desired.IsValid() {
		return nil
	}
	if ctx == nil || ctx.Context == nil {
		return nil
	}
	db := orm.WithContext(ctx.Context)
	if db == nil {
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
		query := r.scopedDB(ctx, action, db).
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
			// If the duplicate probe itself fails, fall back to the generic conflict.
			return nil
		}
		if activeCount > 0 {
			return nil
		}

		var deletedCount int64
		if err := query.Session(&gorm.Session{}).
			Where(clause.Neq{Column: clause.Column{Name: softDeleteField.Meta.Column}, Value: nil}).
			Count(&deletedCount).Error; err != nil {
			// If the duplicate probe itself fails, fall back to the generic conflict.
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

func applyFilter(db *gorm.DB, query url.Values, field *fieldMeta, meta FieldMeta) (*gorm.DB, error) {
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
		key := meta.Name + candidate.Suffix
		raw := strings.TrimSpace(query.Get(key))
		if raw == "" {
			continue
		}
		column, err := safeQueryColumnFor(field, meta)
		if err != nil {
			return nil, err
		}
		var (
			value any
		)
		switch candidate.Op {
		case "in":
			parts := strings.Split(raw, ",")
			values := make([]any, 0, len(parts))
			for _, part := range parts {
				parsed, parseErr := field.parseString(strings.TrimSpace(part))
				if parseErr != nil {
					return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q: %s", meta.Name, parseErr.Error()))
				}
				values = append(values, parsed)
			}
			value = values
		case "like":
			value = "%" + raw + "%"
		default:
			value, err = field.parseString(raw)
			if err != nil {
				return nil, ninja.NewError(http.StatusBadRequest, fmt.Sprintf("field %q: %s", meta.Name, err.Error()))
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
