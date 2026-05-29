package ninja

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pkg/i18n"
)

var validate = func() *validator.Validate {
	v := validator.New()
	// Use the "binding" tag (gin convention) instead of the default "validate" tag.
	v.SetTagName("binding")
	return v
}()

var timeType = reflect.TypeOf(time.Time{})

const defaultJSONBodyLimit int64 = 32 << 20 // 32 MiB

// bindingMetadataCache stores precomputed binding field/tag metadata keyed by
// reflect.Type. sync.Map makes it safe for concurrent requests, and entries
// intentionally live for the process lifetime because Go struct types are
// stable once compiled.
var bindingMetadataCache sync.Map

type bindingMetadata struct {
	fields   []bindingField
	hasQuery bool
	hasForm  bool
}

type bindingField struct {
	index        []int
	name         string
	pathTag      string
	queryTag     string
	formTag      string
	headerTag    string
	cookieTag    string
	fileTag      string
	defaultValue string
	isNonBody    bool
}

// bindInput populates the input struct from the incoming gin request.
//
// Tag conventions:
//   - `path:"name"`   – URL path parameter (e.g. /users/:id)
//   - `query:"name"`  – URL query parameter
//   - `form:"name"`   – application/x-www-form-urlencoded form-body parameter
//   - `header:"name"` – request header
//   - `cookie:"name"` – request cookie
//   - `json:"name"`   – request JSON body field (POST/PUT/PATCH only)
//   - `binding:"…"`   – go-playground/validator constraints
func bindInput(c *gin.Context, method string, input interface{}) error {
	t := reflect.TypeOf(input)
	v := reflect.ValueOf(input)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("input must be a struct, got %s", t.Kind())
	}

	meta := getBindingMetadata(t)
	formBody := isBodyMethod(method) && isFormURLEncodedRequest(c)
	multipartRequest := isMultipartRequest(c)
	var formValues map[string][]string
	if formBody && meta.hasForm {
		var err error
		formValues, err = formBodyValues(c)
		if err != nil {
			return err
		}
	}
	var multipartForm *multipart.Form
	if multipartRequest {
		form, err := c.MultipartForm()
		if err != nil {
			return &Error{
				Code:    http.StatusBadRequest,
				Message: "invalid multipart form",
			}
		}
		multipartForm = form
	}
	var queryValues map[string][]string
	if meta.hasQuery {
		queryValues = c.Request.URL.Query()
	}
	willBindJSON := isBodyMethod(method) && !multipartRequest && !formBody

	nonBodyValues, err := bindRequestFields(c, meta, v, queryValues, formValues, formBody, multipartForm, willBindJSON)
	if err != nil {
		return err
	}

	// Bind JSON body for mutating methods.
	if willBindJSON {
		body, err := readJSONBody(c, defaultJSONBodyLimit)
		if err != nil {
			return err
		}

		if len(body) > 0 {
			if err := json.Unmarshal(body, input); err != nil {
				return &Error{
					Code:    http.StatusBadRequest,
					Message: "invalid request body",
				}
			}
			restoreFieldValues(v, nonBodyValues)
		}
	}

	if err := applyDefaults(c, meta, v); err != nil {
		return err
	}

	// Run validation.
	if err := validate.Struct(input); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return buildValidationError(ve, localeFromContext(c))
		}
		return err
	}
	return nil
}

func readJSONBody(c *gin.Context, limit int64) ([]byte, error) {
	// Read one extra byte so oversized bodies are reported explicitly instead of
	// being truncated and decoded as invalid JSON. Bodies up to limit bytes are
	// accepted; len(body) > limit proves at least one byte exceeded the limit.
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, limit+1))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, requestBodyTooLargeError()
		}
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, requestBodyTooLargeError()
	}
	// Restore body so gin middleware can re-read it if needed.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func requestBodyTooLargeError() *Error {
	return &Error{
		Code:    http.StatusRequestEntityTooLarge,
		Message: "request body too large",
	}
}

func getBindingMetadata(t reflect.Type) *bindingMetadata {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return &bindingMetadata{}
	}
	if cached, ok := bindingMetadataCache.Load(t); ok {
		return cached.(*bindingMetadata)
	}
	meta := buildBindingMetadata(t)
	actual, _ := bindingMetadataCache.LoadOrStore(t, meta)
	return actual.(*bindingMetadata)
}

func buildBindingMetadata(t reflect.Type) *bindingMetadata {
	meta := &bindingMetadata{}
	buildBindingMetadataInto(t, nil, meta)
	return meta
}

func buildBindingMetadataInto(t reflect.Type, prefix []int, meta *bindingMetadata) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() && !field.Anonymous {
			continue
		}
		index := append(append([]int(nil), prefix...), i)
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			buildBindingMetadataInto(deref(field.Type), index, meta)
			continue
		}
		bf := bindingField{
			index:        index,
			name:         field.Name,
			pathTag:      field.Tag.Get("path"),
			queryTag:     field.Tag.Get("query"),
			formTag:      field.Tag.Get("form"),
			headerTag:    field.Tag.Get("header"),
			cookieTag:    field.Tag.Get("cookie"),
			fileTag:      field.Tag.Get("file"),
			defaultValue: field.Tag.Get("default"),
		}
		bf.isNonBody = bf.pathTag != "" ||
			bf.queryTag != "" ||
			bf.formTag != "" ||
			bf.headerTag != "" ||
			bf.cookieTag != "" ||
			bf.fileTag != ""
		if bf.queryTag != "" && bf.queryTag != "-" {
			meta.hasQuery = true
		}
		if bf.formTag != "" && bf.formTag != "-" {
			meta.hasForm = true
		}
		meta.fields = append(meta.fields, bf)
	}
}

func hasFormFields(t reflect.Type) bool {
	return getBindingMetadata(t).hasForm
}

func hasQueryFields(t reflect.Type) bool {
	return getBindingMetadata(t).hasQuery
}

func formBodyValues(c *gin.Context) (map[string][]string, error) {
	if c.Request.PostForm == nil {
		// PostForm may already be populated by upstream middleware or tests.
		// Parse only when needed so we don't redo form parsing work.
		if err := c.Request.ParseForm(); err != nil {
			return nil, &Error{
				Code:    http.StatusBadRequest,
				Message: "invalid form body",
			}
		}
	}
	return c.Request.PostForm, nil
}

func bindRequestFields(
	c *gin.Context,
	meta *bindingMetadata,
	v reflect.Value,
	queryValues map[string][]string,
	formValues map[string][]string,
	formBody bool,
	multipartForm *multipart.Form,
	collectSnapshots bool,
) ([]fieldValueSnapshot, error) {
	var snapshots []fieldValueSnapshot
	for _, field := range meta.fields {
		fv := fieldByIndexAlloc(v, field.index)
		if !fv.CanSet() {
			continue
		}

		if field.pathTag != "" {
			raw := c.Param(field.pathTag)
			if raw != "" {
				if err := setFieldFromStrings(fv, valuesForStringField(fv, raw)); err != nil {
					return nil, &Error{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("path param '%s': %s", field.pathTag, err.Error()),
					}
				}
			}
		}

		if field.headerTag != "" {
			raw := c.Request.Header.Values(field.headerTag)
			if len(raw) > 0 {
				if err := setFieldFromStrings(fv, valuesForStringField(fv, raw...)); err != nil {
					return nil, &Error{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("header '%s': %s", field.headerTag, err.Error()),
					}
				}
			}
		}

		if field.cookieTag != "" {
			raw, err := c.Cookie(field.cookieTag)
			if err == nil && raw != "" {
				if err := setFieldFromStrings(fv, valuesForStringField(fv, raw)); err != nil {
					return nil, &Error{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("cookie '%s': %s", field.cookieTag, err.Error()),
					}
				}
			}
		}

		if field.queryTag != "" && field.queryTag != "-" {
			if raw := queryValues[field.queryTag]; len(raw) > 0 {
				if err := setFieldFromStrings(fv, raw); err != nil {
					return nil, &Error{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("query field '%s': %s", field.queryTag, err.Error()),
					}
				}
			}
		}

		if formBody && field.formTag != "" && field.formTag != "-" {
			if raw := formValues[field.formTag]; len(raw) > 0 {
				if err := setFieldFromStrings(fv, raw); err != nil {
					return nil, &Error{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("form field '%s': %s", field.formTag, err.Error()),
					}
				}
			}
		}

		if multipartForm != nil {
			if field.formTag != "" && field.formTag != "-" {
				values := multipartForm.Value[field.formTag]
				if len(values) > 0 {
					if err := setFieldFromStrings(fv, values); err != nil {
						return nil, &Error{
							Code:    http.StatusBadRequest,
							Message: fmt.Sprintf("form field '%s': %s", field.formTag, err.Error()),
						}
					}
				}
			} else if err := bindMultipartFileValue(field, fv, multipartForm.File); err != nil {
				return nil, err
			}
		}

		if collectSnapshots && field.isNonBody {
			copyValue := reflect.New(fv.Type()).Elem()
			copyValue.Set(fv)
			snapshots = append(snapshots, fieldValueSnapshot{index: field.index, value: copyValue})
		}
	}
	return snapshots, nil
}

func bindMultipartFileValue(field bindingField, fv reflect.Value, filesByName map[string][]*multipart.FileHeader) error {
	if field.fileTag == "" || field.fileTag == "-" {
		return nil
	}
	files := filesByName[field.fileTag]
	if len(files) == 0 {
		return nil
	}
	if err := setFileField(fv, files); err != nil {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("file field '%s': %s", field.fileTag, err.Error()),
		}
	}
	return nil
}

func applyDefaults(c *gin.Context, meta *bindingMetadata, v reflect.Value) error {
	for _, field := range meta.fields {
		fv := fieldByIndexAlloc(v, field.index)
		if !fv.CanSet() {
			continue
		}

		if field.defaultValue == "" || !isZeroValue(fv) {
			continue
		}

		switch {
		case field.headerTag != "":
			if c.GetHeader(field.headerTag) != "" {
				continue
			}
		case field.cookieTag != "":
			if _, err := c.Cookie(field.cookieTag); err == nil {
				continue
			}
		case field.queryTag != "":
			if hasQueryValue(c, field.queryTag) {
				continue
			}
		case field.formTag != "":
			if hasFormBodyValue(c, field.formTag) {
				continue
			}
		default:
			continue
		}

		if err := setFieldFromString(fv, field.defaultValue); err != nil {
			return &Error{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("default for field '%s': %s", field.name, err.Error()),
			}
		}
	}
	return nil
}

func valuesForStringField(fv reflect.Value, raw ...string) []string {
	if len(raw) == 0 {
		return nil
	}
	if !shouldBindStringsAsSlice(fv) {
		return raw[:1]
	}
	values := make([]string, 0, len(raw))
	for _, value := range raw {
		parts := splitCommaValues(value)
		if len(parts) == 0 {
			continue
		}
		values = append(values, parts...)
	}
	return values
}

// setFieldFromString converts a raw string value into the target reflect.Value.
func setFieldFromString(fv reflect.Value, raw string) error {
	if fv.Kind() == reflect.Ptr {
		elem := reflect.New(fv.Type().Elem())
		if err := setFieldFromString(elem.Elem(), raw); err != nil {
			return err
		}
		fv.Set(elem)
		return nil
	}

	// Handle framework-supported common types before generic TextUnmarshaler
	// parsing so date-only values remain accepted for time.Time fields.
	if fv.Type() == timeType {
		parsed, err := parseTimeValue(raw)
		if err != nil {
			return err
		}
		fv.Set(reflect.ValueOf(parsed))
		return nil
	}

	if fv.CanAddr() {
		if unmarshaler, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(raw))
		}
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	default:
		return fmt.Errorf("unsupported kind %s", fv.Kind())
	}
	return nil
}

func parseTimeValue(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func setFieldFromStrings(fv reflect.Value, raw []string) error {
	if len(raw) == 0 {
		return nil
	}
	if shouldBindStringsAsSlice(fv) {
		slice := reflect.MakeSlice(fv.Type(), 0, len(raw))
		for _, item := range raw {
			elem := reflect.New(fv.Type().Elem()).Elem()
			if err := setFieldFromString(elem, item); err != nil {
				return err
			}
			slice = reflect.Append(slice, elem)
		}
		fv.Set(slice)
		return nil
	}
	return setFieldFromString(fv, raw[0])
}

func shouldBindStringsAsSlice(fv reflect.Value) bool {
	return fv.Kind() == reflect.Slice && !implementsTextUnmarshalerValue(fv)
}

func implementsTextUnmarshalerValue(fv reflect.Value) bool {
	if fv.CanAddr() {
		if _, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return true
		}
	}
	return false
}

func setFileField(fv reflect.Value, files []*multipart.FileHeader) error {
	switch {
	case isUploadedFilePointerType(fv.Type()):
		fv.Set(reflect.ValueOf(newUploadedFile(files[0])))
		return nil
	case isMultipartFileHeaderPointerType(fv.Type()):
		fv.Set(reflect.ValueOf(files[0]))
		return nil
	case isUploadedFileSliceType(fv.Type()):
		slice := reflect.MakeSlice(fv.Type(), 0, len(files))
		for _, file := range files {
			slice = reflect.Append(slice, reflect.ValueOf(newUploadedFile(file)))
		}
		fv.Set(slice)
		return nil
	case isMultipartFileHeaderSliceType(fv.Type()):
		slice := reflect.MakeSlice(fv.Type(), 0, len(files))
		for _, file := range files {
			slice = reflect.Append(slice, reflect.ValueOf(file))
		}
		fv.Set(slice)
		return nil
	default:
		return fmt.Errorf("unsupported file field type %s", fv.Type())
	}
}

func isZeroValue(v reflect.Value) bool {
	return v.IsZero()
}

// isBodyMethod returns true for methods that carry a request body.
func isBodyMethod(method string) bool {
	m := strings.ToUpper(method)
	return m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch
}

func isMultipartRequest(c *gin.Context) bool {
	return strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data")
}

func isFormURLEncodedRequest(c *gin.Context) bool {
	return strings.HasPrefix(strings.ToLower(c.ContentType()), "application/x-www-form-urlencoded")
}

func hasQueryValue(c *gin.Context, name string) bool {
	return c.Request.URL.Query().Has(name)
}

func hasFormBodyValue(c *gin.Context, name string) bool {
	if _, ok := c.GetPostForm(name); ok {
		return true
	}
	return false
}

type fieldValueSnapshot struct {
	index []int
	value reflect.Value
}

func restoreFieldValues(root reflect.Value, snapshots []fieldValueSnapshot) {
	for _, snapshot := range snapshots {
		field := root.FieldByIndex(snapshot.index)
		if field.CanSet() {
			field.Set(snapshot.value)
		}
	}
}

// fieldByIndexAlloc resolves a cached field index path for binding/defaults and
// allocates nil pointer structs encountered along the path so embedded fields
// can be populated.
func fieldByIndexAlloc(root reflect.Value, index []int) reflect.Value {
	v := root
	for _, i := range index {
		for v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// localeFromContext returns the negotiated locale stored by the I18n middleware.
// Falls back to the default locale ("en") when the middleware is absent or the
// context is nil.
func localeFromContext(c *gin.Context) string {
	if c == nil {
		return i18n.Default
	}
	v, exists := c.Get(contextkeys.Locale)
	if !exists {
		return i18n.Default
	}
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return i18n.Default
}

// buildValidationError converts validator.ValidationErrors into our ValidationError type.
// Messages are translated using the supplied locale (e.g. "en", "zh").
func buildValidationError(ve validator.ValidationErrors, locale string) *ValidationError {
	errs := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		errs = append(errs, FieldError{
			Field:   strings.ToLower(fe.Field()),
			Message: i18n.TranslateValidation(fe.Tag(), fe.Param(), locale),
		})
	}
	return &ValidationError{Errors: errs}
}
