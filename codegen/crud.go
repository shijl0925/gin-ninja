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
	packageName   string
	modelName     string
	tag           string
	fields        []fieldSpec
	outputFields  []string
	createFields  []fieldSpec
	updateFields  []fieldSpec
	imports       []importSpec
	idTypeExpr    string
	pluralModel   string
	singularLabel string
	pluralLabel   string
}

type fieldSpec struct {
	Name        string
	TypeExpr    string
	UpdateType  string
	JSONName    string
	Binding     string
	Description string
}

type importSpec struct {
	Alias string
	Path  string
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
	CreateFields      []fieldSpec
	UpdateFields      []fieldSpec
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
		CreateFields:      model.createFields,
		UpdateFields:      model.updateFields,
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

	var structType *ast.StructType
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != cfg.Model {
				continue
			}
			candidate, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return modelSpec{}, fmt.Errorf("model %q is not a struct", cfg.Model)
			}
			structType = candidate
			break
		}
	}
	if structType == nil {
		return modelSpec{}, fmt.Errorf("model %q not found in %s", cfg.Model, cfg.ModelFile)
	}

	packageName := strings.TrimSpace(cfg.PackageName)
	if packageName == "" {
		packageName = file.Name.Name
	}

	collector := newImportCollector(importAliases)
	fields := make([]fieldSpec, 0, len(structType.Fields.List))
	outputFields := []string{}
	createFields := []fieldSpec{}
	updateFields := []fieldSpec{}
	idTypeExpr := "string"
	idResolved := false
	hasEmbeddedGormModel := false

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

			typeExpr := exprString(fset, field.Type)
			collector.addExpr(field.Type)
			jsonName, hidden := resolveJSONName(name.Name, tags)
			if isIDField(name.Name, jsonName, gormTag) && !idResolved {
				idTypeExpr = typeExpr
				idResolved = true
			}

			if hidden || shouldSkipOutputField(gormTag) || !isRenderableFieldType(field.Type, importPaths) {
				continue
			}

			binding := strings.TrimSpace(tags.Get("binding"))
			description := strings.TrimSpace(tags.Get("description"))
			fieldSpec := fieldSpec{
				Name:        name.Name,
				TypeExpr:    typeExpr,
				UpdateType:  optionalType(typeExpr),
				JSONName:    jsonName,
				Binding:     binding,
				Description: description,
			}
			fields = append(fields, fieldSpec)
			outputFields = append(outputFields, jsonName)

			if isWritableField(name.Name, field.Type, gormTag, importPaths) {
				createFields = append(createFields, fieldSpec)
				fieldSpec.Binding = normalizeUpdateBinding(binding)
				updateFields = append(updateFields, fieldSpec)
			}
		}
	}

	if hasEmbeddedGormModel && !contains(outputFields, "id") {
		outputFields = append([]string{"id"}, outputFields...)
	}
	outputFields = uniqueStrings(outputFields)

	imports := []importSpec{
		{Alias: "", Path: "errors"},
		{Alias: "", Path: "github.com/shijl0925/gin-ninja/orm"},
		{Alias: "ninja", Path: "github.com/shijl0925/gin-ninja"},
		{Alias: "", Path: "github.com/shijl0925/gin-ninja/pagination"},
		{Alias: "", Path: "gorm.io/gorm"},
	}
	imports = append(imports, collector.list()...)
	imports = uniqueImports(imports)

	tag := strings.TrimSpace(cfg.Tag)
	pluralModel := pluralizeName(cfg.Model)
	if tag == "" {
		tag = pluralModel
	}

	return modelSpec{
		packageName:   packageName,
		modelName:     cfg.Model,
		tag:           tag,
		fields:        fields,
		outputFields:  outputFields,
		createFields:  createFields,
		updateFields:  updateFields,
		imports:       imports,
		idTypeExpr:    idTypeExpr,
		pluralModel:   pluralModel,
		singularLabel: lowerLabel(cfg.Model),
		pluralLabel:   lowerLabel(pluralModel),
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

func isIDField(name, jsonName, gormTag string) bool {
	if name == "ID" || jsonName == "id" {
		return true
	}
	lower := strings.ToLower(gormTag)
	return strings.Contains(lower, "primarykey") || strings.Contains(lower, "primary_key")
}

func isWritableField(name string, expr ast.Expr, gormTag string, imports map[string]string) bool {
	if isReadOnlyName(name) || hasDisallowedGORMTag(gormTag) {
		return false
	}
	return isRenderableFieldType(expr, imports)
}

func shouldSkipOutputField(gormTag string) bool {
	lower := strings.ToLower(gormTag)
	return strings.Contains(lower, "-") ||
		strings.Contains(lower, "many2many") ||
		strings.Contains(lower, "foreignkey") ||
		strings.Contains(lower, "references")
}

func isReadOnlyName(name string) bool {
	switch name {
	case "ID", "CreatedAt", "UpdatedAt", "DeletedAt":
		return true
	default:
		return false
	}
}

func hasDisallowedGORMTag(tag string) bool {
	lower := strings.ToLower(tag)
	return strings.Contains(lower, "-") ||
		strings.Contains(lower, "primarykey") ||
		strings.Contains(lower, "primary_key") ||
		strings.Contains(lower, "autocreatetime") ||
		strings.Contains(lower, "autoupdatetime") ||
		strings.Contains(lower, "many2many") ||
		strings.Contains(lower, "foreignkey") ||
		strings.Contains(lower, "references")
}

func isRenderableFieldType(expr ast.Expr, imports map[string]string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name != "interface{}" && t.Name != "any"
	case *ast.StarExpr:
		return isRenderableFieldType(t.X, imports)
	case *ast.ArrayType:
		return isRenderableFieldType(t.Elt, imports)
	case *ast.SelectorExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return false
		}
		path := imports[ident.Name]
		return path == "time"
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

func optionalType(typeExpr string) string {
	if strings.HasPrefix(strings.TrimSpace(typeExpr), "*") {
		return typeExpr
	}
	return "*" + typeExpr
}

func pluralizeName(name string) string {
	if name == "" {
		return name
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "y") && len(name) > 1 && !isVowel(rune(lower[len(lower)-2])):
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
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
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
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, "")
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
}

// List{{ .PluralModel }}Input is the generated list query schema.
type List{{ .PluralModel }}Input struct {
pagination.PageInput
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
}

// Update{{ .ModelName }}Input is the generated request body for updating {{ .SingularLabel }} records.
type Update{{ .ModelName }}Input struct {
	ID {{ .IDTypeExpr }} ` + "`path:\"id\" json:\"-\" binding:\"required\"`" + `
{{ range .UpdateFields }}
	{{ .Name }} {{ .UpdateType }} ` + "`json:\"{{ .JSONName }}\"{{ if .Binding }} binding:\"{{ .Binding }}\"{{ end }}{{ if .Description }} description:\"{{ .Description }}\"{{ end }}`" + `
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
ninja.Post(router, "/", Create{{ .ModelName }}, ninja.Summary("Create {{ .SingularLabel }}"))
ninja.Put(router, "/:id", Update{{ .ModelName }}, ninja.Summary("Update {{ .SingularLabel }}"))
ninja.Delete(router, "/:id", Delete{{ .ModelName }}, ninja.Summary("Delete {{ .SingularLabel }}"))
}

// List{{ .PluralModel }} returns a paginated list of {{ .PluralLabel }}.
func List{{ .PluralModel }}(ctx *ninja.Context, in *List{{ .PluralModel }}Input) (*pagination.Page[{{ .ModelName }}Out], error) {
db := orm.WithContext(ctx.Context)
query := db.Model(&{{ .ModelName }}{})

var total int64
if err := query.Count(&total).Error; err != nil {
return nil, err
}

var items []{{ .ModelName }}
if err := query.Offset(in.Offset()).Limit(in.Limit()).Find(&items).Error; err != nil {
return nil, err
}

out := make([]{{ .ModelName }}Out, len(items))
for i, item := range items {
bound, err := ninja.BindModelSchema[{{ .ModelName }}Out](item)
if err != nil {
return nil, err
}
out[i] = *bound
}
return pagination.NewPage(out, total, in.PageInput), nil
}

// Get{{ .ModelName }} retrieves a single {{ .SingularLabel }} by primary key.
func Get{{ .ModelName }}(ctx *ninja.Context, in *Get{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
var item {{ .ModelName }}
if err := orm.WithContext(ctx.Context).First(&item, in.ID).Error; err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}
return ninja.BindModelSchema[{{ .ModelName }}Out](item)
}

// Create{{ .ModelName }} inserts a new {{ .SingularLabel }} record.
func Create{{ .ModelName }}(ctx *ninja.Context, in *Create{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
	item := {{ .ModelName }}{}
{{ range .CreateFields }}
	item.{{ .Name }} = in.{{ .Name }}
{{ end }}
	if err := orm.WithContext(ctx.Context).Create(&item).Error; err != nil {
		return nil, err
	}
return ninja.BindModelSchema[{{ .ModelName }}Out](item)
}

// Update{{ .ModelName }} updates a {{ .SingularLabel }} record by primary key.
func Update{{ .ModelName }}(ctx *ninja.Context, in *Update{{ .ModelName }}Input) (*{{ .ModelName }}Out, error) {
db := orm.WithContext(ctx.Context)
var item {{ .ModelName }}
	if err := db.First(&item, in.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
{{ range .UpdateFields }}
	if in.{{ .Name }} != nil {
		item.{{ .Name }} = *in.{{ .Name }}
	}
{{ end }}
	if err := db.Save(&item).Error; err != nil {
		return nil, err
	}
return ninja.BindModelSchema[{{ .ModelName }}Out](item)
}

// Delete{{ .ModelName }} removes a {{ .SingularLabel }} record by primary key.
func Delete{{ .ModelName }}(ctx *ninja.Context, in *Delete{{ .ModelName }}Input) error {
db := orm.WithContext(ctx.Context)
var item {{ .ModelName }}
if err := db.First(&item, in.ID).Error; err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
return db.Delete(&item).Error
}
`))
