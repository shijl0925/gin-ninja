package ninja

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type BindEmbeddedInput struct {
	Trace string `header:"X-Trace"`
}

type bindComplexInput struct {
	BindEmbeddedInput
	ID     int     `path:"id"`
	Page   int     `form:"page"`
	Active bool    `form:"active"`
	Score  float64 `header:"X-Score"`
	Name   string  `json:"name" binding:"required"`
	Age    int     `json:"age"`
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

	t.Run("bad path/header", func(t *testing.T) {
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
	})
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
	c.Set(jwtClaimsKey, contextClaims{userID: 7})

	ctx := newContext(c)
	if ctx.Value("gin-key") != "gin-value" || ctx.Value("request-key") != "request-value" {
		t.Fatal("expected context values from gin and request context")
	}
	if deadline, ok := ctx.Deadline(); !ok || deadline == nil {
		t.Fatal("expected deadline from request context")
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
	if ErrForbidden.Error() == "" || (&ValidationError{Errors: []FieldError{{Field: "x", Message: "y"}}}).Error() == "" {
		t.Fatal("expected error strings to be non-empty")
	}
}

func TestOptionHelpers(t *testing.T) {
	router := NewRouter("/users", WithTags("Users", "Admin"), WithSecurity("oauth2", "read"), WithBearerAuth())
	if len(router.tags) != 2 || len(router.security) != 2 {
		t.Fatalf("unexpected router options: %+v", router)
	}

	op := &operation{}
	Summary("list users")(op)
	Description("full description")(op)
	OperationID("listUsers")(op)
	Tags("Users")(op)
	Security("oauth2", "read")(op)
	BearerAuth()(op)
	Deprecated()(op)
	SuccessStatus(http.StatusAccepted)(op)

	if op.summary != "list users" || op.description != "full description" || op.operationID != "listUsers" {
		t.Fatalf("unexpected operation metadata: %+v", op)
	}
	if !op.deprecated || op.successStatus != http.StatusAccepted || len(op.security) != 2 {
		t.Fatalf("unexpected operation options: %+v", op)
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
