package ninja

import (
	"reflect"
	"strings"
	"testing"
)

type coverageSchemaProfile struct {
	Bio string `json:"bio" binding:"required"`
}

type coverageSchemaModel struct {
	ID        uint                    `json:"id" gorm:"primaryKey"`
	Name      string                  `json:"name" binding:"required"`
	Secret    string                  `json:"secret" ninja:"write_only"`
	Profile   *coverageSchemaProfile  `json:"profile"`
	Tags      []coverageSchemaProfile `json:"tags"`
	Embedded  coverageSchemaProfile   `json:"embedded" gorm:"embedded"`
	Ignored   coverageSchemaProfile   `json:"ignored" gorm:"-"`
	CreatedAt string                  `json:"created_at" gorm:"autoCreateTime"`
}

type coverageSchemaOut struct {
	ModelSchema[coverageSchemaModel] `fields:"id,name,profile,tags" mode:"read" depth:"1"`
}

func TestModelSchemaCoverageForInputAndDescriptorEdges(t *testing.T) {
	descriptor := ModelSchemaOf[coverageSchemaModel](SchemaMode("output"), Depth(-1))
	if descriptor.filter.mode != ModelSchemaModeRead || descriptor.filter.depth != 0 {
		t.Fatalf("expected aliases and negative depth to normalize, got %+v", descriptor.filter)
	}
	if ModelReadSchemaOf[coverageSchemaModel]().filter.mode != ModelSchemaModeRead {
		t.Fatal("expected read constructor to set read mode")
	}
	if _, ok := resolveModelSchemaDescriptor(nil); ok {
		t.Fatal("expected nil descriptor type to miss")
	}
	if _, ok := resolveModelSchemaDescriptor(reflect.TypeOf(struct{ Name string }{})); ok {
		t.Fatal("expected plain struct descriptor to miss")
	}

	var model coverageSchemaModel
	if _, err := descriptor.BindInput(123); err == nil || !strings.Contains(err.Error(), "must be a struct") {
		t.Fatalf("expected non-struct input error, got %v", err)
	}
	if err := descriptor.ApplyInput(nil, struct{}{}); err == nil || !strings.Contains(err.Error(), "target is nil") {
		t.Fatalf("expected nil model target error, got %v", err)
	}
	var nilInput *struct{}
	if err := descriptor.ApplyInput(&model, nilInput); err == nil || !strings.Contains(err.Error(), "value is nil") {
		t.Fatalf("expected nil input error, got %v", err)
	}
	var nilModel *coverageSchemaModel
	if err := applyModelSchemaInput(reflect.ValueOf(nilModel), reflect.ValueOf(struct{}{}), modelSchemaFilter{}); err == nil || !strings.Contains(err.Error(), "target is nil") {
		t.Fatalf("expected nil reflected model target error, got %v", err)
	}
	if err := applyModelSchemaInput(reflect.ValueOf(new(int)), reflect.ValueOf(struct{}{}), modelSchemaFilter{}); err == nil || !strings.Contains(err.Error(), "target must be a struct") {
		t.Fatalf("expected non-struct target error, got %v", err)
	}
	if err := applyModelSchemaInput(reflect.ValueOf(&model), reflect.Value{}, modelSchemaFilter{}); err == nil || !strings.Contains(err.Error(), "value is invalid") {
		t.Fatalf("expected invalid input error, got %v", err)
	}

	type CoverageEmbeddedInput struct {
		Name *string `json:"name"`
	}
	type updateInput struct {
		*CoverageEmbeddedInput
		Secret *string `json:"secret"`
	}
	name := "updated"
	secret := "hidden"
	if err := ModelUpdateSchemaOf[coverageSchemaModel]().ApplyInput(&model, updateInput{
		CoverageEmbeddedInput: &CoverageEmbeddedInput{Name: &name},
		Secret:                &secret,
	}); err != nil {
		t.Fatalf("ApplyInput embedded pointer: %v", err)
	}
	if err := ModelUpdateSchemaOf[coverageSchemaModel]().ApplyInput(&model, &updateInput{Secret: &secret}); err != nil {
		t.Fatalf("ApplyInput pointer input: %v", err)
	}
	if model.Name != "updated" || model.Secret != "hidden" {
		t.Fatalf("expected embedded and direct update fields to apply, got %+v", model)
	}

	type CoverageEmbeddedModel struct {
		Alias string `json:"alias"`
	}
	type modelWithEmbeddedPointer struct {
		*CoverageEmbeddedModel
		Name string `json:"name"`
	}
	type pointerInput struct {
		Alias string `json:"alias"`
		Name  string `json:"name"`
	}
	var nested modelWithEmbeddedPointer
	if err := ModelSchemaOf[modelWithEmbeddedPointer]().ApplyInput(&nested, pointerInput{Alias: "a", Name: "n"}); err != nil {
		t.Fatalf("ApplyInput allocated anonymous pointer: %v", err)
	}
	if nested.CoverageEmbeddedModel == nil || nested.Alias != "a" || nested.Name != "n" {
		t.Fatalf("expected anonymous pointer model field to be allocated and set, got %+v", nested)
	}

	if modelSchemaInputValuePresent(reflect.Value{}) {
		t.Fatal("expected invalid reflected input value to be absent")
	}
	if !containsModelSchemaName([]string{"name"}, "", "name") {
		t.Fatal("expected empty candidate to be skipped before matching name")
	}

	type skippedInput struct {
		private string
		Skip    string `json:"-"`
		Keep    string `json:"keep"`
	}
	fields := map[string]reflect.Value{}
	collectModelSchemaInputFields(reflect.ValueOf(skippedInput{private: "x", Skip: "skip", Keep: "keep"}), fields)
	if _, ok := fields["private"]; ok {
		t.Fatalf("expected unexported input field to be skipped, got %+v", fields)
	}
	if _, ok := fields["Skip"]; ok {
		t.Fatalf("expected json '-' input field to be skipped, got %+v", fields)
	}
	if _, ok := fields["keep"]; !ok {
		t.Fatalf("expected normal input field to be collected, got %+v", fields)
	}

	var unaddressable coverageSchemaModel
	if err := applyModelSchemaInputFields(reflect.ValueOf(unaddressable), modelSchemaFilter{}, map[string]reflect.Value{"name": reflect.ValueOf("x")}); err == nil || !strings.Contains(err.Error(), "cannot be set") {
		t.Fatalf("expected unsettable model field error, got %v", err)
	}
	if err := applyModelSchemaInputFields(reflect.ValueOf(&unaddressable).Elem(), modelSchemaFilter{}, map[string]reflect.Value{"name": reflect.ValueOf(struct{}{})}); err == nil || !strings.Contains(err.Error(), "cannot assign") {
		t.Fatalf("expected incompatible input field error, got %v", err)
	}
}

func TestModelSchemaCoverageForHelperEdges(t *testing.T) {
	if parseModelSchemaDepthTag("bad") != 0 || parseModelSchemaDepthTag("-5") != 0 {
		t.Fatal("expected invalid and negative depth tags to normalize to zero")
	}
	aliases := map[ModelSchemaMode]ModelSchemaMode{
		"details":  ModelSchemaModeDetail,
		"creation": ModelSchemaModeCreate,
		"insert":   ModelSchemaModeCreate,
		"patch":    ModelSchemaModeUpdate,
		"custom":   "custom",
	}
	for raw, want := range aliases {
		if got := normalizeModelSchemaMode(raw); got != want {
			t.Fatalf("normalizeModelSchemaMode(%q) = %q, want %q", raw, got, want)
		}
	}
	if child := (modelSchemaFilter{}).child(); !child.isZero() {
		t.Fatalf("expected empty child filter, got %+v", child)
	}
	if child := (modelSchemaFilter{mode: "OUTPUT", depth: 2}).child(); child.mode != ModelSchemaModeRead || child.depth != 1 {
		t.Fatalf("expected child filter to normalize mode and decrement depth, got %+v", child)
	}

	type accessModel struct {
		CreateBlocked string `json:"create_blocked" gorm:"<-:update"`
		UpdateBlocked string `json:"update_blocked" gorm:"<-:create"`
		Dashed        string `json:"dashed" ninja:"create-only"`
		Readable      string `json:"readable"`
	}
	tp := reflect.TypeOf(accessModel{})
	if modelSchemaWritableField(tp.Field(0), "create_blocked", ModelSchemaModeCreate) {
		t.Fatal("expected create mode to reject update-only gorm field")
	}
	if modelSchemaWritableField(tp.Field(1), "update_blocked", ModelSchemaModeUpdate) {
		t.Fatal("expected update mode to reject create-only gorm field")
	}
	if !modelSchemaWritableField(tp.Field(2), "dashed", ModelSchemaModeCreate) {
		t.Fatal("expected dashed create-only tag to be normalized")
	}
	if !modelSchemaModeIncludes("unexpected", tp.Field(3), "readable") {
		t.Fatal("expected unknown schema mode to include fields")
	}
	if !modelSchemaHasGORMSetting("embeddedPrefix:profile_;column:name", "embedded_prefix") {
		t.Fatal("expected gorm setting lookup to normalize underscores")
	}
	if modelSchemaListField(reflect.TypeOf(struct{ Name string }{})) {
		t.Fatal("expected plain struct not to be a list scalar field")
	}
	if !modelSchemaListField(reflect.TypeOf((*string)(nil))) {
		t.Fatal("expected pointer to scalar to be a list field")
	}
	if modelSchemaImplementsMarshaler(reflect.TypeOf((*struct{ Name string })(nil))) {
		t.Fatal("expected pointer to plain struct not to implement marshaling")
	}
	if !modelSchemaNestedField(reflect.TypeOf([]coverageSchemaProfile{})) {
		t.Fatal("expected slice of structs to be a nested schema field")
	}
	if got := modelSchemaRelationType(reflect.TypeOf([]*coverageSchemaProfile{})); got != reflect.TypeOf(coverageSchemaProfile{}) {
		t.Fatalf("unexpected relation type: %v", got)
	}
	if paths := appendModelSchemaPreloadPath([]string{"Owner"}, ""); !reflect.DeepEqual(paths, []string{"Owner"}) {
		t.Fatalf("expected blank preload path to be ignored, got %v", paths)
	}
	if paths := appendModelSchemaPreloadPath([]string{"Owner"}, "Owner"); !reflect.DeepEqual(paths, []string{"Owner"}) {
		t.Fatalf("expected duplicate preload path to be ignored, got %v", paths)
	}
	if !modelSchemaReadOnlyField(reflect.TypeOf(coverageSchemaModel{}).Field(0), "id") {
		t.Fatal("expected primary key field to be read-only")
	}
	if isJSONOmitEmpty(reflect.TypeOf(struct{ Name string }{}).Field(0)) {
		t.Fatal("expected field without json tag not to be omitempty")
	}
}

func TestModelSchemaCoverageForBindingPreloadsAndSerialization(t *testing.T) {
	if out, ok, err := bindModelSchemaForType(reflect.TypeOf(ModelSchema[coverageSchemaModel]{}), coverageSchemaModel{Name: "direct"}); err != nil || !ok {
		t.Fatalf("expected direct ModelSchema binding, out=%+v ok=%v err=%v", out, ok, err)
	} else if schema, ok := out.(*ModelSchema[coverageSchemaModel]); !ok || schema.Model.Name != "direct" {
		t.Fatalf("unexpected direct ModelSchema binding output: %+v", out)
	}

	if _, _, err := bindModelSchemaForType(reflect.TypeOf(coverageSchemaOut{}), 123); err == nil || !strings.Contains(err.Error(), "cannot assign model type") {
		t.Fatalf("expected embedded schema assignment error, got %v", err)
	}
	if _, _, ok := modelSchemaBindingField(reflect.ValueOf(struct{ Name string }{}), reflect.TypeOf(struct{ Name string }{})); ok {
		t.Fatal("expected binding field lookup to miss plain structs")
	}
	if _, ok, err := bindModelSchemaForType(reflect.TypeOf(0), coverageSchemaModel{}); ok || err != nil {
		t.Fatalf("expected non-struct schema type to miss, ok=%v err=%v", ok, err)
	}
	type pointerEmbeddedSchema struct {
		*ModelSchema[coverageSchemaModel]
	}
	if out, ok, err := bindModelSchemaForType(reflect.TypeOf(pointerEmbeddedSchema{}), coverageSchemaModel{Name: "ptr"}); err != nil || !ok {
		t.Fatalf("expected pointer embedded schema binding, out=%+v ok=%v err=%v", out, ok, err)
	} else if schema := out.(*pointerEmbeddedSchema); schema.ModelSchema == nil || schema.Model.Name != "ptr" {
		t.Fatalf("unexpected pointer embedded schema output: %+v", out)
	}
	if _, err := BindModelSchema[coverageSchemaOut](123); err == nil || !strings.Contains(err.Error(), "cannot assign") {
		t.Fatalf("expected public BindModelSchema assignment error, got %v", err)
	}

	type CoverageEmbeddedRelations struct {
		Audit coverageSchemaProfile `json:"audit"`
	}
	type preloadModel struct {
		CoverageEmbeddedRelations
		Owner   *coverageSchemaProfile  `json:"owner"`
		Members []coverageSchemaProfile `json:"members"`
		Local   coverageSchemaProfile   `json:"local" gorm:"embedded"`
		Ignored coverageSchemaProfile   `json:"ignored" gorm:"-"`
	}
	preloads := ModelSchemaOf[preloadModel](Depth(2)).Preloads()
	for _, want := range []string{"Audit", "Owner", "Members"} {
		if !containsCoverageString(preloads, want) {
			t.Fatalf("expected preload %q in %v", want, preloads)
		}
	}
	if containsCoverageString(preloads, "Local") || containsCoverageString(preloads, "Ignored") {
		t.Fatalf("expected embedded and ignored fields not to become preloads, got %v", preloads)
	}
	var directPaths []string
	collectModelSchemaPreloads(reflect.TypeOf(123), modelSchemaFilter{depth: 1}, "", &directPaths)
	collectModelSchemaPreloads(reflect.TypeOf(struct {
		private coverageSchemaProfile
	}{}), modelSchemaFilter{depth: 1}, "", &directPaths)
	if len(directPaths) != 0 {
		t.Fatalf("expected non-struct and unexported preload fields to be ignored, got %v", directPaths)
	}

	type nested struct {
		Child coverageSchemaProfile `json:"child"`
	}
	serialized, err := serializeModelSchemaElement(reflect.ValueOf(nested{Child: coverageSchemaProfile{Bio: "nested"}}), modelSchemaFilter{depth: 1})
	if err != nil {
		t.Fatalf("serializeModelSchemaElement nested: %v", err)
	}
	if child, ok := serialized.(map[string]any)["child"].(map[string]any); !ok || child["bio"] != "nested" {
		t.Fatalf("expected nested struct to serialize through child filter, got %+v", serialized)
	}
	if value, err := serializeModelSchemaValue(reflect.ValueOf(&coverageSchemaProfile{Bio: "ptr"}), modelSchemaFilter{}); err != nil {
		t.Fatalf("serialize pointer value: %v", err)
	} else if value.(map[string]any)["bio"] != "ptr" {
		t.Fatalf("unexpected pointer serialization: %+v", value)
	}
	if value, err := serializeModelSchemaValue(reflect.ValueOf(7), modelSchemaFilter{}); err != nil || value != 7 {
		t.Fatalf("unexpected scalar serialization: value=%v err=%v", value, err)
	}
	if value, err := serializeModelSchemaElement(reflect.ValueOf((*coverageSchemaProfile)(nil)), modelSchemaFilter{}); err != nil || value != nil {
		t.Fatalf("unexpected nil element serialization: value=%v err=%v", value, err)
	}
	if value, err := serializeModelSchemaElement(reflect.ValueOf(textValue("element")), modelSchemaFilter{}); err != nil || value == nil {
		t.Fatalf("expected text marshaler element to be preserved, value=%v err=%v", value, err)
	}
	holder := struct {
		Text textValue
	}{Text: "addr"}
	if value, ok := preserveCustomJSONValue(reflect.ValueOf(&holder).Elem().Field(0)); !ok || value == nil {
		t.Fatalf("expected addressable text marshaler to be preserved, value=%v ok=%v", value, ok)
	}
}

func TestModelSchemaCoverageForAssignmentConversions(t *testing.T) {
	type aliasInt int
	var int64Target int64
	if err := assignModelSchemaModel(reflect.ValueOf(&int64Target).Elem(), reflect.ValueOf(aliasInt(7))); err != nil || int64Target != 7 {
		t.Fatalf("expected convertible assignment, target=%d err=%v", int64Target, err)
	}
	var intTarget int
	intValue := 8
	if err := assignModelSchemaModel(reflect.ValueOf(&intTarget).Elem(), reflect.ValueOf(&intValue)); err != nil || intTarget != 8 {
		t.Fatalf("expected pointer elem assignment, target=%d err=%v", intTarget, err)
	}
	var convertedTarget int64
	aliasValue := aliasInt(9)
	if err := assignModelSchemaModel(reflect.ValueOf(&convertedTarget).Elem(), reflect.ValueOf(&aliasValue)); err != nil || convertedTarget != 9 {
		t.Fatalf("expected pointer elem conversion, target=%d err=%v", convertedTarget, err)
	}
	var pointerTarget *int64
	if err := assignModelSchemaModel(reflect.ValueOf(&pointerTarget).Elem(), reflect.ValueOf(aliasInt(10))); err != nil || pointerTarget == nil || *pointerTarget != 10 {
		t.Fatalf("expected destination pointer conversion, target=%v err=%v", pointerTarget, err)
	}
	if err := assignModelSchemaModel(reflect.ValueOf(&pointerTarget).Elem(), reflect.Value{}); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid source assignment error, got %v", err)
	}
}

func containsCoverageString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
