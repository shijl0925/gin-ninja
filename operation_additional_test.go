package ninja

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/pagination"
)

func TestOperationCoverageForResponseHelperEdges(t *testing.T) {
	if !isNilResponse(nil) {
		t.Fatal("expected nil response to be nil")
	}
	var nilMap map[string]string
	if !isNilResponse(nilMap) {
		t.Fatal("expected nil map response to be nil")
	}
	if isNilResponse(0) {
		t.Fatal("expected scalar zero response not to be nil")
	}

	if (&operation{}).usesDirectResponseWriter() {
		t.Fatal("expected empty operation not to use direct response writer")
	}
	if !(&operation{stream: operationStream{config: &streamConfig{}}}).usesDirectResponseWriter() {
		t.Fatal("expected stream operation to use direct response writer")
	}
	if !(&operation{route: operationRoute{outputType: reflect.TypeOf(Download{})}}).usesDirectResponseWriter() {
		t.Fatal("expected Download value output to use direct response writer")
	}
	if !(&operation{route: operationRoute{outputType: reflect.TypeOf(&Download{})}}).usesDirectResponseWriter() {
		t.Fatal("expected Download pointer output to use direct response writer")
	}

	op := &operation{}
	appendOperationAuthMiddleware(op)
	if len(op.route.ginMiddleware) != 0 {
		t.Fatalf("expected no middleware to be appended, got %d", len(op.route.ginMiddleware))
	}
	appendOperationAuthMiddleware(op, func(*gin.Context) {})
	if len(op.route.ginMiddleware) != 1 {
		t.Fatalf("expected middleware to be appended, got %d", len(op.route.ginMiddleware))
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected nil auth middleware to panic")
			}
		}()
		appendOperationAuthMiddleware(op, nil)
	}()

	type envelope struct {
		Items   []schemaModel `json:"items"`
		Skip    string        `json:"-"`
		Empty   string        `json:"empty,omitempty"`
		Text    textValue     `json:"text"`
		Count   int           `json:"count"`
		private string
	}
	body, err := serializePaginatedResponseEnvelope(reflect.ValueOf(envelope{
		Items:   []schemaModel{{ID: 1, Email: "hidden@example.com"}},
		Text:    "demo",
		Count:   2,
		private: "ignored",
	}), []any{map[string]any{"email": "public@example.com"}})
	if err != nil {
		t.Fatalf("serializePaginatedResponseEnvelope: %v", err)
	}
	if _, ok := body["Skip"]; ok {
		t.Fatalf("expected json '-' field to be skipped, got %+v", body)
	}
	if _, ok := body["empty"]; ok {
		t.Fatalf("expected omitempty zero field to be skipped, got %+v", body)
	}
	if _, ok := body["private"]; ok {
		t.Fatalf("expected unexported field to be skipped, got %+v", body)
	}
	if items, ok := body["items"].([]any); !ok || len(items) != 1 {
		t.Fatalf("expected serialized items replacement, got %+v", body["items"])
	}
	if body["text"] == nil || body["count"] != 2 {
		t.Fatalf("expected custom JSON and regular fields, got %+v", body)
	}
}

func TestOperationCoverageForSerializationErrorEdges(t *testing.T) {
	schemaDescriptor := ModelSchemaOf[requiredSchemaModel]().Fields("id", "name", "email").schemaDescriptor()
	if _, err := serializeModelSchemaResponse(schemaDescriptor, 123); err == nil || !strings.Contains(err.Error(), "cannot serialize response type") {
		t.Fatalf("expected schema response type error, got %v", err)
	}
	if err := validateModelSchemaResponseType(reflect.TypeOf(schemaModel{}), nil); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid model schema response type error, got %v", err)
	}
	if _, err := serializeModelSchemaResponse(schemaDescriptor, requiredSchemaModel{ID: 1, Email: "missing@example.com"}); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected schema response required field error, got %v", err)
	}
	if err := validateModelSchemaRequiredFields(reflect.ValueOf([]requiredSchemaModel{{ID: 2}}), modelSchemaFilter{fields: []string{"name"}}); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected slice required field error, got %v", err)
	}
	type nestedRequired struct {
		Child requiredSchemaModel `json:"child"`
	}
	if err := validateModelSchemaRequiredFields(reflect.ValueOf(nestedRequired{Child: requiredSchemaModel{ID: 3}}), modelSchemaFilter{depth: 1}); err == nil || !strings.Contains(err.Error(), "child.name is required") {
		t.Fatalf("expected nested required field path, got %v", err)
	}
	if modelSchemaFieldPath("parent", "child") != "parent.child" {
		t.Fatal("expected nested model schema field path")
	}
	if isModelSchemaResponseType(nil, reflect.TypeOf(schemaModel{})) || isModelSchemaResponseType(reflect.TypeOf(schemaModel{}), nil) {
		t.Fatal("expected nil schema response types to miss")
	}
	if !isModelSchemaResponseType(reflect.TypeOf(schemaModel{}), reflect.TypeOf([]*schemaModel{})) {
		t.Fatal("expected slice pointer item type to match model schema target")
	}

	page := pagination.NewPage([]schemaModel{{ID: 1, Name: "Ada", Email: "ada@example.com"}}, 1, pagination.PageInput{Page: 1, Size: 10})
	if serialized, err := serializePaginatedModelSchemaResponse(ModelSchemaOf[schemaModel]().Fields("id", "email").schemaDescriptor(), page); err != nil {
		t.Fatalf("serializePaginatedModelSchemaResponse: %v", err)
	} else if items := serialized.(map[string]any)["items"].([]any); items[0].(map[string]any)["email"] != "ada@example.com" {
		t.Fatalf("unexpected paginated schema response: %+v", serialized)
	}
	for name, output := range map[string]any{
		"invalid": nil,
		"nil ptr": (*pagination.Page[schemaModel])(nil),
		"scalar":  123,
		"missing": struct{ Total int }{Total: 1},
	} {
		if _, err := serializePaginatedModelSchemaResponse(ModelSchemaOf[schemaModel]().schemaDescriptor(), output); err == nil {
			t.Fatalf("expected paginated schema response error for %s", name)
		}
	}
	if _, err := serializePaginatedModelSchemaResponse(ModelSchemaOf[schemaModel]().schemaDescriptor(), pagination.NewPage([]int{1}, 1, pagination.PageInput{})); err == nil || !strings.Contains(err.Error(), "cannot serialize response type") {
		t.Fatalf("expected paginated schema item type error, got %v", err)
	}
	if _, err := serializePaginatedModelSchemaResponse(ModelSchemaOf[requiredSchemaModel]().Fields("id", "name").schemaDescriptor(), pagination.NewPage([]requiredSchemaModel{{ID: 4}}, 1, pagination.PageInput{})); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected paginated schema required field error, got %v", err)
	}

	if _, err := serializePaginatedResponseModel(reflect.TypeOf(publicSchema{}), 123); err == nil || !strings.Contains(err.Error(), "must be a struct") {
		t.Fatalf("expected paginated response model scalar error, got %v", err)
	}
	if _, err := serializePaginatedResponseModel(reflect.TypeOf(publicSchema{}), (*pagination.Page[schemaModel])(nil)); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("expected paginated response model nil pointer error, got %v", err)
	}
	if _, err := serializePaginatedResponseModel(reflect.TypeOf(publicSchema{}), struct{ Total int }{Total: 1}); err == nil || !strings.Contains(err.Error(), "Items") {
		t.Fatalf("expected missing items error, got %v", err)
	}
	if _, err := serializePaginatedResponseModel(reflect.TypeOf(struct {
		Name string `binding:"required"`
	}{}), pagination.NewPage([]struct {
		Name string `binding:"required"`
	}{{}}, 1, pagination.PageInput{})); err == nil || !strings.Contains(err.Error(), "response schema validation failed") {
		t.Fatalf("expected paginated response model validation error, got %v", err)
	}
}

func TestOperationCoverageForResponseModelBindingEdges(t *testing.T) {
	itemType := reflect.TypeOf(publicSchema{})
	responseOp := &operation{spec: operationDocSpec{responseType: itemType}}
	if _, err := responseOp.serializeResponse(123); err == nil || !strings.Contains(err.Error(), "cannot assign") {
		t.Fatalf("expected serializeResponse to surface response model binding error, got %v", err)
	}
	if _, err := serializePaginatedResponseModel(itemType, nil); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected nil paginated response model error, got %v", err)
	}
	if out, err := bindResponseModel(reflect.TypeOf(int64(0)), int(7)); err != nil || out.(int64) != 7 {
		t.Fatalf("expected scalar response model coercion, out=%v err=%v", out, err)
	}
	if _, err := bindResponseModel(reflect.TypeOf(0), "not-int"); err == nil || !strings.Contains(err.Error(), "must be a struct") {
		t.Fatalf("expected non-struct response model error, got %v", err)
	}
	if _, err := bindResponseModel(reflect.TypeOf(struct{ Name string }{}), nil); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid response model error, got %v", err)
	}
	if out, err := bindResponseModel(reflect.SliceOf(itemType), []schemaModel{{ID: 3}}); err != nil {
		t.Fatalf("expected slice response model binding, err=%v", err)
	} else if reflect.ValueOf(out).Len() != 1 {
		t.Fatalf("expected one slice item, got %+v", out)
	}
	if out, err := bindResponseModelItems(reflect.ArrayOf(2, itemType), []schemaModel{{ID: 1}, {ID: 2}}); err != nil {
		t.Fatalf("bindResponseModelItems array: %v", err)
	} else if reflect.ValueOf(out).Len() != 2 {
		t.Fatalf("expected two array items, got %+v", out)
	}
	if _, err := bindResponseModelItems(reflect.ArrayOf(2, itemType), []schemaModel{{ID: 1}}); err == nil || !strings.Contains(err.Error(), "length") {
		t.Fatalf("expected array length mismatch error, got %v", err)
	}
	if _, err := bindResponseModelItems(reflect.SliceOf(itemType), (*[]schemaModel)(nil)); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("expected nil slice pointer error, got %v", err)
	}
	if _, err := bindResponseModelItems(reflect.SliceOf(itemType), 123); err == nil || !strings.Contains(err.Error(), "cannot serialize response type") {
		t.Fatalf("expected non-slice items error, got %v", err)
	}
	if _, err := bindResponseModelItems(reflect.SliceOf(itemType), []int{1}); err == nil || !strings.Contains(err.Error(), "response item 0") {
		t.Fatalf("expected item binding error, got %v", err)
	}
	if _, err := responseModelValueForType(itemType, nil); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid response model value error, got %v", err)
	}
	if value, err := responseModelValueForType(itemType, &publicSchema{}); err != nil || value.Type() != itemType {
		t.Fatalf("expected pointer response model value to dereference, value=%v err=%v", value, err)
	}
	if value, err := responseModelValueForType(reflect.TypeOf(int64(0)), int(12)); err != nil || value.Int() != 12 {
		t.Fatalf("expected convertible response model value, value=%v err=%v", value, err)
	}
	stringForResponse := "response"
	type responseAliasString string
	if value, err := responseModelValueForType(reflect.TypeOf(responseAliasString("")), &stringForResponse); err != nil || value.Interface().(responseAliasString) != "response" {
		t.Fatalf("expected pointer convertible response model value, value=%v err=%v", value, err)
	}
	if value, ok := coerceResponseValue(reflect.TypeOf(int64(0)), int(11)); !ok || value.(int64) != 11 {
		t.Fatalf("expected direct convertible response value, value=%v ok=%v", value, ok)
	}
	stringValue := "hello"
	type aliasString string
	if value, ok := coerceResponseValue(reflect.TypeOf(aliasString("")), &stringValue); !ok || value.(aliasString) != "hello" {
		t.Fatalf("expected pointer elem convertible response value, value=%v ok=%v", value, ok)
	}
	if _, ok := coerceResponseValue(reflect.TypeOf(int(0)), nil); ok {
		t.Fatal("expected invalid response value not to coerce")
	}
	if err := validateResponseModel(nil); err != nil {
		t.Fatalf("expected nil response model validation to pass, got %v", err)
	}
	if err := validateResponseModelValue(reflect.Value{}); err != nil {
		t.Fatalf("expected invalid reflected response model validation to pass, got %v", err)
	}
	if err := validateResponseModel((*struct {
		Name string `binding:"required"`
	})(nil)); err != nil {
		t.Fatalf("expected nil response model pointer validation to pass, got %v", err)
	}
	if err := validateResponseModel([]struct {
		Name string `binding:"required"`
	}{{}}); err == nil || !strings.Contains(err.Error(), "response schema validation failed") {
		t.Fatalf("expected slice response model validation error, got %v", err)
	}
}

func TestOperationCoverageForHandlerWrapperEdges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	op := newOperation(http.MethodGet, "/tx", func(ctx *Context, input *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	}, nil)
	op.behavior.withTransaction = true
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/tx", nil)
	op.route.ginHandler(ctx)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected missing transaction handlers to fail, got %d", recorder.Code)
	}

	voidBindError := newVoidOperation(http.MethodGet, "/void", func(ctx *Context, input *struct {
		Page int `query:"page" binding:"omitempty,min=1"`
	}) error {
		return nil
	}, nil)
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/void?page=bad", nil)
	voidBindError.route.ginHandler(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected void bind error, got %d", recorder.Code)
	}

	voidTx := newVoidOperation(http.MethodGet, "/void-tx", func(ctx *Context, input *struct{}) error {
		return nil
	}, nil)
	voidTx.behavior.withTransaction = true
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/void-tx", nil)
	voidTx.route.ginHandler(ctx)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected void missing transaction handlers to fail, got %d", recorder.Code)
	}

	voidHandlerError := newVoidOperation(http.MethodGet, "/void-error", func(ctx *Context, input *struct{}) error {
		return errors.New("void failed")
	}, nil)
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/void-error", nil)
	voidHandlerError.route.ginHandler(ctx)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected void handler error, got %d", recorder.Code)
	}

	empty := &operation{}
	empty.finalize()
	wrapped := &operation{
		route: operationRoute{
			outputType: reflect.TypeOf(Download{}),
			ginHandler: func(*gin.Context) {},
		},
		behavior: operationBehavior{timeout: time.Millisecond},
	}
	wrapped.finalize()
	if wrapped.route.ginHandler == nil {
		t.Fatal("expected finalize to keep a wrapped handler")
	}
}
