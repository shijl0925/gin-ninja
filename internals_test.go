package ninja

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pagination"
)

type BindEmbeddedInput struct {
	Trace string `header:"X-Trace"`
}

type bindComplexInput struct {
	BindEmbeddedInput
	ID      int     `path:"id"`
	Page    int     `query:"page"`
	Active  bool    `query:"active"`
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

type requiredSchemaModel struct {
	ID    uint   `json:"id"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email"`
}

type schemaModeModel struct {
	ID        uint        `json:"id" gorm:"primaryKey"`
	Name      string      `json:"name"`
	Password  string      `json:"password" ninja:"write_only"`
	Invite    string      `json:"invite_code" ninja:"write_only,create_only"`
	Status    string      `json:"status_note" ninja:"update_only"`
	CreatedAt time.Time   `json:"created_at"`
	Profile   schemaModel `json:"profile"`
	Tags      []string    `json:"tags"`
	Computed  string      `json:"computed" gorm:"-"`
}

type schemaRelationModel struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Secret string `json:"secret" ninja:"write_only"`
}

type schemaDepthModel struct {
	ID       uint                  `json:"id"`
	Name     string                `json:"name"`
	Owner    schemaRelationModel   `json:"owner"`
	Members  []schemaRelationModel `json:"members"`
	Ignored  schemaRelationModel   `json:"ignored" gorm:"-"`
	Internal string                `json:"internal" ninja:"write_only"`
}

type schemaDeepRelationModel struct {
	ID    uint                `json:"id"`
	Owner schemaRelationModel `json:"owner"`
}

type schemaDeepModel struct {
	ID     uint                    `json:"id"`
	Parent schemaDeepRelationModel `json:"parent"`
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

type multipartComplexBindInput struct {
	Title    string                  `form:"title" binding:"required"`
	Counts   []int                   `form:"count"`
	Primary  *multipart.FileHeader   `file:"primary" binding:"required"`
	RawFiles []*multipart.FileHeader `file:"raw_files"`
	Uploads  []*UploadedFile         `file:"uploads"`
}

type bindEdgeQueryInput struct {
	Search string   `query:"search"`
	Tags   []string `query:"tag"`
}

type testContextKey string

type bindOverrideInput struct {
	ID      int    `path:"id" json:"id"`
	Page    int    `query:"page" json:"page"`
	Trace   string `header:"X-Trace" json:"trace"`
	Session string `cookie:"session" json:"session"`
	Name    string `json:"name"`
}

type customTextValue string

func (v *customTextValue) UnmarshalText(text []byte) error {
	*v = customTextValue("custom:" + string(text))
	return nil
}

type formURLEncodedInput struct {
	Name string          `form:"name" binding:"required"`
	Tags []string        `form:"tag"`
	When time.Time       `form:"when"`
	IP   net.IP          `form:"ip"`
	Mode customTextValue `form:"mode"`
}

type embeddedPointerBindInput struct {
	*BindEmbeddedInput
	Page int `query:"page" default:"1"`
}

type nestedJSONAddress struct {
	City string `json:"city" binding:"required"`
	Zip  int    `json:"zip"`
}

type nestedJSONProfile struct {
	Name    string            `json:"name" binding:"required"`
	Address nestedJSONAddress `json:"address" binding:"required"`
	Tags    []string          `json:"tags"`
}

type nestedJSONBindInput struct {
	ID      int               `path:"id" json:"id"`
	Page    int               `query:"page" json:"page"`
	Trace   string            `header:"X-Trace" json:"trace"`
	Profile nestedJSONProfile `json:"profile" binding:"required"`
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

func TestBindInput_JSONDoesNotOverrideNonBodyFields(t *testing.T) {
	c, _ := newTestContext(http.MethodPut, "/users/42?page=3", `{"id":99,"page":99,"trace":"body","session":"body","name":"alice"}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request.Header.Set("X-Trace", "trace-1")
	c.Request.AddCookie(&http.Cookie{Name: "session", Value: "sess-1"})

	var in bindOverrideInput
	if err := bindInput(c, http.MethodPut, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.ID != 42 || in.Page != 3 || in.Trace != "trace-1" || in.Session != "sess-1" || in.Name != "alice" {
		t.Fatalf("unexpected bound input: %+v", in)
	}
}

func TestBindInput_FormURLEncodedAndCommonTypes(t *testing.T) {
	body := "name=alice&tag=go&tag=api&when=2026-04-26&ip=127.0.0.1&mode=fast"
	c, _ := newTestContext(http.MethodPost, "/submit", body)
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var in formURLEncodedInput
	if err := bindInput(c, http.MethodPost, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.Name != "alice" || !reflect.DeepEqual(in.Tags, []string{"go", "api"}) {
		t.Fatalf("unexpected form values: %+v", in)
	}
	if in.When.Format("2006-01-02") != "2026-04-26" {
		t.Fatalf("expected date to bind, got %s", in.When.Format(time.RFC3339))
	}
	if !in.IP.Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("expected IP to bind, got %v", in.IP)
	}
	if in.Mode != "custom:fast" {
		t.Fatalf("expected custom text value to bind, got %q", in.Mode)
	}
}

func TestBindInput_EmbeddedPointerStruct(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/items?page=3", "")
	c.Request.Header.Set("X-Trace", "trace-1")

	var in embeddedPointerBindInput
	if err := bindInput(c, http.MethodGet, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.BindEmbeddedInput == nil {
		t.Fatal("expected embedded pointer to be allocated")
	}
	if in.Trace != "trace-1" || in.Page != 3 {
		t.Fatalf("unexpected embedded pointer bind: %+v", in)
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
		if apiErr.Code != http.StatusBadRequest {
			t.Fatalf("unexpected api error: %+v", apiErr)
		}
	})

	t.Run("json body too large", func(t *testing.T) {
		const testBodySizeLimit int64 = 5
		oversizedBody := `{"name":"alice"}`
		c, _ := newTestContext(http.MethodPost, "/users/42", oversizedBody)
		c.Params = gin.Params{{Key: "id", Value: "42"}}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, testBodySizeLimit)
		var in bindComplexInput
		err := bindInput(c, http.MethodPost, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if apiErr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("unexpected api error: %+v", apiErr)
		}
	})

	t.Run("json body exceeds size limit after complete prefix", func(t *testing.T) {
		prefix := `{"name":"alice"}`
		c, _ := newTestContext(http.MethodPost, "/users/42", prefix+"x")
		_, err := readJSONBody(c, int64(len(prefix)))
		var apiErr *Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if apiErr.Code != http.StatusRequestEntityTooLarge {
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
		if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
			t.Fatalf("expected BAD_PATH_PARAM, got %v", err)
		}

		c, _ = newTestContext(http.MethodPost, "/users/1", `{"name":"alice"}`)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request.Header.Set("X-Score", "bad")
		err = bindInput(c, http.MethodPost, &in)
		if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
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
		if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
			t.Fatalf("expected BAD_COOKIE, got %v", err)
		}
	})

	t.Run("bad query conversion", func(t *testing.T) {
		c, _ := newTestContext(http.MethodGet, "/users/42?page=nope", "")
		c.Params = gin.Params{{Key: "id", Value: "42"}}
		var in bindComplexInput
		err := bindInput(c, http.MethodGet, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
			t.Fatalf("expected INVALID_QUERY bad request, got %v", err)
		}
	})

	t.Run("bad form body conversion", func(t *testing.T) {
		c, _ := newTestContext(http.MethodPost, "/users", "age=nope")
		c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		type formInput struct {
			Age int `form:"age"`
		}
		var in formInput
		err := bindInput(c, http.MethodPost, &in)
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
			t.Fatalf("expected INVALID_FORM bad request, got %v", err)
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

func TestBindInput_NestedJSONPreservesNonBodyFields(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/users/42?page=3", `{
		"id": 99,
		"page": 99,
		"trace": "body-trace",
		"profile": {
			"name": "alice",
			"address": {"city": "Shanghai", "zip": 200000},
			"tags": ["go", "api"]
		}
	}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request.Header.Set("X-Trace", "header-trace")

	var in nestedJSONBindInput
	if err := bindInput(c, http.MethodPost, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.ID != 42 || in.Page != 3 || in.Trace != "header-trace" {
		t.Fatalf("expected non-body fields to override JSON values, got %+v", in)
	}
	if in.Profile.Name != "alice" || in.Profile.Address.City != "Shanghai" || in.Profile.Address.Zip != 200000 {
		t.Fatalf("unexpected nested profile: %+v", in.Profile)
	}
	if !reflect.DeepEqual(in.Profile.Tags, []string{"go", "api"}) {
		t.Fatalf("unexpected nested tags: %+v", in.Profile.Tags)
	}
}

func TestBindInput_NestedJSONValidationError(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/users/42", `{
		"profile": {
			"name": "alice",
			"address": {"zip": 200000}
		}
	}`)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	var in nestedJSONBindInput
	err := bindInput(c, http.MethodPost, &in)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(validationErr.Errors) != 1 || validationErr.Errors[0].Field != "city" {
		t.Fatalf("unexpected validation errors: %+v", validationErr.Errors)
	}
}

func TestBindInput_MultipartComplexMultiFileUpload(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "demo"); err != nil {
		t.Fatalf("WriteField title: %v", err)
	}
	for _, count := range []string{"1", "2", "3"} {
		if err := writer.WriteField("count", count); err != nil {
			t.Fatalf("WriteField count: %v", err)
		}
	}
	for _, field := range []struct {
		name string
		file string
		body string
	}{
		{name: "primary", file: "primary.txt", body: "primary"},
		{name: "raw_files", file: "raw-a.txt", body: "raw-a"},
		{name: "raw_files", file: "raw-b.txt", body: "raw-b"},
		{name: "uploads", file: "upload-a.txt", body: "upload-a"},
		{name: "uploads", file: "upload-b.txt", body: "upload-b"},
	} {
		part, err := writer.CreateFormFile(field.name, field.file)
		if err != nil {
			t.Fatalf("CreateFormFile %s/%s: %v", field.name, field.file, err)
		}
		if _, err := part.Write([]byte(field.body)); err != nil {
			t.Fatalf("part.Write %s/%s: %v", field.name, field.file, err)
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

	var in multipartComplexBindInput
	if err := bindInput(c, http.MethodPost, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
	}
	if in.Title != "demo" || !reflect.DeepEqual(in.Counts, []int{1, 2, 3}) {
		t.Fatalf("unexpected multipart form values: %+v", in)
	}
	if in.Primary == nil || in.Primary.Filename != "primary.txt" {
		t.Fatalf("unexpected primary file: %+v", in.Primary)
	}
	if len(in.RawFiles) != 2 || in.RawFiles[0].Filename != "raw-a.txt" || in.RawFiles[1].Filename != "raw-b.txt" {
		t.Fatalf("unexpected raw files: %+v", in.RawFiles)
	}
	if len(in.Uploads) != 2 || in.Uploads[0].Filename != "upload-a.txt" || in.Uploads[1].Filename != "upload-b.txt" {
		t.Fatalf("unexpected uploaded files: %+v", in.Uploads)
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

	// Overflow checks: values outside the target type's range must return an error.
	var i8 int8
	if err := setFieldFromString(reflect.ValueOf(&i8).Elem(), "128"); err == nil {
		t.Fatal("expected overflow error for int8 value 128")
	}
	var u8 uint8
	if err := setFieldFromString(reflect.ValueOf(&u8).Elem(), "256"); err == nil {
		t.Fatal("expected overflow error for uint8 value 256")
	}
	var f32 float32
	if err := setFieldFromString(reflect.ValueOf(&f32).Elem(), "3.4028236e+38"); err == nil {
		t.Fatal("expected overflow error for float32 value 3.4028236e+38")
	}
}

func TestBindInput_AnonymousStructHeader(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/", "")
	c.Request.Header.Set("X-Trace", "trace-1")

	type input struct {
		BindEmbeddedInput
	}
	var in input
	if err := bindInput(c, http.MethodGet, &in); err != nil {
		t.Fatalf("bindInput: %v", err)
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
	reqCtx = context.WithValue(reqCtx, testContextKey("request-key"), "request-value")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)
	c.Set("gin-key", "gin-value")
	c.Set(requestIDContextKey, "req-123")
	c.Set(contextkeys.JWTClaims, contextClaims{userID: 7})

	ctx := newContext(c)
	if ctx.Value("gin-key") != "gin-value" || ctx.Value(testContextKey("request-key")) != "request-value" {
		t.Fatal("expected context values from gin and request context")
	}
	expectedDeadline, expectedOK := reqCtx.Deadline()
	deadline, ok := ctx.Deadline()
	if ok != expectedOK || !deadline.Equal(expectedDeadline) {
		t.Fatalf("expected deadline %v (ok=%v), got %v (ok=%v)", expectedDeadline, expectedOK, deadline, ok)
	}
	if ctx.StdContext().Value(testContextKey("request-key")) != "request-value" {
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

		c, _ = newTestContext(http.MethodGet, "/", "")
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
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"code":403`) {
			t.Fatalf("unexpected forbidden response: %d %s", w.Code, w.Body.String())
		}

		c, w = newTestContext(http.MethodGet, "/", "")
		ctx = newContext(c)
		ctx.Unauthorized("bad token")
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"code":401`) {
			t.Fatalf("unexpected unauthorized response: %d %s", w.Code, w.Body.String())
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("api error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		WriteError(c, &Error{Code: http.StatusTeapot, Message: "short and stout"})
		body := w.Body.String()
		if w.Code != http.StatusTeapot || !strings.Contains(body, `"code":418`) || strings.Contains(body, `"error"`) {
			t.Fatalf("unexpected response: %d %s", w.Code, body)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		WriteError(c, &ValidationError{Errors: []FieldError{{Field: "name", Message: "field is required"}}})
		body := w.Body.String()
		if w.Code != http.StatusUnprocessableEntity || !strings.Contains(body, `"code":422`) || strings.Contains(body, `"error"`) {
			t.Fatalf("unexpected response: %d %s", w.Code, body)
		}
	})

	t.Run("generic error", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		WriteError(c, errors.New("boom"))
		body := w.Body.String()
		if w.Code != http.StatusInternalServerError || !strings.Contains(body, `"code":500`) || strings.Contains(body, `"error"`) {
			t.Fatalf("unexpected response: %d %s", w.Code, body)
		}
		if strings.Contains(body, "boom") {
			t.Fatalf("raw error detail must not be exposed to clients: %s", body)
		}
	})

	t.Run("instance mapper", func(t *testing.T) {
		sentinel := errors.New("mapped")
		api := New(Config{})
		api.RegisterErrorMapper(func(err error) error {
			if errors.Is(err, sentinel) {
				return &Error{Code: http.StatusTeapot, Message: "mapped"}
			}
			return nil
		})

		c, w := newTestContext(http.MethodGet, "/", "")
		c.Set(ninjaAPIContextKey, api)
		WriteError(c, sentinel)
		if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), `"code":418`) {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("default mapper fallback without api", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/", "")
		WriteError(c, context.DeadlineExceeded)
		if w.Code != http.StatusRequestTimeout || !strings.Contains(w.Body.String(), `"code":408`) {
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

func TestBindModelSchemas(t *testing.T) {
	items, err := BindModelSchemas[publicSchema]([]schemaModel{
		{ID: 1, Name: "alice", Email: "alice@example.com", Password: "secret"},
		{ID: 2, Name: "bob", Email: "bob@example.com", Password: "hidden"},
	})
	if err != nil {
		t.Fatalf("BindModelSchemas: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two bound schemas, got %d", len(items))
	}
	if items[0].Model.Email != "alice@example.com" || items[1].Model.Password != "hidden" {
		t.Fatalf("expected models to be assigned, got %+v", items)
	}
	if got := items[0].Fields; len(got) != 3 || got[0] != "email" || got[1] != "id" || got[2] != "name" {
		t.Fatalf("expected fields from tags, got %v", got)
	}

	empty, err := BindModelSchemas[publicSchema]([]schemaModel{})
	if err != nil {
		t.Fatalf("BindModelSchemas empty: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected empty non-nil result for empty input, got %#v", empty)
	}
	nilItems, err := BindModelSchemas[publicSchema, schemaModel](nil)
	if err != nil {
		t.Fatalf("BindModelSchemas nil: %v", err)
	}
	if nilItems != nil {
		t.Fatalf("expected nil result for nil input, got %#v", nilItems)
	}

	if _, err := BindModelSchemas[struct{}]([]schemaModel{{ID: 3}}); err == nil {
		t.Fatalf("expected BindModelSchemas to return binding errors")
	}
}

func TestResponseModelBindsModelSchemaAtRuntime(t *testing.T) {
	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/user", func(ctx *Context, in *struct{}) (*schemaModel, error) {
		return &schemaModel{
			ID:       7,
			Name:     "alice",
			Email:    "alice@example.com",
			Password: "secret",
		}, nil
	}, ResponseModel[publicSchema]())
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/user", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var data map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := data["password"]; ok {
		t.Fatalf("expected response model to prune password, got %+v", data)
	}
	if data["email"] != "alice@example.com" || data["id"] != float64(7) {
		t.Fatalf("expected public response fields, got %+v", data)
	}
}

func TestResponseModelAcceptsAlreadyBoundSchema(t *testing.T) {
	typed, err := BindModelSchema[publicSchema](schemaModel{
		ID:       9,
		Name:     "bob",
		Email:    "bob@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("BindModelSchema: %v", err)
	}

	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/bound", func(ctx *Context, in *struct{}) (*publicSchema, error) {
		return typed, nil
	}, ResponseModel[publicSchema]())
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/bound", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Fatalf("expected already-bound schema to preserve filtering, got %s", w.Body.String())
	}
}

func TestResponseModelOverridesOpenAPISuccessSchema(t *testing.T) {
	spec := newOpenAPISpec(Config{})
	op := newOperation(http.MethodGet, "/user", func(ctx *Context, in *struct{}) (*schemaModel, error) {
		return &schemaModel{}, nil
	}, nil)
	ResponseModel[publicSchema]()(op)

	spec.addOperation(op)
	built := spec.build()
	response := built.Paths["/user"].Get.Responses["200"]
	schema := response.Content["application/json"].Schema
	if schema == nil || !strings.Contains(schema.Ref, "publicSchema") {
		t.Fatalf("expected publicSchema response ref, got %+v", schema)
	}
	component := built.Components.Schemas["publicSchema"]
	if component == nil {
		t.Fatalf("expected publicSchema component, got %+v", built.Components.Schemas)
	}
	if _, ok := component.Properties["password"]; ok {
		t.Fatalf("expected password to be omitted from response schema, got %+v", component.Properties)
	}
}

func TestResponseModelValidatesPlainResponseSchema(t *testing.T) {
	type requiredOutput struct {
		Name string `json:"name" binding:"required"`
	}

	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/invalid", func(ctx *Context, in *struct{}) (*requiredOutput, error) {
		return &requiredOutput{}, nil
	}, ResponseModel[requiredOutput]())
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/invalid", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected response validation to fail with 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResponseSchemaDescriptorSerializesModelAtRuntime(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().
		Fields("id", "email", "password").
		Exclude("password").
		ComponentName("PublicUser")

	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/descriptor", func(ctx *Context, in *struct{}) (*schemaModel, error) {
		return &schemaModel{
			ID:       11,
			Name:     "carol",
			Email:    "carol@example.com",
			Password: "secret",
		}, nil
	}, ResponseSchema(userSchema))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/descriptor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var data map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := data["password"]; ok {
		t.Fatalf("expected descriptor to prune password, got %+v", data)
	}
	if _, ok := data["name"]; ok {
		t.Fatalf("expected descriptor fields to omit name, got %+v", data)
	}
	if data["email"] != "carol@example.com" || data["id"] != float64(11) {
		t.Fatalf("expected descriptor response fields, got %+v", data)
	}
}

func TestResponseSchemaDescriptorValidatesRequiredFields(t *testing.T) {
	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/descriptor-invalid", func(ctx *Context, in *struct{}) (*requiredSchemaModel, error) {
		return &requiredSchemaModel{ID: 15, Email: "missing-name@example.com"}, nil
	}, ResponseSchema(ModelSchemaOf[requiredSchemaModel]().Fields("id", "name", "email")))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/descriptor-invalid", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected response schema validation to fail with 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResponseSchemaDescriptorSkipsExcludedRequiredFields(t *testing.T) {
	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/descriptor-excluded-required", func(ctx *Context, in *struct{}) (*requiredSchemaModel, error) {
		return &requiredSchemaModel{ID: 16, Email: "missing-name@example.com"}, nil
	}, ResponseSchema(ModelSchemaOf[requiredSchemaModel]().Fields("id", "email")))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/descriptor-excluded-required", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected excluded required field to be ignored, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResponseSchemaDescriptorOverridesOpenAPI(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().
		Fields("id", "email").
		ComponentName("PublicUser")

	spec := newOpenAPISpec(Config{})
	op := newOperation(http.MethodGet, "/descriptor", func(ctx *Context, in *struct{}) (*schemaModel, error) {
		return &schemaModel{}, nil
	}, nil)
	ResponseSchema(userSchema)(op)

	spec.addOperation(op)
	built := spec.build()
	schema := built.Paths["/descriptor"].Get.Responses["200"].Content["application/json"].Schema
	if schema == nil || schema.Ref != "#/components/schemas/PublicUser" {
		t.Fatalf("expected PublicUser response ref, got %+v", schema)
	}
	component := built.Components.Schemas["PublicUser"]
	if component == nil {
		t.Fatalf("expected PublicUser component, got %+v", built.Components.Schemas)
	}
	if _, ok := component.Properties["email"]; !ok {
		t.Fatalf("expected email field, got %+v", component.Properties)
	}
	if _, ok := component.Properties["password"]; ok {
		t.Fatalf("expected password to be omitted, got %+v", component.Properties)
	}
}

func TestResponseSchemaDescriptorDocumentsSliceOutput(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().
		Fields("id", "email").
		ComponentName("PublicUserListItem")

	spec := newOpenAPISpec(Config{})
	op := newOperation(http.MethodGet, "/descriptors", func(ctx *Context, in *struct{}) (*[]schemaModel, error) {
		return &[]schemaModel{}, nil
	}, nil)
	ResponseSchema(userSchema)(op)

	spec.addOperation(op)
	built := spec.build()
	schema := built.Paths["/descriptors"].Get.Responses["200"].Content["application/json"].Schema
	if schema == nil || schema.Type != "array" || schema.Items == nil || schema.Items.Ref != "#/components/schemas/PublicUserListItem" {
		t.Fatalf("expected array response with PublicUserListItem items, got %+v", schema)
	}
}

func TestPaginatedSchemaValidatesRequiredItemFields(t *testing.T) {
	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/page-invalid", func(ctx *Context, in *struct{}) (*pagination.Page[requiredSchemaModel], error) {
		return pagination.NewPage([]requiredSchemaModel{{
			ID:    17,
			Email: "missing-name@example.com",
		}}, 1, pagination.PageInput{Page: 1, Size: 10}), nil
	}, PaginatedSchema(ModelSchemaOf[requiredSchemaModel]().Fields("id", "name", "email")))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/page-invalid", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected paginated item validation to fail with 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPaginatedSchemaSerializesItemsAtRuntime(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().
		Fields("id", "email").
		ComponentName("PublicUserPageItem")

	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/page", func(ctx *Context, in *struct{}) (*pagination.Page[schemaModel], error) {
		return pagination.NewPage([]schemaModel{{
			ID:       13,
			Name:     "eve",
			Email:    "eve@example.com",
			Password: "secret",
		}}, 1, pagination.PageInput{Page: 1, Size: 10}), nil
	}, PaginatedSchema(userSchema))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/page", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var data map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one paginated item, got %+v", data["items"])
	}
	item := items[0].(map[string]any)
	if _, ok := item["password"]; ok {
		t.Fatalf("expected paginated schema to prune password, got %+v", item)
	}
	if _, ok := item["name"]; ok {
		t.Fatalf("expected paginated schema fields to omit name, got %+v", item)
	}
	if item["email"] != "eve@example.com" || data["total"] != float64(1) {
		t.Fatalf("expected serialized page item and metadata, got %+v", data)
	}
}

func TestCursorPaginatedSchemaSerializesItemsAtRuntime(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().Fields("id", "email")

	api := New(Config{DisableDocs: true, DisableHomepage: true})
	router := NewRouter("/runtime")
	Get(router, "/cursor-page", func(ctx *Context, in *struct{}) (*pagination.CursorPage[schemaModel], error) {
		return pagination.NewCursorPage([]schemaModel{{
			ID:       14,
			Name:     "frank",
			Email:    "frank@example.com",
			Password: "secret",
		}}, pagination.CursorPagination{Size: 5}, "next"), nil
	}, CursorPaginatedSchema(userSchema))
	api.AddRouter(router)

	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runtime/cursor-page", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "frank\"") {
		t.Fatalf("expected cursor paginated schema to prune fields, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "next_cursor") {
		t.Fatalf("expected cursor metadata to remain, got %s", w.Body.String())
	}
}

func TestPaginatedSchemaOverridesOpenAPI(t *testing.T) {
	userSchema := ModelSchemaOf[schemaModel]().
		Fields("id", "email").
		ComponentName("PublicUserPageItem")

	spec := newOpenAPISpec(Config{})
	op := newOperation(http.MethodGet, "/page", func(ctx *Context, in *struct{}) (*pagination.Page[schemaModel], error) {
		return pagination.NewPage([]schemaModel{}, 0, pagination.PageInput{}), nil
	}, nil)
	PaginatedSchema(userSchema)(op)

	spec.addOperation(op)
	built := spec.build()
	items := built.Paths["/page"].Get.Responses["200"].Content["application/json"].Schema.Properties["items"]
	if items == nil || items.Items == nil || items.Items.Ref != "#/components/schemas/PublicUserPageItem" {
		t.Fatalf("expected paginated item schema ref, got %+v", items)
	}
	component := built.Components.Schemas["PublicUserPageItem"]
	if component == nil {
		t.Fatalf("expected PublicUserPageItem component, got %+v", built.Components.Schemas)
	}
	if _, ok := component.Properties["password"]; ok {
		t.Fatalf("expected password to be omitted, got %+v", component.Properties)
	}
}

func TestModelSchemaDescriptorWrap(t *testing.T) {
	wrapped := ModelSchemaOf[schemaModel]().
		Fields("id", "name").
		Wrap(schemaModel{ID: 12, Name: "dave", Email: "dave@example.com"})

	payload, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if strings.Contains(string(payload), "email") {
		t.Fatalf("expected descriptor wrapped schema to omit email, got %s", string(payload))
	}
	if !strings.Contains(string(payload), `"name":"dave"`) {
		t.Fatalf("expected descriptor wrapped schema to include name, got %s", string(payload))
	}
}

func TestModelSchemaDescriptorModesUseAccessAndGORMConventions(t *testing.T) {
	model := schemaModeModel{
		ID:        21,
		Name:      "erin",
		Password:  "secret",
		Invite:    "invite-1",
		Status:    "pending",
		CreatedAt: time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC),
		Profile:   schemaModel{Name: "nested"},
		Tags:      []string{"admin"},
		Computed:  "derived",
	}

	readPayload, err := json.Marshal(ModelSchemaOf[schemaModeModel]().Read().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal read schema: %v", err)
	}
	var readData map[string]any
	if err := json.Unmarshal(readPayload, &readData); err != nil {
		t.Fatalf("Unmarshal read schema: %v", err)
	}
	if _, ok := readData["password"]; ok {
		t.Fatalf("expected read mode to omit top-level password, got %s", string(readPayload))
	}
	if _, ok := readData["invite_code"]; ok {
		t.Fatalf("expected read mode to omit top-level invite_code, got %s", string(readPayload))
	}
	if _, ok := readData["profile"]; !ok {
		t.Fatalf("expected read mode to keep detail fields, got %s", string(readPayload))
	}

	listPayload, err := json.Marshal(ModelSchemaOf[schemaModeModel]().List().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal list schema: %v", err)
	}
	if strings.Contains(string(listPayload), "profile") || strings.Contains(string(listPayload), "tags") {
		t.Fatalf("expected list mode to omit non-scalar fields, got %s", string(listPayload))
	}
	if !strings.Contains(string(listPayload), "created_at") {
		t.Fatalf("expected list mode to keep marshaler scalar fields, got %s", string(listPayload))
	}

	createPayload, err := json.Marshal(ModelSchemaOf[schemaModeModel]().Create().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal create schema: %v", err)
	}
	var createData map[string]any
	if err := json.Unmarshal(createPayload, &createData); err != nil {
		t.Fatalf("Unmarshal create schema: %v", err)
	}
	for _, field := range []string{"id", "created_at", "status_note", "computed"} {
		if _, ok := createData[field]; ok {
			t.Fatalf("expected create mode to omit top-level %s, got %s", field, string(createPayload))
		}
	}
	if _, ok := createData["password"]; !ok {
		t.Fatalf("expected create mode to keep writable password, got %s", string(createPayload))
	}
	if _, ok := createData["invite_code"]; !ok {
		t.Fatalf("expected create mode to keep writable create fields, got %s", string(createPayload))
	}

	updatePayload, err := json.Marshal(ModelSchemaOf[schemaModeModel]().Update().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal update schema: %v", err)
	}
	var updateData map[string]any
	if err := json.Unmarshal(updatePayload, &updateData); err != nil {
		t.Fatalf("Unmarshal update schema: %v", err)
	}
	for _, field := range []string{"id", "created_at", "invite_code", "computed"} {
		if _, ok := updateData[field]; ok {
			t.Fatalf("expected update mode to omit top-level %s, got %s", field, string(updatePayload))
		}
	}
	if _, ok := updateData["password"]; !ok {
		t.Fatalf("expected update mode to keep writable password, got %s", string(updatePayload))
	}
	if _, ok := updateData["status_note"]; !ok {
		t.Fatalf("expected update mode to keep writable status_note, got %s", string(updatePayload))
	}
}

func TestModelSchemaConvenienceConstructors(t *testing.T) {
	model := schemaModeModel{
		ID:       22,
		Name:     "gina",
		Password: "secret",
		Invite:   "invite-2",
		Status:   "approved",
		Profile:  schemaModel{Name: "nested"},
		Tags:     []string{"ops"},
		Computed: "derived",
	}

	createPayload, err := json.Marshal(ModelCreateSchemaOf[schemaModeModel]().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal create schema: %v", err)
	}
	var createData map[string]any
	if err := json.Unmarshal(createPayload, &createData); err != nil {
		t.Fatalf("Unmarshal create schema: %v", err)
	}
	if _, ok := createData["id"]; ok {
		t.Fatalf("expected create constructor to omit id, got %s", string(createPayload))
	}
	if _, ok := createData["invite_code"]; !ok {
		t.Fatalf("expected create constructor to keep create-only field, got %s", string(createPayload))
	}
	if _, ok := createData["status_note"]; ok {
		t.Fatalf("expected create constructor to omit update-only field, got %s", string(createPayload))
	}

	updatePayload, err := json.Marshal(ModelUpdateSchemaOf[schemaModeModel]().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal update schema: %v", err)
	}
	var updateData map[string]any
	if err := json.Unmarshal(updatePayload, &updateData); err != nil {
		t.Fatalf("Unmarshal update schema: %v", err)
	}
	if _, ok := updateData["invite_code"]; ok {
		t.Fatalf("expected update constructor to omit create-only field, got %s", string(updatePayload))
	}
	if _, ok := updateData["status_note"]; !ok {
		t.Fatalf("expected update constructor to keep update-only field, got %s", string(updatePayload))
	}

	listPayload, err := json.Marshal(ModelListSchemaOf[schemaModeModel]().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal list schema: %v", err)
	}
	if strings.Contains(string(listPayload), "profile") || strings.Contains(string(listPayload), "tags") {
		t.Fatalf("expected list constructor to omit relation fields, got %s", string(listPayload))
	}

	detailPreloads := ModelDetailSchemaOf[schemaDepthModel](Depth(1)).Preloads()
	if !reflect.DeepEqual(detailPreloads, []string{"Owner", "Members"}) {
		t.Fatalf("expected detail constructor to support depth preloads, got %v", detailPreloads)
	}
}

func TestModelSchemaModeAffectsOpenAPIComponents(t *testing.T) {
	type createSchema struct {
		ModelSchema[schemaModeModel] `mode:"create"`
	}

	registry := newSchemaRegistry()
	ref := registry.schemaForType(reflect.TypeOf(createSchema{}))
	if ref.Ref == "" {
		t.Fatalf("expected create schema ref, got %+v", ref)
	}
	component := registry.schemas["createSchema"]
	if component == nil {
		t.Fatalf("expected createSchema component, got %+v", registry.schemas)
	}
	if _, ok := component.Properties["id"]; ok {
		t.Fatalf("expected create mode to omit id, got %+v", component.Properties)
	}
	if _, ok := component.Properties["status_note"]; ok {
		t.Fatalf("expected create mode to omit update-only field, got %+v", component.Properties)
	}
	if _, ok := component.Properties["password"]; !ok {
		t.Fatalf("expected create mode to keep write-only password input, got %+v", component.Properties)
	}
}

func TestModelSchemaDescriptorDepthSerializesNestedModels(t *testing.T) {
	model := schemaDepthModel{
		ID:   31,
		Name: "team",
		Owner: schemaRelationModel{
			ID:     1,
			Name:   "owner",
			Secret: "owner-secret",
		},
		Members: []schemaRelationModel{{
			ID:     2,
			Name:   "member",
			Secret: "member-secret",
		}},
		Internal: "hidden",
	}

	shallowPayload, err := json.Marshal(ModelSchemaOf[schemaDepthModel]().Read().Wrap(model))
	if err != nil {
		t.Fatalf("Marshal shallow schema: %v", err)
	}
	if !strings.Contains(string(shallowPayload), "owner-secret") {
		t.Fatalf("expected depth 0 to preserve existing nested serialization, got %s", string(shallowPayload))
	}

	deepPayload, err := json.Marshal(ModelSchemaOf[schemaDepthModel]().Read().Depth(1).Wrap(model))
	if err != nil {
		t.Fatalf("Marshal depth schema: %v", err)
	}
	if strings.Contains(string(deepPayload), "owner-secret") || strings.Contains(string(deepPayload), "member-secret") {
		t.Fatalf("expected depth 1 to prune nested write-only fields, got %s", string(deepPayload))
	}
	if strings.Contains(string(deepPayload), "internal") {
		t.Fatalf("expected read mode to prune top-level write-only fields, got %s", string(deepPayload))
	}
	if !strings.Contains(string(deepPayload), `"owner"`) || !strings.Contains(string(deepPayload), `"members"`) {
		t.Fatalf("expected depth schema to keep nested relation fields, got %s", string(deepPayload))
	}
}

func TestModelSchemaDepthAffectsOpenAPIComponents(t *testing.T) {
	schema := ModelSchemaOf[schemaDepthModel]().
		Read().
		Depth(1).
		ComponentName("DepthParent")

	spec := newOpenAPISpec(Config{})
	op := newOperation(http.MethodGet, "/depth", func(ctx *Context, in *struct{}) (*schemaDepthModel, error) {
		return &schemaDepthModel{}, nil
	}, nil)
	ResponseSchema(schema)(op)

	spec.addOperation(op)
	built := spec.build()
	parent := built.Components.Schemas["DepthParent"]
	if parent == nil {
		t.Fatalf("expected DepthParent component, got %+v", built.Components.Schemas)
	}
	if _, ok := parent.Properties["internal"]; ok {
		t.Fatalf("expected read mode to omit parent write-only field, got %+v", parent.Properties)
	}
	owner := parent.Properties["owner"]
	if owner == nil || owner.Ref == "" {
		t.Fatalf("expected owner relation ref, got %+v", owner)
	}
	childName := strings.TrimPrefix(owner.Ref, "#/components/schemas/")
	child := built.Components.Schemas[childName]
	if child == nil {
		t.Fatalf("expected child component %q, got %+v", childName, built.Components.Schemas)
	}
	if _, ok := child.Properties["secret"]; ok {
		t.Fatalf("expected depth schema to omit child write-only field, got %+v", child.Properties)
	}
	if _, ok := child.Properties["name"]; !ok {
		t.Fatalf("expected depth schema to keep child public field, got %+v", child.Properties)
	}
}

func TestModelSchemaDescriptorPreloadsFollowDepthAndFieldFilters(t *testing.T) {
	all := ModelSchemaOf[schemaDepthModel]().Read().Depth(1).Preloads()
	if !reflect.DeepEqual(all, []string{"Owner", "Members"}) {
		t.Fatalf("expected top-level relation preloads, got %v", all)
	}

	filtered := ModelSchemaOf[schemaDepthModel]().
		Read().
		Depth(1).
		Fields("id", "owner", "members").
		Exclude("members").
		Preloads()
	if !reflect.DeepEqual(filtered, []string{"Owner"}) {
		t.Fatalf("expected filtered relation preloads, got %v", filtered)
	}

	if got := ModelSchemaOf[schemaDepthModel]().Read().Preloads(); got != nil {
		t.Fatalf("expected depth 0 to have no preloads, got %v", got)
	}
	if got := ModelSchemaOf[schemaDepthModel]().List().Depth(1).Preloads(); got != nil {
		t.Fatalf("expected list mode to omit relation preloads, got %v", got)
	}
}

func TestModelSchemaDescriptorPreloadsIncludeNestedPaths(t *testing.T) {
	preloads := ModelSchemaOf[schemaDeepModel]().Read().Depth(2).Preloads()
	want := []string{"Parent", "Parent.Owner"}
	if !reflect.DeepEqual(preloads, want) {
		t.Fatalf("Preloads() = %v, want %v", preloads, want)
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

	schemes := map[string]SecurityScheme{
		"bearerAuth": HTTPBearerSecurityScheme("JWT"),
		"basicAuth":  HTTPBasicSecurityScheme(),
		"apiKey":     APIKeyHeaderSecurityScheme("X-API-Key"),
		"oauth2": OAuth2SecurityScheme(OAuthFlows{
			ClientCredentials: &OAuthFlow{
				TokenURL: "https://example.com/token",
				Scopes:   map[string]string{"read": "read data"},
			},
		}),
	}
	clonedSchemes := cloneSecuritySchemes(schemes)
	scheme := clonedSchemes["bearerAuth"]
	scheme.BearerFormat = "opaque"
	clonedSchemes["bearerAuth"] = scheme
	clonedSchemes["oauth2"].Flows.ClientCredentials.Scopes["read"] = "changed"
	if schemes["bearerAuth"].BearerFormat != "JWT" {
		t.Fatalf("expected security schemes clone to be independent: %+v", schemes)
	}
	if schemes["oauth2"].Flows.ClientCredentials.Scopes["read"] != "read data" {
		t.Fatalf("expected oauth2 scopes clone to be independent: %+v", schemes["oauth2"].Flows.ClientCredentials.Scopes)
	}
	if schemes["basicAuth"].Scheme != "basic" || schemes["apiKey"].In != "header" {
		t.Fatalf("unexpected security scheme helpers: %+v", schemes)
	}

	if err := NewError(http.StatusBadRequest, "bad"); err.Code != http.StatusBadRequest || err.Message != "bad" {
		t.Fatalf("unexpected NewError result: %+v", err)
	}
	if err := NewError(http.StatusConflict, "duplicate"); err.Code != http.StatusConflict {
		t.Fatalf("unexpected NewError result: %+v", err)
	}
	if ForbiddenError().Error() == "" || (&ValidationError{Errors: []FieldError{{Field: "x", Message: "y"}}}).Error() == "" {
		t.Fatal("expected error strings to be non-empty")
	}
}

func TestOptionHelpers(t *testing.T) {
	noopAuth := func(c *gin.Context) {
		c.Next()
	}
	router := NewRouter(
		"/users",
		WithTags("Users", "Admin"),
		WithSecurity("oauth2", "read"),
		WithBearerAuth(noopAuth),
		WithBasicAuth(noopAuth),
		WithAPIKeyAuth("apiKey", noopAuth),
		WithOAuth2Auth("write"),
		WithVersion("v1"),
	)
	WithTagDescription("Users", "user operations")(router)
	if len(router.tags) != 2 || len(router.security) != 5 || router.version != "v1" {
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
	BearerAuth(noopAuth)(op)
	BasicAuth(noopAuth)(op)
	APIKeyAuth("apiKey", noopAuth)(op)
	OAuth2Auth("write")(op)
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
	CursorPaginated[schemaSample]()(op)
	CursorPaginatedResponse[schemaSample](http.StatusMultiStatus, "cursor partial")(op)
	Timeout(time.Second)(op)
	RateLimit(2, 3)(op)
	WithTransaction()(op)

	if op.spec.summary != "list users" || op.spec.description != "full description" || op.spec.operationID != "listUsers" {
		t.Fatalf("unexpected operation metadata: %+v", op)
	}
	if !op.spec.deprecated || !op.spec.excludeFromDocs || op.spec.successStatus != http.StatusAccepted || len(op.spec.security) != 5 {
		t.Fatalf("unexpected operation options: %+v", op)
	}
	if op.spec.tagDescriptions["Users"] != "user operations" || op.spec.paginatedItemType == nil || op.spec.cursorPaginatedItemType == nil || op.behavior.timeout != time.Second || op.behavior.rateLimit == nil || op.cache.config == nil || !op.cache.etagEnabled {
		t.Fatalf("unexpected extended operation options: %+v", op)
	}
	if len(op.spec.responses) != 4 || op.spec.responses[0].responseType == nil || op.spec.responses[1].responseType != nil || op.spec.responses[2].paginatedItemType == nil || op.spec.responses[3].cursorPaginatedItemType == nil {
		t.Fatalf("unexpected documented responses: %+v", op.spec.responses)
	}
	if !op.behavior.withTransaction {
		t.Fatalf("expected WithTransaction to enable transaction wrapping: %+v", op)
	}
}

func TestAuthHelpersRequireMiddleware(t *testing.T) {
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected %s to panic without middleware", name)
			}
		}()
		fn()
	}

	assertPanics("WithBearerAuth", func() { WithBearerAuth()(NewRouter("/")) })
	assertPanics("WithBasicAuth", func() { WithBasicAuth()(NewRouter("/")) })
	assertPanics("WithAPIKeyAuth", func() { WithAPIKeyAuth("apiKey")(NewRouter("/")) })
	assertPanics("BearerAuth", func() { BearerAuth()(&operation{}) })
	assertPanics("BasicAuth", func() { BasicAuth()(&operation{}) })
	assertPanics("APIKeyAuth", func() { APIKeyAuth("apiKey")(&operation{}) })
}

func TestRouterRegistrationHelpers(t *testing.T) {
	router := NewRouter("/items", WithTags("Items"))
	WithTagDescriptions(map[string]string{
		"Items": "item operations",
		"Admin": "admin operations",
	})(router)

	Put[schemaSample, schemaSample](router, "/:id", func(ctx *Context, in *schemaSample) (*schemaSample, error) {
		return in, nil
	})
	Patch[schemaSample, schemaSample](router, "/:id", func(ctx *Context, in *schemaSample) (*schemaSample, error) {
		return in, nil
	})

	if len(router.operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(router.operations))
	}

	putOp := router.operations[0]
	patchOp := router.operations[1]
	if putOp.route.method != http.MethodPut || patchOp.route.method != http.MethodPatch {
		t.Fatalf("unexpected methods: %s %s", putOp.route.method, patchOp.route.method)
	}
	if putOp.spec.successStatus != http.StatusOK || patchOp.spec.successStatus != http.StatusOK {
		t.Fatalf("unexpected success statuses: %d %d", putOp.spec.successStatus, patchOp.spec.successStatus)
	}
	if putOp.spec.tagDescriptions["Items"] != "item operations" || patchOp.spec.tagDescriptions["Admin"] != "admin operations" {
		t.Fatalf("expected tag descriptions to be copied into operations: %+v %+v", putOp.spec.tagDescriptions, patchOp.spec.tagDescriptions)
	}

	router.tagDescriptions["Items"] = "mutated"
	if putOp.spec.tagDescriptions["Items"] != "item operations" || patchOp.spec.tagDescriptions["Items"] != "item operations" {
		t.Fatalf("expected operation tag descriptions to be cloned, got %+v %+v", putOp.spec.tagDescriptions, patchOp.spec.tagDescriptions)
	}
}

func TestNewOperationNilOutputAndVoidOperation(t *testing.T) {
	op := newOperation(http.MethodGet, "/", func(ctx *Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	}, nil)

	c, _ := newTestContext(http.MethodGet, "/", "")
	op.route.ginHandler(c)
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204 for nil output, got %d", c.Writer.Status())
	}

	voidOp := newVoidOperation(http.MethodDelete, "/:id", func(ctx *Context, input *struct{}) error {
		return nil
	}, nil)
	c, _ = newTestContext(http.MethodDelete, "/1", "")
	voidOp.route.ginHandler(c)
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204 for void operation, got %d", c.Writer.Status())
	}
}

func TestOperationsWithTransactionHandlers(t *testing.T) {
	beginCalled := false
	commitCalled := false
	rollbackCalled := false
	withTxCalled := false
	handlers := &TransactionHandlers{}
	handlers.Begin = func(*gin.Context) error {
		beginCalled = true
		return nil
	}
	handlers.Commit = func(*gin.Context) error {
		commitCalled = true
		return nil
	}
	handlers.Rollback = func(*gin.Context) error {
		rollbackCalled = true
		return nil
	}
	handlers.WithTransaction = func(c *gin.Context, fn func() error) error {
		withTxCalled = true
		if err := handlers.Begin(c); err != nil {
			return err
		}
		if err := fn(); err != nil {
			_ = handlers.Rollback(c)
			return err
		}
		return handlers.Commit(c)
	}
	api := New(Config{DisableGinDefault: true, DisableHomepage: true, DisableOpenAPI: true, TransactionHandlers: handlers})

	op := newOperation(http.MethodGet, "/", func(ctx *Context, input *struct{}) (*schemaSample, error) {
		return &schemaSample{Name: "ok"}, nil
	}, nil)
	WithTransaction()(op)

	c, w := newTestContext(http.MethodGet, "/", "")
	c.Set(ninjaAPIContextKey, api)
	op.route.ginHandler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for transaction-wrapped operation, got %d", w.Code)
	}
	if !withTxCalled || !beginCalled || !commitCalled || rollbackCalled {
		t.Fatalf("unexpected transaction calls: with=%v begin=%v commit=%v rollback=%v", withTxCalled, beginCalled, commitCalled, rollbackCalled)
	}

	beginCalled = false
	commitCalled = false
	rollbackCalled = false
	withTxCalled = false

	voidOp := newVoidOperation(http.MethodDelete, "/:id", func(ctx *Context, input *struct{}) error {
		return errors.New("boom")
	}, nil)
	WithTransaction()(voidOp)

	c, w = newTestContext(http.MethodDelete, "/1", "")
	c.Set(ninjaAPIContextKey, api)
	voidOp.route.ginHandler(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for transaction-wrapped void operation error, got %d", w.Code)
	}
	if !withTxCalled || !beginCalled || !rollbackCalled || commitCalled {
		t.Fatalf("unexpected void transaction calls: with=%v begin=%v commit=%v rollback=%v", withTxCalled, beginCalled, commitCalled, rollbackCalled)
	}
}

func TestOperationWithTransactionRollsBackWhenTimeoutContextExpires(t *testing.T) {
	committed := make(chan struct{}, 1)
	rolledBack := make(chan struct{}, 1)
	transactionDone := make(chan error, 1)

	handlers := &TransactionHandlers{}
	handlers.WithTransaction = func(_ *gin.Context, fn func() error) error {
		err := fn()
		transactionDone <- err
		if err != nil {
			rolledBack <- struct{}{}
			return err
		}
		committed <- struct{}{}
		return nil
	}
	api := New(Config{DisableGinDefault: true, DisableHomepage: true, DisableOpenAPI: true, TransactionHandlers: handlers})

	op := newVoidOperation(http.MethodGet, "/", func(ctx *Context, input *struct{}) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}, nil)
	WithTransaction()(op)
	Timeout(10 * time.Millisecond)(op)
	op.finalize()

	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		c.Set(ninjaAPIContextKey, api)
		op.route.ginHandler(c)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected timeout response, got %d", w.Code)
	}

	select {
	case err := <-transactionDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected transaction callback to receive deadline error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("transaction callback did not finish")
	}

	select {
	case <-rolledBack:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected custom transaction wrapper to roll back")
	}

	select {
	case <-committed:
		t.Fatal("expected custom transaction wrapper not to commit after timeout")
	default:
	}
}
