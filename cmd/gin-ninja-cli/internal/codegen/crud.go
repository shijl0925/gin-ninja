package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

// CRUDConfig defines the generator inputs for one model scaffold.
type CRUDConfig struct {
	ModelFile   string
	Model       string
	PackageName string
	Tag         string
	WithGormX   *bool
}

func (c CRUDConfig) useGormX() bool {
	return c.WithGormX == nil || *c.WithGormX
}

// WriteCRUDFile generates a CRUD scaffold file for the configured model.
func WriteCRUDFile(cfg CRUDConfig, outputPath string) error {
	content, err := GenerateCRUD(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("write generated file: %w", err)
	}
	return nil
}

// GenerateCRUD returns formatted Go source for a CRUD scaffold.
func GenerateCRUD(cfg CRUDConfig) ([]byte, error) {
	model, err := loadModelSpec(cfg)
	if err != nil {
		return nil, err
	}

	payload := buildTemplateData(model)

	var raw bytes.Buffer
	if err := crudTemplate.Execute(&raw, payload); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}
	return formatted, nil
}

type modelSpec struct {
	packageName    string
	modelName      string
	tag            string
	fields         []fieldSpec
	outputFields   []string
	createFields   []fieldSpec
	updateFields   []fieldSpec
	listFields     []listFieldSpec
	sortFields     []sortFieldSpec
	searchFields   []string
	relations      []relationSpec
	relationOuts   []relationOutSpec
	imports        []importSpec
	idTypeExpr     string
	idField        string
	idColumn       string
	pluralModel    string
	singularLabel  string
	pluralLabel    string
	repoIfaceName  string
	repoImplName   string
	toOutFuncName  string
	useByIDMethods bool
	useGormX       bool
}

type fieldSpec struct {
	Name        string
	TypeExpr    string
	UpdateType  string
	JSONName    string
	ColumnName  string
	Binding     string
	Description string
}

type fieldAccess struct {
	WriteOnly  bool
	CreateOnly bool
	UpdateOnly bool
}

type importSpec struct {
	Alias string
	Path  string
}

type listFieldSpec struct {
	Name        string
	TypeExpr    string
	FormName    string
	FilterTag   string
	Description string
}

type sortFieldSpec struct {
	Alias  string
	Column string
}

type relationKind string

const (
	relationBelongsTo relationKind = "belongs_to"
	relationHasMany   relationKind = "has_many"
	relationMany2Many relationKind = "many2many"
)

type relationSpec struct {
	FieldName           string
	JSONName            string
	TargetModel         string
	TargetOutType       string
	TargetIDType        string
	TargetIDField       string
	TargetIDColumn      string
	InputName           string
	InputJSONName       string
	InputType           string
	UpdateInputType     string
	Kind                relationKind
	Collection          bool
	Pointer             bool
	Preload             string
	ExistingInputField  bool
	ExistingCreateField string
	ExistingUpdateField string
	UseAssociationInput bool
}

type relationOutSpec struct {
	TypeName          string
	ModelName         string
	OutputFieldsValue string
}

type structModelSpec struct {
	idField      string
	idTypeExpr   string
	idColumn     string
	outputFields []string
}

type crudSettings struct {
	Filter   bool
	FilterOp string
	Sort     bool
	Search   bool
}

type templateData struct {
	PackageName       string
	Imports           []importSpec
	ModelName         string
	Tag               string
	PluralModel       string
	SingularLabel     string
	PluralLabel       string
	OutputFieldsValue string
	IDTypeExpr        string
	IDField           string
	IDColumn          string
	CreateFields      []fieldSpec
	UpdateFields      []fieldSpec
	ListFields        []listFieldSpec
	SortFields        []sortFieldSpec
	SearchFields      []string
	RelationOuts      []relationOutSpec
	Relations         []relationSpec
	RepoIfaceName     string
	RepoImplName      string
	ToOutFuncName     string
	UseByIDMethods    bool
	UseGormX          bool
}

func buildTemplateData(model modelSpec) templateData {
	return templateData{
		PackageName:       model.packageName,
		Imports:           model.imports,
		ModelName:         model.modelName,
		Tag:               model.tag,
		PluralModel:       model.pluralModel,
		SingularLabel:     model.singularLabel,
		PluralLabel:       model.pluralLabel,
		OutputFieldsValue: strings.Join(model.outputFields, ","),
		IDTypeExpr:        model.idTypeExpr,
		IDField:           model.idField,
		IDColumn:          model.idColumn,
		CreateFields:      model.createFields,
		UpdateFields:      model.updateFields,
		ListFields:        model.listFields,
		SortFields:        model.sortFields,
		SearchFields:      model.searchFields,
		RelationOuts:      model.relationOuts,
		Relations:         model.relations,
		RepoIfaceName:     model.repoIfaceName,
		RepoImplName:      model.repoImplName,
		ToOutFuncName:     model.toOutFuncName,
		UseByIDMethods:    model.useByIDMethods,
		UseGormX:          model.useGormX,
	}
}

func loadModelSpec(cfg CRUDConfig) (modelSpec, error) {
	if strings.TrimSpace(cfg.ModelFile) == "" {
		return modelSpec{}, fmt.Errorf("model file is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return modelSpec{}, fmt.Errorf("model name is required")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cfg.ModelFile, nil, parser.ParseComments)
	if err != nil {
		return modelSpec{}, fmt.Errorf("parse model file: %w", err)
	}

	importAliases := map[string]importSpec{}
	importPaths := map[string]string{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return modelSpec{}, fmt.Errorf("decode import path: %w", err)
		}
		lookup := filepath.Base(path)
		renderAlias := ""
		if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "." && imp.Name.Name != "_" {
			lookup = imp.Name.Name
			renderAlias = imp.Name.Name
		}
		importAliases[lookup] = importSpec{Alias: renderAlias, Path: path}
		importPaths[lookup] = path
	}

	structTypes := map[string]*ast.StructType{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			candidate, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				structTypes[typeSpec.Name.Name] = candidate
			}
		}
	}
	structType := structTypes[cfg.Model]
	if structType == nil {
		return modelSpec{}, fmt.Errorf("model %q not found in %s", cfg.Model, cfg.ModelFile)
	}

	packageName := strings.TrimSpace(cfg.PackageName)
	if packageName == "" {
		packageName = file.Name.Name
	}

	collector := newImportCollector(importAliases)
	rootFieldNames := map[string]struct{}{}
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name != nil && name.IsExported() {
				rootFieldNames[name.Name] = struct{}{}
			}
		}
	}
	fields := make([]fieldSpec, 0, len(structType.Fields.List))
	outputFields := []string{}
	createFields := []fieldSpec{}
	updateFields := []fieldSpec{}
	listFields := []listFieldSpec{}
	sortFields := []sortFieldSpec{}
	searchFields := []string{}
	relations := []relationSpec{}
	relationOuts := []relationOutSpec{}
	idTypeExpr := "string"
	idField := "ID"
	idColumn := "id"
	idResolved := false
	hasEmbeddedGormModel := false
	fieldByName := map[string]fieldSpec{}
	targetCache := map[string]structModelSpec{}

	for _, field := range structType.Fields.List {
		tags := parseTagLiteral(field.Tag)
		gormTag := tags.Get("gorm")

		if len(field.Names) == 0 {
			if isEmbeddedGormModel(field.Type, importPaths) {
				hasEmbeddedGormModel = true
				if !contains(outputFields, "id") {
					outputFields = append(outputFields, "id")
				}
				if !idResolved {
					idTypeExpr = "uint"
					idResolved = true
				}
			}
			continue
		}

		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}

			if rel, ok := buildRelationSpec(fset, cfg.Model, name.Name, field, tags, gormTag, structTypes, importPaths, targetCache, rootFieldNames); ok {
				relations = append(relations, rel)
				if rel.TargetOutType != "" && !containsRelationOut(relationOuts, rel.TargetOutType) {
					target := targetCache[rel.TargetModel]
					relationOuts = append(relationOuts, relationOutSpec{
						TypeName:          rel.TargetOutType,
						ModelName:         rel.TargetModel,
						OutputFieldsValue: strings.Join(target.outputFields, ","),
					})
				}
				continue
			}

			typeExpr := exprString(fset, field.Type)
			jsonName, hidden := resolveJSONName(name.Name, tags)
			access := resolveFieldAccess(tags)
			crudSettings := parseCRUDSettings(tags.Get("crud"))
			if isIDField(name.Name, jsonName, gormTag) && !idResolved {
				idField = name.Name
				idTypeExpr = typeExpr
				idColumn = resolveColumnName(name.Name, gormTag)
				collector.addExpr(field.Type)
				idResolved = true
			}

			if !isRenderableFieldType(field.Type, importPaths) {
				continue
			}

			binding := strings.TrimSpace(tags.Get("binding"))
			description := strings.TrimSpace(tags.Get("description"))
			fieldSpec := fieldSpec{
				Name:        name.Name,
				TypeExpr:    typeExpr,
				UpdateType:  optionalType(typeExpr),
				JSONName:    resolveInputJSONName(name.Name, tags),
				ColumnName:  resolveColumnName(name.Name, gormTag),
				Binding:     binding,
				Description: description,
			}
			fields = append(fields, fieldSpec)
			fieldByName[name.Name] = fieldSpec
			if !hidden && !access.WriteOnly && !shouldSkipOutputField(gormTag) {
				outputFields = append(outputFields, jsonName)
			}

			if isWritableField(field.Type, gormTag, importPaths) {
				usedInIO := false
				if allowCreateField(access) && allowCreateWritableField(name.Name, jsonName, typeExpr, gormTag) {
					createFields = append(createFields, fieldSpec)
					usedInIO = true
				}
				if allowUpdateField(access) && allowUpdateWritableField(name.Name, jsonName, gormTag) {
					fieldSpec.Binding = normalizeUpdateBinding(binding)
					updateFields = append(updateFields, fieldSpec)
					usedInIO = true
				}
				if usedInIO {
					collector.addExpr(field.Type)
				}
			}

			if crudSettings.Filter && isListFilterType(field.Type, importPaths) {
				listFields = append(listFields, listFieldSpec{
					Name:        name.Name,
					TypeExpr:    optionalType(typeExpr),
					FormName:    jsonName,
					FilterTag:   fmt.Sprintf("%s,%s", fieldSpec.ColumnName, crudSettings.FilterOp),
					Description: description,
				})
				collector.addExpr(field.Type)
			}
			if crudSettings.Sort {
				sortFields = append(sortFields, sortFieldSpec{Alias: jsonName, Column: fieldSpec.ColumnName})
			}
			if crudSettings.Search {
				searchFields = append(searchFields, fieldSpec.ColumnName)
			}
		}
	}

	if hasEmbeddedGormModel && !contains(outputFields, "id") {
		outputFields = append([]string{"id"}, outputFields...)
	}
	outputFields = uniqueStrings(outputFields)
	searchFields = uniqueStrings(searchFields)
	sortFields = uniqueSortFields(sortFields)
	relations = finalizeRelationInputBindings(relations, fieldByName)
	for _, relation := range relations {
		if relation.Collection || !relation.UseAssociationInput {
			continue
		}
		return modelSpec{}, fmt.Errorf("belongs-to relation %q on model %q requires exported foreign key field %q", relation.FieldName, cfg.Model, relation.InputName)
	}

	useGormX := cfg.useGormX()
	imports := []importSpec{
		{Alias: "", Path: "errors"},
		{Alias: "ninja", Path: "github.com/shijl0925/gin-ninja"},
		{Alias: "", Path: "github.com/shijl0925/gin-ninja/orm"},
		{Alias: "", Path: "github.com/shijl0925/gin-ninja/pagination"},
		{Alias: "", Path: "gorm.io/gorm"},
	}
	if useGormX {
		imports = append(imports, importSpec{Alias: "", Path: "github.com/shijl0925/go-toolkits/gormx"})
	}
	if len(listFields) > 0 || len(searchFields) > 0 {
		imports = append(imports, importSpec{Alias: "", Path: "github.com/shijl0925/gin-ninja/filter"})
	}
	if len(sortFields) > 0 {
		imports = append(imports, importSpec{Alias: "", Path: "github.com/shijl0925/gin-ninja/order"})
	}
	if needsFmtImport(relations) {
		imports = append(imports, importSpec{Alias: "", Path: "fmt"})
	}
	imports = append(imports, collector.list()...)
	imports = uniqueImports(imports)

	tag := strings.TrimSpace(cfg.Tag)
	pluralModel := pluralizeName(cfg.Model)
	if tag == "" {
		tag = pluralModel
	}

	return modelSpec{
		packageName:    packageName,
		modelName:      cfg.Model,
		tag:            tag,
		fields:         fields,
		outputFields:   outputFields,
		createFields:   createFields,
		updateFields:   updateFields,
		listFields:     listFields,
		sortFields:     sortFields,
		searchFields:   searchFields,
		relations:      relations,
		relationOuts:   relationOuts,
		imports:        imports,
		idTypeExpr:     idTypeExpr,
		idField:        idField,
		idColumn:       idColumn,
		pluralModel:    pluralModel,
		singularLabel:  lowerLabel(cfg.Model),
		pluralLabel:    lowerLabel(pluralModel),
		repoIfaceName:  "I" + cfg.Model + "Repo",
		repoImplName:   lowerCamel(cfg.Model) + "Repo",
		toOutFuncName:  "to" + cfg.Model + "Out",
		useByIDMethods: isIntConvertibleIDType(idTypeExpr),
		useGormX:       useGormX,
	}, nil
}

type importCollector struct {
	used      map[string]struct{}
	importMap map[string]importSpec
}

func newImportCollector(imports map[string]importSpec) *importCollector {
	return &importCollector{used: map[string]struct{}{}, importMap: imports}
}

func (c *importCollector) addExpr(expr ast.Expr) {
	ast.Inspect(expr, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, exists := c.importMap[ident.Name]; exists {
			c.used[ident.Name] = struct{}{}
		}
		return true
	})
}

func (c *importCollector) list() []importSpec {
	out := make([]importSpec, 0, len(c.used))
	for alias := range c.used {
		imp := c.importMap[alias]
		out = append(out, imp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func parseCRUDSettings(raw string) crudSettings {
	settings := crudSettings{FilterOp: string(defaultFilterOperator(nil))}
	for _, token := range splitAccessTag(raw) {
		key, value, _ := strings.Cut(strings.TrimSpace(token), ":")
		if key == token {
			key, value, _ = strings.Cut(strings.TrimSpace(token), "=")
		}
		switch normalizeAccessToken(key) {
		case "filter":
			settings.Filter = true
			if strings.TrimSpace(value) != "" {
				settings.FilterOp = strings.TrimSpace(value)
			}
		case "sort":
			settings.Sort = true
		case "search":
			settings.Search = true
		}
	}
	if settings.FilterOp == "" {
		settings.FilterOp = string(defaultFilterOperator(nil))
	}
	return settings
}

func defaultFilterOperator(expr ast.Expr) string {
	if expr == nil {
		return "eq"
	}
	if isStringLikeExpr(expr) {
		return "eq"
	}
	return "eq"
}

func isListFilterType(expr ast.Expr, imports map[string]string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name != "interface{}" && t.Name != "any"
	case *ast.StarExpr:
		return isListFilterType(t.X, imports)
	case *ast.SelectorExpr:
		ident, ok := t.X.(*ast.Ident)
		return ok && imports[ident.Name] != ""
	default:
		return false
	}
}

func isStringLikeExpr(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == "string"
	case *ast.StarExpr:
		return isStringLikeExpr(t.X)
	default:
		return false
	}
}

func buildRelationSpec(
	fset *token.FileSet,
	rootModel string,
	fieldName string,
	field *ast.Field,
	tags reflect.StructTag,
	gormTag string,
	structTypes map[string]*ast.StructType,
	imports map[string]string,
	cache map[string]structModelSpec,
	rootFieldNames map[string]struct{},
) (relationSpec, bool) {
	if shouldSkipRelationField(gormTag) {
		return relationSpec{}, false
	}

	targetModel, collection, pointer, ok := relationTargetModel(field.Type, structTypes)
	if !ok {
		return relationSpec{}, false
	}
	if !collection && !looksLikeBelongsToRelation(fieldName, gormTag, rootFieldNames) {
		return relationSpec{}, false
	}
	targetStruct := structTypes[targetModel]
	targetSpec, ok := buildStructModelSpec(fset, targetStruct, structTypes, imports)
	if !ok {
		return relationSpec{}, false
	}
	cache[targetModel] = targetSpec

	jsonName, hidden := resolveJSONName(fieldName, tags)
	if hidden || jsonName == "" {
		jsonName = lowerCamel(fieldName)
	}

	kind := relationBelongsTo
	if strings.Contains(strings.ToLower(gormTag), "many2many") {
		kind = relationMany2Many
	} else if collection {
		kind = relationHasMany
	}

	rel := relationSpec{
		FieldName:      fieldName,
		JSONName:       jsonName,
		TargetModel:    targetModel,
		TargetOutType:  rootModel + fieldName + "Out",
		TargetIDType:   targetSpec.idTypeExpr,
		TargetIDField:  targetSpec.idField,
		TargetIDColumn: targetSpec.idColumn,
		Kind:           kind,
		Collection:     collection,
		Pointer:        pointer,
		Preload:        fieldName,
	}
	if !collection {
		rel.InputName = resolveBelongsToForeignKey(fieldName, gormTag)
		rel.InputJSONName = toSnake(rel.InputName)
		rel.InputType = targetSpec.idTypeExpr
		rel.UpdateInputType = optionalType(targetSpec.idTypeExpr)
		rel.UseAssociationInput = true
		return rel, true
	}

	rel.InputName = fieldName + "IDs"
	rel.InputJSONName = toSnake(fieldName) + "_ids"
	rel.InputType = "[]" + targetSpec.idTypeExpr
	rel.UpdateInputType = "*[]" + targetSpec.idTypeExpr
	return rel, true
}

func relationTargetModel(expr ast.Expr, structTypes map[string]*ast.StructType) (string, bool, bool, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		_, ok := structTypes[t.Name]
		return t.Name, false, false, ok
	case *ast.StarExpr:
		name, collection, _, ok := relationTargetModel(t.X, structTypes)
		return name, collection, true, ok
	case *ast.ArrayType:
		name, _, pointer, ok := relationTargetModel(t.Elt, structTypes)
		return name, true, pointer, ok
	default:
		return "", false, false, false
	}
}

func shouldSkipRelationField(gormTag string) bool {
	return hasSkippedGORMFieldTag(gormTag)
}

func looksLikeBelongsToRelation(fieldName, gormTag string, rootFieldNames map[string]struct{}) bool {
	if parseGORMSetting(gormTag, "foreignKey") != "" || parseGORMSetting(gormTag, "references") != "" {
		return true
	}
	_, ok := rootFieldNames[fieldName+"ID"]
	return ok
}

func buildStructModelSpec(fset *token.FileSet, structType *ast.StructType, structTypes map[string]*ast.StructType, imports map[string]string) (structModelSpec, bool) {
	if structType == nil {
		return structModelSpec{}, false
	}
	spec := structModelSpec{idField: "ID", idTypeExpr: "uint", idColumn: "id"}
	hasEmbeddedGormModel := false
	for _, field := range structType.Fields.List {
		tags := parseTagLiteral(field.Tag)
		gormTag := tags.Get("gorm")
		if len(field.Names) == 0 {
			if isEmbeddedGormModel(field.Type, imports) {
				hasEmbeddedGormModel = true
			}
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			if _, _, _, ok := relationTargetModel(field.Type, structTypes); ok {
				continue
			}
			typeExpr := exprString(fset, field.Type)
			jsonName, hidden := resolveJSONName(name.Name, tags)
			if isIDField(name.Name, jsonName, gormTag) {
				spec.idField = name.Name
				spec.idTypeExpr = typeExpr
				spec.idColumn = resolveColumnName(name.Name, gormTag)
			}
			if hidden || shouldSkipOutputField(gormTag) || !isRenderableFieldType(field.Type, imports) {
				continue
			}
			spec.outputFields = append(spec.outputFields, jsonName)
		}
	}
	if hasEmbeddedGormModel && !contains(spec.outputFields, "id") {
		spec.outputFields = append([]string{"id"}, spec.outputFields...)
	}
	spec.outputFields = uniqueStrings(spec.outputFields)
	return spec, true
}

func finalizeRelationInputBindings(relations []relationSpec, fields map[string]fieldSpec) []relationSpec {
	out := make([]relationSpec, len(relations))
	copy(out, relations)
	for i := range out {
		if out[i].Collection {
			continue
		}
		if field, ok := fields[out[i].InputName]; ok {
			out[i].ExistingInputField = true
			out[i].ExistingCreateField = field.Name
			out[i].ExistingUpdateField = field.Name
			out[i].UseAssociationInput = false
		}
	}
	return out
}

func uniqueSortFields(fields []sortFieldSpec) []sortFieldSpec {
	seen := map[string]struct{}{}
	out := make([]sortFieldSpec, 0, len(fields))
	for _, field := range fields {
		key := field.Alias + "|" + field.Column
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	return out
}

func containsRelationOut(values []relationOutSpec, typeName string) bool {
	for _, value := range values {
		if value.TypeName == typeName {
			return true
		}
	}
	return false
}

func needsFmtImport(relations []relationSpec) bool {
	for _, relation := range relations {
		if relation.Collection || relation.UseAssociationInput {
			return true
		}
	}
	return false
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)
	return buf.String()
}

func parseTagLiteral(lit *ast.BasicLit) reflect.StructTag {
	if lit == nil {
		return ""
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return reflect.StructTag(value)
}

func resolveJSONName(fieldName string, tag reflect.StructTag) (string, bool) {
	jsonTag := strings.TrimSpace(tag.Get("json"))
	if jsonTag == "-" {
		return "", true
	}
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0], false
		}
	}
	return lowerCamel(fieldName), false
}

func resolveInputJSONName(fieldName string, tag reflect.StructTag) string {
	jsonName, hidden := resolveJSONName(fieldName, tag)
	if hidden {
		return lowerCamel(fieldName)
	}
	return jsonName
}

func normalizeUpdateBinding(binding string) string {
	if strings.TrimSpace(binding) == "" {
		return ""
	}
	parts := strings.Split(binding, ",")
	filtered := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "required" {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) == 0 {
		return "omitempty"
	}
	if filtered[0] != "omitempty" {
		filtered = append([]string{"omitempty"}, filtered...)
	}
	return strings.Join(filtered, ",")
}

func resolveFieldAccess(tag reflect.StructTag) fieldAccess {
	var access fieldAccess
	applyFieldAccessTag(&access, tag.Get("ninja"))
	applyFieldAccessTag(&access, tag.Get("crud"))
	return access
}

func applyFieldAccessTag(access *fieldAccess, raw string) {
	for _, token := range splitAccessTag(raw) {
		switch normalizeAccessToken(token) {
		case "writeonly":
			access.WriteOnly = true
		case "createonly":
			access.CreateOnly = true
		case "updateonly":
			access.UpdateOnly = true
		}
	}
}

func splitAccessTag(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	})
}

func normalizeAccessToken(token string) string {
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

func isIDField(name, jsonName, gormTag string) bool {
	if name == "ID" || jsonName == "id" {
		return true
	}
	lower := strings.ToLower(gormTag)
	return strings.Contains(lower, "primarykey") || strings.Contains(lower, "primary_key")
}

func resolveColumnName(fieldName, gormTag string) string {
	for _, part := range strings.Split(gormTag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "column") {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	// Fall back to GORM's default snake_case column naming when no explicit column is configured, e.g. UserID -> user_id.
	return toSnake(fieldName)
}

func isIntConvertibleIDType(typeExpr string) bool {
	return strings.TrimSpace(typeExpr) == "int"
}

func isWritableField(expr ast.Expr, gormTag string, imports map[string]string) bool {
	if hasNonWritableGORMTag(gormTag) {
		return false
	}
	return isRenderableFieldType(expr, imports)
}

func shouldSkipOutputField(gormTag string) bool {
	lower := strings.ToLower(gormTag)
	return hasSkippedGORMFieldTag(gormTag) ||
		strings.Contains(lower, "many2many") ||
		strings.Contains(lower, "foreignkey") ||
		strings.Contains(lower, "references")
}

func allowCreateWritableField(name, jsonName, typeExpr, gormTag string) bool {
	switch name {
	case "CreatedAt", "UpdatedAt", "DeletedAt":
		return false
	}
	if !isIDField(name, jsonName, gormTag) {
		return true
	}
	if strings.Contains(strings.ToLower(gormTag), "autoincrement:false") {
		return true
	}
	return !isIntegerIDType(typeExpr)
}

func allowUpdateWritableField(name, jsonName, gormTag string) bool {
	switch name {
	case "CreatedAt", "UpdatedAt", "DeletedAt":
		return false
	default:
		return !isIDField(name, jsonName, gormTag)
	}
}

func isIntegerIDType(typeExpr string) bool {
	switch strings.TrimSpace(typeExpr) {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	default:
		return false
	}
}

func resolveBelongsToForeignKey(fieldName, gormTag string) string {
	if value := parseGORMSetting(gormTag, "foreignKey"); value != "" {
		return value
	}
	return fieldName + "ID"
}

func parseGORMSetting(gormTag, key string) string {
	for _, part := range strings.Split(gormTag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, value, ok := strings.Cut(part, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func hasNonWritableGORMTag(tag string) bool {
	lower := strings.ToLower(tag)
	return hasSkippedGORMFieldTag(tag) ||
		strings.Contains(lower, "autocreatetime") ||
		strings.Contains(lower, "autoupdatetime") ||
		strings.Contains(lower, "many2many") ||
		strings.Contains(lower, "foreignkey") ||
		strings.Contains(lower, "references")
}

func isRenderableFieldType(expr ast.Expr, imports map[string]string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return true
	case *ast.StarExpr:
		return isRenderableFieldType(t.X, imports)
	case *ast.ArrayType:
		return isRenderableFieldType(t.Elt, imports)
	case *ast.MapType:
		return isRenderableFieldType(t.Key, imports) && isRenderableFieldType(t.Value, imports)
	case *ast.StructType:
		return true
	case *ast.InterfaceType:
		return len(t.Methods.List) == 0
	case *ast.SelectorExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = imports[ident.Name]
		return ok
	case *ast.IndexExpr:
		return isRenderableFieldType(t.X, imports) && isRenderableFieldType(t.Index, imports)
	case *ast.IndexListExpr:
		if !isRenderableFieldType(t.X, imports) {
			return false
		}
		for _, index := range t.Indices {
			if !isRenderableFieldType(index, imports) {
				return false
			}
		}
		return true
	case *ast.ParenExpr:
		return isRenderableFieldType(t.X, imports)
	default:
		return false
	}
}

func isEmbeddedGormModel(expr ast.Expr, imports map[string]string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Model" && imports[ident.Name] == "gorm.io/gorm"
}

func hasSkippedGORMFieldTag(tag string) bool {
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		if part == "-" || strings.HasPrefix(part, "-:") {
			return true
		}
	}
	return false
}

func optionalType(typeExpr string) string {
	if strings.HasPrefix(strings.TrimSpace(typeExpr), "*") {
		return typeExpr
	}
	return "*" + typeExpr
}

// DefaultOutputName returns the default scaffold file name for a model.
func DefaultOutputName(model string) string {
	return toSnake(model) + "_crud_gen.go"
}

func pluralizeName(name string) string {
	if name == "" {
		return name
	}
	lower := strings.ToLower(name)
	runes := []rune(lower)
	switch {
	case strings.HasSuffix(lower, "y") && len(runes) > 1 && !isVowel(runes[len(runes)-2]):
		return name[:len(name)-1] + "ies"
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return name + "es"
	default:
		return name + "s"
	}
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func lowerLabel(name string) string {
	parts := splitWords(name)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, " ")
}

func splitWords(name string) []string {
	if name == "" {
		return nil
	}
	var parts []string
	var current []rune
	runes := []rune(name)
	for i, r := range runes {
		hasLowercasePrev := i > 0 && unicode.IsLower(runes[i-1])
		hasLowercaseNext := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if i > 0 && unicode.IsUpper(r) && (hasLowercasePrev || hasLowercaseNext) {
			parts = append(parts, string(current))
			current = current[:0]
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}

func lowerCamel(name string) string {
	parts := splitWords(name)
	if len(parts) == 0 {
		return ""
	}
	for i := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(parts[i])
			continue
		}
		if len(parts[i]) == 0 {
			continue
		}
		if len(parts[i]) == 1 {
			parts[i] = strings.ToUpper(parts[i])
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, "")
}

func toSnake(input string) string {
	if input == "" {
		return input
	}
	runes := []rune(input)
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func uniqueImports(values []importSpec) []importSpec {
	seen := map[string]struct{}{}
	out := make([]importSpec, 0, len(values))
	for _, value := range values {
		key := value.Alias + "|" + value.Path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Path < out[j].Path
	})
	return out
}

var crudTemplate = template.Must(template.New("crud").Parse(`// Code generated by gin-ninja generate crud; DO NOT EDIT.

package {{ .PackageName }}

import (
{{ range .Imports }}	{{ if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{ end }}
)

// {{ .ModelName }}Out is the default public response schema generated from {{ .ModelName }}.
type {{ .ModelName }}Out struct {
ninja.ModelSchema[{{ .ModelName }}] ` + "`fields:\"{{ .OutputFieldsValue }}\"`" + `
{{- range .Relations }}
	{{ .FieldName }} {{ if .Collection }}[]{{ else }}*{{ end }}{{ .TargetOutType }} ` + "`json:\"{{ .JSONName }},omitempty\"`" + `
{{- end }}
}

{{ range .RelationOuts }}
type {{ .TypeName }} struct {
	ninja.ModelSchema[{{ .ModelName }}] ` + "`fields:\"{{ .OutputFieldsValue }}\"`" + `
}
{{ end }}

{{- if .UseGormX }}
// {{ .RepoIfaceName }} exposes the generated gormx repository contract for {{ .ModelName }}.
type {{ .RepoIfaceName }} interface {
	gormx.IBaseRepo[{{ .ModelName }}]
}

type {{ .RepoImplName }} struct {
	gormx.BaseRepo[{{ .ModelName }}]
}

// New{{ .ModelName }}Repo constructs the generated gormx repository for {{ .ModelName }}.
func New{{ .ModelName }}Repo() {{ .RepoIfaceName }} {
	return &{{ .RepoImplName }}{}
}
{{- end }}

// {{ .ToOutFuncName }} converts a {{ .SingularLabel }} model to the generated response schema.
func {{ .ToOutFuncName }}(item {{ .ModelName }}) (*{{ .ModelName }}Out, error) {
	out, err := ninja.BindModelSchema[{{ .ModelName }}Out](item)
	if err != nil {
		return nil, err
	}
{{- range .Relations }}
{{- if .Collection }}
	if len(item.{{ .FieldName }}) > 0 {
		out.{{ .FieldName }} = make([]{{ .TargetOutType }}, 0, len(item.{{ .FieldName }}))
		for _, related := range item.{{ .FieldName }} {
			bound, err := ninja.BindModelSchema[{{ .TargetOutType }}](related)
			if err != nil {
				return nil, err
			}
			out.{{ .FieldName }} = append(out.{{ .FieldName }}, *bound)
		}
	}
{{- else if .Pointer }}
	if item.{{ .FieldName }} != nil {
		bound, err := ninja.BindModelSchema[{{ .TargetOutType }}](*item.{{ .FieldName }})
		if err != nil {
			return nil, err
		}
		out.{{ .FieldName }} = bound
	}
{{- else }}
	{
		var zero {{ .TargetIDType }}
		if item.{{ .FieldName }}.{{ .TargetIDField }} != zero {
			bound, err := ninja.BindModelSchema[{{ .TargetOutType }}](item.{{ .FieldName }})
			if err != nil {
				return nil, err
			}
			out.{{ .FieldName }} = bound
		}
	}
{{- end }}
{{- end }}
	return out, nil
}

// List{{ .PluralModel }}Input is the generated list query schema.
type List{{ .PluralModel }}Input struct {
pagination.PageInput
{{- range .ListFields }}
	{{ .Name }} {{ .TypeExpr }} ` + "`form:\"{{ .FormName }}\" filter:\"{{ .FilterTag }}\"{{ if .Description }} description:\"{{ .Description }}\"{{ end }}`" + `
{{- end }}
{{- if .SearchFields }}
	Search string ` + "`form:\"search\" filter:\"{{ range $i, $field := .SearchFields }}{{ if $i }}|{{ end }}{{ $field }}{{ end }},like\" description:\"Keyword search\"`" + `
{{- end }}
{{- if .SortFields }}
	Sort string ` + "`form:\"sort\" order:\"{{ range $i, $field := .SortFields }}{{ if $i }}|{{ end }}{{ if eq $field.Alias $field.Column }}{{ $field.Alias }}{{ else }}{{ $field.Alias }}:{{ $field.Column }}{{ end }}{{ end }}\" description:\"Validated sort fields\"`" + `
{{- end }}
}

// Get{{ .ModelName }}Input identifies a single {{ .SingularLabel }} record.
type Get{{ .ModelName }}Input struct {
ID {{ .IDTypeExpr }} ` + "`path:\"id\" json:\"-\" binding:\"required\"`" + `
}

// Create{{ .ModelName }}Input is the generated request body for creating {{ .SingularLabel }} records.
type Create{{ .ModelName }}Input struct {
{{ range .CreateFields }}
	{{ .Name }} {{ .TypeExpr }} ` + "`json:\"{{ .JSONName }}\"{{ if .Binding }} binding:\"{{ .Binding }}\"{{ end }}{{ if .Description }} description:\"{{ .Description }}\"{{ end }}`" + `
{{ end }}
{{ range .Relations }}
{{- if not .ExistingInputField }}
	{{ .InputName }} {{ .InputType }} ` + "`json:\"{{ .InputJSONName }}\"`" + `
{{- end }}
{{ end }}
}

// Update{{ .ModelName }}Input is the generated request body for partially updating {{ .SingularLabel }} records.
type Update{{ .ModelName }}Input struct {
	ID {{ .IDTypeExpr }} ` + "`path:\"id\" json:\"-\" binding:\"required\"`" + `
{{ range .UpdateFields }}
	{{ .Name }} {{ .UpdateType }} ` + "`json:\"{{ .JSONName }}\"{{ if .Binding }} binding:\"{{ .Binding }}\"{{ end }}{{ if .Description }} description:\"{{ .Description }}\"{{ end }}`" + `
{{ end }}
{{ range .Relations }}
{{- if not .ExistingInputField }}
	{{ .InputName }} {{ .UpdateInputType }} ` + "`json:\"{{ .InputJSONName }}\"`" + `
{{- end }}
{{ end }}
}

// Delete{{ .ModelName }}Input identifies the {{ .SingularLabel }} record to delete.
type Delete{{ .ModelName }}Input struct {
ID {{ .IDTypeExpr }} ` + "`path:\"id\" json:\"-\" binding:\"required\"`" + `
}

// Register{{ .ModelName }}CRUDRoutes wires the generated CRUD handlers onto a router.
func Register{{ .ModelName }}CRUDRoutes(router *ninja.Router) {
ninja.Get(router, "/", List{{ .PluralModel }}, ninja.Summary("List {{ .PluralLabel }}"), ninja.Paginated[{{ .ModelName }}Out]())
ninja.Get(router, "/:id", Get{{ .ModelName }}, ninja.Summary("Get {{ .SingularLabel }}"))
ninja.Post(router, "/", Create{{ .ModelName }}, ninja.Summary("Create {{ .SingularLabel }}"), ninja.WithTransaction())
ninja.Patch(router, "/:id", Update{{ .ModelName }}, ninja.Summary("Patch {{ .SingularLabel }}"), ninja.WithTransaction())
ninja.Delete(router, "/:id", Delete{{ .ModelName }}, ninja.Summary("Delete {{ .SingularLabel }}"), ninja.WithTransaction())
}

// List{{ .PluralModel }} returns a paginated list of {{ .PluralLabel }}.
func List{{ .PluralModel }}(ctx *ninja.Context, in *List{{ .PluralModel }}Input) (*pagination.Page[{{ .ModelName }}Out], error) {
	db := orm.WithContext(ctx.Context)
{{- if .UseGormX }}
	repo := New{{ .ModelName }}Repo()
query, _ := gormx.NewQuery[{{ .ModelName }}]()
{{- range .Relations }}
query.Preload("{{ .Preload }}")
{{- end }}
{{- if or .ListFields .SearchFields }}
filterOpts, err := filter.BuildOptions(in)
if err != nil {
return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
}
{{- else }}
filterOpts := []gormx.DBOption{}
{{- end }}
{{- if .SortFields }}
if err := order.ApplyOrder(query, in); err != nil {
return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
}
{{- end }}
opts := append([]gormx.DBOption{gormx.UseDB(db)}, append(filterOpts, query.ToOptions()...)...)
items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
if err != nil {
return nil, err
}
{{- else }}
	query := db.Model(&{{ .ModelName }}{})
{{- if or .ListFields .SearchFields .SortFields }}
	var err error
{{- end }}
{{- range .Relations }}
	query = query.Preload("{{ .Preload }}")
{{- end }}
{{- if or .ListFields .SearchFields }}
	query, err = filter.ApplyDB(query, in)
	if err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
	}
{{- end }}
	countQuery := query.Session(&gorm.Session{})
{{- if .SortFields }}
	query, err = order.ApplyDB(query, in)
	if err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
	}
{{- end }}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []{{ .ModelName }}
	if err := query.Limit(in.GetSize()).Offset(in.Offset()).Find(&items).Error; err != nil {
		return nil, err
	}
{{- end }}
out := make([]{{ .ModelName }}Out, len(items))
for i, item := range items {
bound, err := {{ .ToOutFuncName }}(item)
if err != nil {
return nil, err
}
out[i] = *bound
}
return pagination.NewPage(out, total, in.PageInput), nil
}

// Get{{ .ModelName }} retrieves a single {{ .SingularLabel }} by primary key.
func Get{{ .ModelName }}(ctx *ninja.Context, in *Get{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
item, err := load{{ .ModelName }}ByID(orm.WithContext(ctx.Context), in.ID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}
return {{ .ToOutFuncName }}(item)
}

// Create{{ .ModelName }} inserts a new {{ .SingularLabel }} record.
func Create{{ .ModelName }}(ctx *ninja.Context, in *Create{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
	db := orm.WithContext(ctx.Context)
{{- if .UseGormX }}
	repo := New{{ .ModelName }}Repo()
{{- end }}
	item := &{{ .ModelName }}{}
{{ range .CreateFields }}
	item.{{ .Name }} = in.{{ .Name }}
{{ end }}
{{- if .UseGormX }}
	if err := repo.Insert(item, gormx.UseDB(db)); err != nil {
{{- else }}
	if err := db.Create(item).Error; err != nil {
{{- end }}
		return nil, err
	}
{{- range .Relations }}
{{- if .Collection }}
	if in.{{ .InputName }} != nil {
		if err := sync{{ $.ModelName }}{{ .FieldName }}Relations(db, item, in.{{ .InputName }}); err != nil {
			return nil, err
		}
	}
{{- else if .UseAssociationInput }}
	{
		var zero {{ .TargetIDType }}
		if in.{{ .InputName }} != zero {
			if err := sync{{ $.ModelName }}{{ .FieldName }}Relation(db, item, in.{{ .InputName }}); err != nil {
				return nil, err
			}
		}
	}
{{- end }}
{{- end }}
	loaded, err := load{{ .ModelName }}ByID(db, item.{{ .IDField }})
	if err != nil {
		return nil, err
	}
	return {{ .ToOutFuncName }}(loaded)
}

// Update{{ .ModelName }} partially updates a {{ .SingularLabel }} record by primary key.
func Update{{ .ModelName }}(ctx *ninja.Context, in *Update{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
db := orm.WithContext(ctx.Context)
item, err := load{{ .ModelName }}ByID(db, in.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	updates := map[string]interface{}{}
{{ range .UpdateFields }}
	if in.{{ .Name }} != nil {
		updates["{{ .ColumnName }}"] = *in.{{ .Name }}
	}
{{ end }}
	if len(updates) > 0 {
{{ if .UseGormX }}
		repo := New{{ .ModelName }}Repo()
{{ if .UseByIDMethods }}
		if err := repo.UpdateById(int(in.ID), updates, gormx.UseDB(db)); err != nil {
{{ else }}
		if err := repo.UpdateByOpts(updates, gormx.UseDB(db), gormx.Where("{{ .IDColumn }} = ?", in.ID)); err != nil {
{{ end }}
{{ else }}
		if err := db.Model(&{{ .ModelName }}{}).Where("{{ .IDColumn }} = ?", in.ID).Updates(updates).Error; err != nil {
{{ end }}
			return nil, err
		}
	}
{{- range .Relations }}
{{- if .Collection }}
	if in.{{ .InputName }} != nil {
		if err := sync{{ $.ModelName }}{{ .FieldName }}Relations(db, &item, *in.{{ .InputName }}); err != nil {
			return nil, err
		}
	}
{{- else if .UseAssociationInput }}
	if in.{{ .InputName }} != nil {
		if err := sync{{ $.ModelName }}{{ .FieldName }}Relation(db, &item, *in.{{ .InputName }}); err != nil {
			return nil, err
		}
	}
{{- end }}
{{- end }}
	if len(updates) == 0 {
{{- range .Relations }}
{{- if .Collection }}
		if in.{{ .InputName }} != nil {
			loaded, err := load{{ $.ModelName }}ByID(db, in.ID)
			if err != nil {
				return nil, err
			}
			return {{ $.ToOutFuncName }}(loaded)
		}
{{- else if .UseAssociationInput }}
		if in.{{ .InputName }} != nil {
			loaded, err := load{{ $.ModelName }}ByID(db, in.ID)
			if err != nil {
				return nil, err
			}
			return {{ $.ToOutFuncName }}(loaded)
		}
{{- end }}
{{- end }}
		return {{ .ToOutFuncName }}(item)
	}
	item, err = load{{ .ModelName }}ByID(db, in.ID)
	if err != nil {
		return nil, err
	}
return {{ .ToOutFuncName }}(item)
}

func load{{ .ModelName }}ByID(db *gorm.DB, id {{ .IDTypeExpr }}) ({{ .ModelName }}, error) {
{{- if .UseGormX }}
	repo := New{{ .ModelName }}Repo()
	opts := []gormx.DBOption{
		gormx.UseDB(db),
		gormx.Where("{{ .IDColumn }} = ?", id),
	}
{{- range .Relations }}
	opts = append(opts, func(db *gorm.DB) *gorm.DB {
		return db.Preload("{{ .Preload }}")
	})
{{- end }}
	return repo.SelectOneByOpts(opts...)
{{- else }}
	var item {{ .ModelName }}
	query := db.Model(&{{ .ModelName }}{})
{{- range .Relations }}
	query = query.Preload("{{ .Preload }}")
{{- end }}
	if err := query.Where("{{ .IDColumn }} = ?", id).First(&item).Error; err != nil {
		return {{ .ModelName }}{}, err
	}
	return item, nil
{{- end }}
}

{{- range .Relations }}
{{- if .Collection }}
func sync{{ $.ModelName }}{{ .FieldName }}Relations(db *gorm.DB, item *{{ $.ModelName }}, ids []{{ .TargetIDType }}) error {
	if item == nil {
		return nil
	}
	related, err := load{{ $.ModelName }}{{ .FieldName }}Relations(db, ids)
	if err != nil {
		return err
	}
	return db.Model(item).Association("{{ .FieldName }}").Replace(related)
}

func load{{ $.ModelName }}{{ .FieldName }}Relations(db *gorm.DB, ids []{{ .TargetIDType }}) ([]{{ .TargetModel }}, error) {
	if len(ids) == 0 {
		return []{{ .TargetModel }}{}, nil
	}
	var related []{{ .TargetModel }}
	if err := db.Where("{{ .TargetIDColumn }} IN ?", ids).Find(&related).Error; err != nil {
		return nil, err
	}
	byID := make(map[{{ .TargetIDType }}]{{ .TargetModel }}, len(related))
	for _, item := range related {
		byID[item.{{ .TargetIDField }}] = item
	}
	out := make([]{{ .TargetModel }}, 0, len(ids))
	seen := map[{{ .TargetIDType }}]struct{}{}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		item, ok := byID[id]
		if !ok {
			return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("relation %q record %v not found", "{{ .FieldName }}", id))
		}
		out = append(out, item)
	}
	return out, nil
}
{{- else if .UseAssociationInput }}
func sync{{ $.ModelName }}{{ .FieldName }}Relation(db *gorm.DB, item *{{ $.ModelName }}, id {{ .TargetIDType }}) error {
	if item == nil {
		return nil
	}
	var zero {{ .TargetIDType }}
	if id == zero {
		return db.Model(item).Association("{{ .FieldName }}").Clear()
	}
	var related {{ .TargetModel }}
	if err := db.Where("{{ .TargetIDColumn }} = ?", id).First(&related).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("relation %q record %v not found", "{{ .FieldName }}", id))
		}
		return err
	}
	return db.Model(item).Association("{{ .FieldName }}").Replace(&related)
}
{{- end }}
{{- end }}

// Delete{{ .ModelName }} removes a {{ .SingularLabel }} record by primary key.
func Delete{{ .ModelName }}(ctx *ninja.Context, in *Delete{{ .ModelName }}Input) error {
db := orm.WithContext(ctx.Context)
{{ if .UseGormX }}
repo := New{{ .ModelName }}Repo()
{{ if .UseByIDMethods }}
if _, err := repo.SelectOneById(int(in.ID), gormx.UseDB(db)); err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
return repo.DeleteById(int(in.ID), gormx.UseDB(db))
{{ else }}
if _, err := repo.SelectOneByOpts(gormx.UseDB(db), gormx.Where("{{ .IDColumn }} = ?", in.ID)); err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
return repo.DeleteByOpts(gormx.UseDB(db), gormx.Where("{{ .IDColumn }} = ?", in.ID))
{{ end }}
{{ else }}
if _, err := load{{ .ModelName }}ByID(db, in.ID); err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
return db.Model(&{{ .ModelName }}{}).Where("{{ .IDColumn }} = ?", in.ID).Delete(&{{ .ModelName }}{}).Error
{{ end }}
}
`))
