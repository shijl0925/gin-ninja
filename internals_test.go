package ninja

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
)

type BindEmbeddedInput struct {
	Trace string `header:"X-Trace"`
}

type bindComplexInput struct {
	BindEmbeddedInput
	ID      int     `path:"id"`
	Page    int     `form:"page"`
	Active  bool    `form:"active"`
	Score   float64 `header:"X-Score"`
	Session string  `cookie:"session"`
	Name    string  `json:"name" binding:"required"`
	Age     int     `json:"age"`
}

type contextClaims struct {
	userID uint
}

func (c contextClaims) GetUserID() uint { return c.userID }

type SchemaEmbedded struct {
	Embedded string `json:"embedded" binding:"required"`
}

type schemaSample struct {
	SchemaEmbedded
	Name  string            `json:"name" binding:"required" description:"display name" example:"alice"`
	Count int               `json:"count"`
	Tags  []string          `json:"tags"`
	Meta  map[string]string `json:"meta"`
	Skip  string            `json:"-"`
}

type schemaModel struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type schemaRootTime struct {
	ModelSchema[time.Time]
}

type pointerMarshaler string

func (p *pointerMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal("wrapped:" + string(*p))
}

type pointerMarshalerModel struct {
	Value pointerMarshaler `json:"value"`
}

type publicSchema struct {
	ModelSchema[schemaModel] `fields:"id,name,email" exclude:"password"`
}

type multipartBindInput struct {
	Title string          `form:"title" binding:"required"`
	File  *UploadedFile   `file:"file" binding:"required"`
	Files []*UploadedFile `file:"files"`
}

type bindEdgeQueryInput struct {
	Search string   `form:"search"`
	Tags   []string `form:"tag"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext(method, target string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func TestBindInput_Success(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/users/42?page=3&active=true", `{"name":"alice","age":30}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request.Header.Set("X-Trace", "trace-1")
	c.Request.Header.Set("X-Score", "9.5")
	c.Request.AddCookie(&http.Cookie{Name: "session", Value: "sess-1"})

	var in bindComplexInput
	if err := bindInput(c, http.MethodPost, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}

	if in.ID != 42 || in.Page != 3 || !in.Active || in.Name != "alice" || in.Age != 30 {
		t.Fatalf("unexpected bound input: %+v", in)
	}
	if in.Score != 9.5 {
		t.Fatalf("expected special fields to bind, got %+v", in)
	}
	if in.Session != "sess-1" {
		t.Fatalf("expected cookie field to bind, got %+v", in)
	}
}

func TestBindInput_Errors(t *testing.T) {
	t.Run("non-struct", func(t *testing.T) {
		c, _ := newTestContext(http.MethodGet, "/", "")
		var n int
		if err := bindInput(c, http.MethodGet, &n); err == nil {
			t.Fatal("expected error for non-struct input")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		c, _ := newTestContext(http.MethodPost, "/users/42", `{"name":`)
		c.Params = gin.Params{{Key: "id", Value: "42"}}
		var in bindComplexInput
		err := bindInput(c, http.MethodPost, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if apiErr.Status != http.StatusBadRequest || apiErr.Code != "INVALID_JSON" {
			t.Fatalf("unexpected api error: %+v", apiErr)
		}
	})

	t.Run("validation", func(t *testing.T) {
		c, _ := newTestContext(http.MethodPost, "/users/42", `{}`)
		c.Params = gin.Params{{Key: "id", Value: "42"}}
		var in bindComplexInput
		err := bindInput(c, http.MethodPost, &in)
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T", err)
		}
		if len(validationErr.Errors) != 1 || validationErr.Errors[0].Field != "name" {
			t.Fatalf("unexpected validation errors: %+v", validationErr.Errors)
		}
	})

	t.Run("bad path/header/cookie", func(t *testing.T) {
		c, _ := newTestContext(http.MethodPost, "/users/nope", `{"name":"alice"}`)
		c.Params = gin.Params{{Key: "id", Value: "bad"}}
		var in bindComplexInput
		err := bindInput(c, http.MethodPost, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_PATH_PARAM" {
			t.Fatalf("expected BAD_PATH_PARAM, got %v", err)
		}

		c, _ = newTestContext(http.MethodPost, "/users/1", `{"name":"alice"}`)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request.Header.Set("X-Score", "bad")
		err = bindInput(c, http.MethodPost, &in)
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_HEADER" {
			t.Fatalf("expected BAD_HEADER, got %v", err)
		}

		c, _ = newTestContext(http.MethodPost, "/users/1", `{"name":"alice"}`)
		type cookieInput struct {
			Session int    `cookie:"session"`
			Name    string `json:"name" binding:"required"`
		}
		var cookieIn cookieInput
		c.Request.AddCookie(&http.Cookie{Name: "session", Value: "bad"})
		err = bindInput(c, http.MethodPost, &cookieIn)
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_COOKIE" {
			t.Fatalf("expected BAD_COOKIE, got %v", err)
		}
	})

	t.Run("bad query conversion", func(t *testing.T) {
		c, _ := newTestContext(http.MethodGet, "/users/42?page=nope", "")
		c.Params = gin.Params{{Key: "id", Value: "42"}}
		var in bindComplexInput
		err := bindInput(c, http.MethodGet, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Code != "INVALID_QUERY" || apiErr.Status != http.StatusBadRequest {
			t.Fatalf("expected INVALID_QUERY bad request, got %v", err)
		}
	})

	t.Run("missing multipart file", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("title", "demo"); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPost, "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req

		var in multipartBindInput
		err := bindInput(c, http.MethodPost, &in)
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T", err)
		}
	})
}

func TestBindInput_MultipartSuccess(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "demo"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	for _, field := range []struct {
		name string
		file string
	}{
		{name: "file", file: "single.txt"},
		{name: "files", file: "a.txt"},
		{name: "files", file: "b.txt"},
	} {
		part, err := writer.CreateFormFile(field.name, field.file)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write([]byte("content:" + field.file)); err != nil {
			t.Fatalf("part.Write: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req

	var in multipartBindInput
	if err := bindInput(c, http.MethodPost, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.Title != "demo" || in.File == nil || in.File.Filename != "single.txt" || len(in.Files) != 2 {
		t.Fatalf("unexpected multipart input: %+v", in)
	}
}

func TestBindInput_QueryBoundaryValues(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/search?search=a%2Bb+%E4%B8%AD%E6%96%87&tag=first&tag=second", "")

	var in bindEdgeQueryInput
	if err := bindInput(c, http.MethodGet, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.Search != "a+b 中文" {
		t.Fatalf("expected decoded query string, got %q", in.Search)
	}
	if len(in.Tags) != 2 || in.Tags[0] != "first" || in.Tags[1] != "second" {
		t.Fatalf("expected repeated query values, got %+v", in.Tags)
	}
}

func TestSetFieldFromString(t *testing.T) {
	var s string
	var b bool
	var i int64
	var u uint
	var f float64
	var unsupported struct{}

	cases := []struct {
		value reflect.Value
		raw   string
	}{
		{reflect.ValueOf(&s).Elem(), "hello"},
		{reflect.ValueOf(&b).Elem(), "true"},
		{reflect.ValueOf(&i).Elem(), "12"},
		{reflect.ValueOf(&u).Elem(), "13"},
		{reflect.ValueOf(&f).Elem(), "3.14"},
	}
	for _, tc := range cases {
		if err := setFieldFromString(tc.value, tc.raw); err != nil {
			t.Fatalf("setFieldFromString(%q): %v", tc.raw, err)
		}
	}

	if s != "hello" || !b || i != 12 || u != 13 || f != 3.14 {
		t.Fatalf("unexpected converted values: %q %v %d %d %v", s, b, i, u, f)
	}

	if err := setFieldFromString(reflect.ValueOf(&unsupported).Elem(), "x"); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}

func TestBindSpecialFields_AnonymousStruct(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/", "")
	c.Request.Header.Set("X-Trace", "trace-1")

	var in bindComplexInput
	v := reflect.ValueOf(&in).Elem()
	if err := bindSpecialFields(c, v.Type(), v); err != nil {
		t.Fatalf("bindSpecialFields: %v", err)
	}
	if in.Trace != "trace-1" {
		t.Fatalf("expected anonymous embedded header binding, got %+v", in)
	}
}

func TestContextHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	t.Cleanup(cancel)
	reqCtx = context.WithValue(reqCtx, "request-key", "request-value")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)
	c.Set("gin-key", "gin-value")
	c.Set(requestIDContextKey, "req-123")
	c.Set(contextkeys.JWTClaims, contextClaims{userID: 7})

	ctx := newContext(c)
	if ctx.Value("gin-key") != "gin-value" || ctx.Value("request-key") != "request-value" {
		t.Fatal("expected context values from gin and request context")
	}
	expectedDeadline, expectedOK := reqCtx.Deadline()
	deadline, ok := ctx.Deadline()
	if ok != expectedOK || !deadline.Equal(expectedDeadline) {
		t.Fatalf("expected deadline %v (ok=%v), got %v (ok=%v)", expectedDeadline, expectedOK, deadline, ok)
	}
	if ctx.StdContext().Value("request-key") != "request-value" {
		t.Fatal("expected StdContext passthrough")
	}
	if ctx.RequestID() != "req-123" || ctx.GetUserID() != 7 {
		t.Fatalf("unexpected helper values: requestID=%q userID=%d", ctx.RequestID(), ctx.GetUserID())
	}
	if ctx.Err() != nil {
		t.Fatalf("expected nil err, got %v", ctx.Err())
	}
	select {
	case <-ctx.Done():
		t.Fatal("expected active context")
	default:
	}
}

func TestContextResponseHelpers(t *testing.T) {
	t.Run("json helpers", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		ctx := newContext(c)
		ctx.JSON200(map[string]string{"status": "ok"})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		c, w = newTestContext(http.MethodGet, "/", "")
		ctx = newContext(c)
		ctx.JSON201(map[string]string{"status": "created"})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}

		c, w = newTestContext(http.MethodGet, "/", "")
		ctx = newContext(c)
		ctx.JSON204()
		if ctx.Writer.Status() != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", ctx.Writer.Status())
		}
	})

	t.Run("auth helpers", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		ctx := newContext(c)
		ctx.Forbidden("nope")
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "FORBIDDEN") {
			t.Fatalf("unexpected forbidden response: %d %s", w.Code, w.Body.String())
		}

		c, w = newTestContext(http.MethodGet, "/", "")
		ctx = newContext(c)
		ctx.Unauthorized("bad token")
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "UNAUTHORIZED") {
			t.Fatalf("unexpected unauthorized response: %d %s", w.Code, w.Body.String())
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("api error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		writeError(c, &Error{Status: http.StatusTeapot, Code: "TEAPOT", Message: "short and stout"})
		if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "TEAPOT") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("validation error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		writeError(c, &ValidationError{Errors: []FieldError{{Field: "name", Message: "field is required"}}})
		if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "VALIDATION_ERROR") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("generic error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		writeError(c, errors.New("boom"))
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "boom") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("instance mapper", func(t *testing.T) {
		sentinel := errors.New("mapped")
		api := New(Config{})
		api.RegisterErrorMapper(func(err error) error {
			if errors.Is(err, sentinel) {
				return &Error{Status: http.StatusTeapot, Code: "MAPPED", Message: "mapped"}
			}
			return nil
		})

		c, w := newTestContext(http.MethodGet, "/", "")
		c.Set(ninjaAPIContextKey, api)
		writeError(c, sentinel)
		if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "MAPPED") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("global custom mapper fallback", func(t *testing.T) {
		sentinel := errors.New("mapped")

		errorMappersMu.Lock()
		original := append([]ErrorMapper(nil), errorMappers...)
		errorMappers = append(errorMappers, func(err error) error {
			if errors.Is(err, sentinel) {
				return &Error{Status: http.StatusTeapot, Code: "MAPPED", Message: "mapped"}
			}
			return nil
		})
		errorMappersMu.Unlock()
		defer func() {
			errorMappersMu.Lock()
			errorMappers = original
			errorMappersMu.Unlock()
		}()

		c, w := newTestContext(http.MethodGet, "/", "")
		writeError(c, sentinel)
		if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "MAPPED") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})
}

func TestSchemaAndHelperFunctions(t *testing.T) {
	registry := newSchemaRegistry()
	schema := registry.schemaForType(reflect.TypeOf(schemaSample{}))
	if schema.Ref == "" {
		t.Fatalf("expected component ref, got %+v", schema)
	}

	name := typeName(reflect.TypeOf(schemaSample{}))
	component := registry.schemas[name]
	if component == nil {
		t.Fatalf("expected registered component %q", name)
	}
	if component.Type != "object" || component.Properties["name"].Description != "display name" {
		t.Fatalf("unexpected component schema: %+v", component)
	}
	if component.Properties["name"].Example != "alice" {
		t.Fatalf("expected example annotation, got %+v", component.Properties["name"])
	}
	if component.Properties["tags"].Type != "array" || component.Properties["meta"].Type != "object" {
		t.Fatalf("unexpected array/map schemas: %+v", component.Properties)
	}
	if _, ok := component.Properties["embedded"]; !ok {
		t.Fatalf("expected embedded field to be flattened: %+v", component.Properties)
	}
	if _, ok := component.Properties["skip"]; ok {
		t.Fatalf("expected skipped field to be omitted: %+v", component.Properties)
	}

	if got := ginPathToOpenAPI("/users/:id/posts/:postID"); got != "/users/{id}/posts/{postID}" {
		t.Fatalf("unexpected openapi path: %s", got)
	}
	if got := sanitizeComponentName("***"); got != "Schema" {
		t.Fatalf("expected Schema fallback, got %q", got)
	}
	if got := jsonFieldName(reflect.TypeOf(schemaSample{}).Field(1)); got != "name" {
		t.Fatalf("expected json field name, got %q", got)
	}
	if got := defaultJSONFieldName("ID"); got != "id" {
		t.Fatalf("expected acronym field name id, got %q", got)
	}
	if got := defaultJSONFieldName("URLValue"); got != "urlValue" {
		t.Fatalf("expected acronym camel field name urlValue, got %q", got)
	}
	if fileSchema := registry.schemaForType(reflect.TypeOf(UploadedFile{})); fileSchema.Format != "binary" {
		t.Fatalf("expected binary schema for uploads, got %+v", fileSchema)
	}
	if !isRequired(reflect.TypeOf(schemaSample{}).Field(1)) {
		t.Fatal("expected required field")
	}
	if got := deref(reflect.TypeOf(&schemaSample{})); got.Kind() != reflect.Struct {
		t.Fatalf("expected dereferenced struct, got %s", got.Kind())
	}
	if got := intFormat(reflect.Int32); got != "int32" {
		t.Fatalf("expected int32 format, got %q", got)
	}
	if got := intFormat(reflect.Int64); got != "int64" {
		t.Fatalf("expected int64 format, got %q", got)
	}
	if got := uintFormat(reflect.Uint32); got != "uint32" {
		t.Fatalf("expected uint32 format, got %q", got)
	}
	if got := uintFormat(reflect.Uint64); got != "uint64" {
		t.Fatalf("expected uint64 format, got %q", got)
	}

	modelSchemaRef := registry.schemaForType(reflect.TypeOf(publicSchema{}))
	if modelSchemaRef.Ref == "" {
		t.Fatalf("expected model schema ref, got %+v", modelSchemaRef)
	}
	publicComponent := registry.schemas[typeName(reflect.TypeOf(publicSchema{}))]
	if publicComponent == nil {
		t.Fatalf("expected public schema component to be registered")
	}
	if _, ok := publicComponent.Properties["password"]; ok {
		t.Fatalf("expected excluded field to be omitted, got %+v", publicComponent.Properties)
	}
	if _, ok := publicComponent.Properties["id"]; !ok {
		t.Fatalf("expected id field to remain, got %+v", publicComponent.Properties)
	}
	if _, ok := publicComponent.Properties["email"]; !ok {
		t.Fatalf("expected whitelisted field to remain, got %+v", publicComponent.Properties)
	}
}

func TestModelSchemaSerializationAndBinding(t *testing.T) {
	payload, err := json.Marshal(NewModelSchema(schemaModel{
		ID:       1,
		Name:     "alice",
		Email:    "alice@example.com",
		Password: "secret",
	}, Fields("id", "name", "email"), Exclude("password")))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := data["password"]; ok {
		t.Fatalf("expected password to be excluded, got %v", data)
	}
	if data["id"] != float64(1) {
		t.Fatalf("expected id to remain, got %v", data)
	}
	if data["email"] != "alice@example.com" {
		t.Fatalf("expected email to remain, got %v", data)
	}

	typed, err := BindModelSchema[publicSchema](schemaModel{
		ID:       2,
		Name:     "bob",
		Email:    "bob@example.com",
		Password: "hidden",
	})
	if err != nil {
		t.Fatalf("BindModelSchema: %v", err)
	}
	if got := typed.Fields; len(got) != 3 || got[0] != "email" || got[1] != "id" || got[2] != "name" {
		t.Fatalf("expected fields from tags, got %v", got)
	}
	if got := typed.Exclude; len(got) != 1 || got[0] != "password" {
		t.Fatalf("expected exclude from tags, got %v", got)
	}
	if typed.Model.Password != "hidden" {
		t.Fatalf("expected model to be assigned, got %+v", typed.Model)
	}

	timePayload, err := json.Marshal(NewModelSchema(time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("MarshalJSON time root: %v", err)
	}
	if string(timePayload) != `"2026-04-06T09:00:00Z"` {
		t.Fatalf("expected time marshaler output, got %s", string(timePayload))
	}

	pointerPayload, err := json.Marshal(NewModelSchema(pointerMarshalerModel{Value: pointerMarshaler("demo")}))
	if err != nil {
		t.Fatalf("MarshalJSON pointer marshaler field: %v", err)
	}
	if string(pointerPayload) != `{"value":"wrapped:demo"}` {
		t.Fatalf("expected pointer marshaler output, got %s", string(pointerPayload))
	}
}

func TestExtractParams_EmbeddedBodyFields(t *testing.T) {
	type EmbeddedBody struct {
		Name string `json:"name" binding:"required"`
	}
	type createInput struct {
		EmbeddedBody
		Age int `json:"age"`
	}

	spec := newOpenAPISpec(Config{})
	params, bodySchema, contentType := spec.extractParams(http.MethodPost, reflect.TypeOf(createInput{}))
	if len(params) != 0 {
		t.Fatalf("expected no parameters, got %+v", params)
	}
	if bodySchema == nil {
		t.Fatal("expected request body schema")
	}
	if _, ok := bodySchema.Properties["name"]; !ok {
		t.Fatalf("expected embedded body field to be preserved, got %+v", bodySchema.Properties)
	}
	if _, ok := bodySchema.Properties["age"]; !ok {
		t.Fatalf("expected direct body field to be preserved, got %+v", bodySchema.Properties)
	}
	if len(bodySchema.Required) != 1 || bodySchema.Required[0] != "name" {
		t.Fatalf("expected embedded required fields to be preserved, got %+v", bodySchema.Required)
	}
	if contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}
}

func TestExtractParams_MultipartBodyFields(t *testing.T) {
	spec := newOpenAPISpec(Config{})
	params, bodySchema, contentType := spec.extractParams(http.MethodPost, reflect.TypeOf(multipartBindInput{}))
	if len(params) != 0 {
		t.Fatalf("expected no parameters, got %+v", params)
	}
	if bodySchema == nil {
		t.Fatal("expected multipart body schema")
	}
	if _, ok := bodySchema.Properties["title"]; !ok {
		t.Fatalf("expected form field in multipart body, got %+v", bodySchema.Properties)
	}
	if prop, ok := bodySchema.Properties["file"]; !ok || prop.Format != "binary" {
		t.Fatalf("expected binary file field, got %+v", bodySchema.Properties["file"])
	}
	if prop, ok := bodySchema.Properties["files"]; !ok || prop.Type != "array" || prop.Items == nil || prop.Items.Format != "binary" {
		t.Fatalf("expected file array field, got %+v", bodySchema.Properties["files"])
	}
	if contentType != "multipart/form-data" {
		t.Fatalf("expected multipart content type, got %q", contentType)
	}
}

func TestSecurityAndErrorHelpers(t *testing.T) {
	requirements := []SecurityRequirement{{"bearerAuth": {}}, {"oauth2": {"read"}}}
	clonedRequirements := cloneSecurityRequirements(requirements)
	requirements[1]["oauth2"][0] = "write"
	if clonedRequirements[1]["oauth2"][0] != "read" {
		t.Fatalf("expected security requirements clone to be independent: %+v", clonedRequirements)
	}

	schemes := map[string]SecurityScheme{"bearerAuth": HTTPBearerSecurityScheme("JWT")}
	clonedSchemes := cloneSecuritySchemes(schemes)
	scheme := clonedSchemes["bearerAuth"]
	scheme.BearerFormat = "opaque"
	clonedSchemes["bearerAuth"] = scheme
	if schemes["bearerAuth"].BearerFormat != "JWT" {
		t.Fatalf("expected security schemes clone to be independent: %+v", schemes)
	}

	if err := NewError(http.StatusBadRequest, "bad"); err.Status != http.StatusBadRequest || err.Message != "bad" {
		t.Fatalf("unexpected NewError result: %+v", err)
	}
	if err := NewErrorWithCode(http.StatusConflict, "CONFLICT", "duplicate"); err.Code != "CONFLICT" {
		t.Fatalf("unexpected NewErrorWithCode result: %+v", err)
	}
	if ForbiddenError().Error() == "" || (&ValidationError{Errors: []FieldError{{Field: "x", Message: "y"}}}).Error() == "" {
		t.Fatal("expected error strings to be non-empty")
	}
}

func TestOptionHelpers(t *testing.T) {
	router := NewRouter("/users", WithTags("Users", "Admin"), WithSecurity("oauth2", "read"), WithBearerAuth(), WithVersion("v1"))
	WithTagDescription("Users", "user operations")(router)
	if len(router.tags) != 2 || len(router.security) != 2 || router.version != "v1" {
		t.Fatalf("unexpected router options: %+v", router)
	}
	if router.tagDescriptions["Users"] != "user operations" {
		t.Fatalf("expected router tag description to be recorded, got %+v", router.tagDescriptions)
	}

	op := &operation{}
	Summary("list users")(op)
	Description("full description")(op)
	OperationID("listUsers")(op)
	Tags("Users")(op)
	TagDescription("Users", "user operations")(op)
	Security("oauth2", "read")(op)
	BearerAuth()(op)
	Deprecated()(op)
	Cache(time.Minute)(op)
	CacheControl("private, max-age=60")(op)
	ETag()(op)
	ExcludeFromDocs()(op)
	SuccessStatus(http.StatusAccepted)(op)
	Response(http.StatusBadRequest, "bad request", schemaSample{})(op)
	Response(http.StatusNotFound, "not found", nil)(op)
	Paginated[schemaSample]()(op)
	PaginatedResponse[schemaSample](http.StatusPartialContent, "partial")(op)
	Timeout(time.Second)(op)
	RateLimit(2, 3)(op)
	UseMiddleware(func(ctx *Context) error { return nil })(op)
	MiddlewareChain("auth")(op)
	Intercept(func(ctx *Context, next NextHandler) (any, error) { return next(ctx) })(op)
	TransformRequest(func(ctx *Context, input any) error { return nil })(op)
	TransformResponse(func(ctx *Context, output any) (any, error) { return output, nil })(op)

	if op.summary != "list users" || op.description != "full description" || op.operationID != "listUsers" {
		t.Fatalf("unexpected operation metadata: %+v", op)
	}
	if !op.deprecated || !op.excludeFromDocs || op.successStatus != http.StatusAccepted || len(op.security) != 2 {
		t.Fatalf("unexpected operation options: %+v", op)
	}
	if op.tagDescriptions["Users"] != "user operations" || op.paginatedItemType == nil || op.timeout != time.Second || op.rateLimit == nil || op.cache == nil || !op.etagEnabled {
		t.Fatalf("unexpected extended operation options: %+v", op)
	}
	if len(op.responses) != 3 || op.responses[0].responseType == nil || op.responses[1].responseType != nil || op.responses[2].paginatedItemType == nil {
		t.Fatalf("unexpected documented responses: %+v", op.responses)
	}
	if len(op.middleware) != 1 || len(op.middlewareChainNames) != 1 || len(op.interceptors) != 1 || len(op.requestTransformers) != 1 || len(op.responseTransformers) != 1 {
		t.Fatalf("unexpected pipeline options: %+v", op)
	}
}

func TestNewOperationNilOutputAndVoidOperation(t *testing.T) {
	op := newOperation(http.MethodGet, "/", func(ctx *Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	}, nil)

	c, _ := newTestContext(http.MethodGet, "/", "")
	op.ginHandler(c)
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204 for nil output, got %d", c.Writer.Status())
	}

	voidOp := newVoidOperation(http.MethodDelete, "/:id", func(ctx *Context, input *struct{}) error {
		return nil
	}, nil)
	c, _ = newTestContext(http.MethodDelete, "/1", "")
	voidOp.ginHandler(c)
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204 for void operation, got %d", c.Writer.Status())
	}
}
