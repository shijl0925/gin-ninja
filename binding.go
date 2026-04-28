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
	"net/url"
	"reflect"
	"strconv"
	"strings"
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

// bindInput populates the input struct from the incoming gin request.
//
// Tag conventions:
//   - `path:"name"`   – URL path parameter (e.g. /users/:id)
//   - `form:"name"`   – query-string parameter for all methods, or form-body for POST
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

	// Bind path + header fields via custom reflection walk.
	if err := bindSpecialFields(c, t, v); err != nil {
		return err
	}

	// Use the framework binder instead of gin's generic form binder so request
	// sources keep gin-ninja's documented precedence: path/header/cookie/query
	// and form values are restored if a JSON body binds the same field.
	formBody := isBodyMethod(method) && isFormURLEncodedRequest(c)
	if hasFormFields(t) {
		values, err := formValues(c, method)
		if err != nil {
			return err
		}
		if err := bindFormFields(t, v, values, formBody); err != nil {
			return err
		}
	}

	if isMultipartRequest(c) {
		if err := bindMultipartFields(c, t, v); err != nil {
			return err
		}
	}

	nonBodyValues := collectNonBodyFieldValues(t, v)

	// Bind JSON body for mutating methods.
	if isBodyMethod(method) && !isMultipartRequest(c) && !isFormURLEncodedRequest(c) {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 32<<20)) // 32 MB limit
		if err != nil {
			return err
		}
		// Restore body so gin middleware can re-read it if needed.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		if len(body) > 0 {
			if err := json.Unmarshal(body, input); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "INVALID_JSON",
					Message: "invalid request body",
				}
			}
			restoreFieldValues(v, nonBodyValues)
		}
	}

	if err := applyDefaults(c, t, v); err != nil {
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

func hasFormFields(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() && !field.Anonymous {
			continue
		}
		if field.Anonymous && hasFormFields(field.Type) {
			return true
		}
		if tag := field.Tag.Get("form"); tag != "" && tag != "-" {
			return true
		}
	}
	return false
}

func formValues(c *gin.Context, method string) (url.Values, error) {
	values := url.Values{}
	for key, items := range c.Request.URL.Query() {
		values[key] = append([]string(nil), items...)
	}
	if isBodyMethod(method) && isFormURLEncodedRequest(c) {
		if err := c.Request.ParseForm(); err != nil {
			return nil, &Error{
				Status:  http.StatusBadRequest,
				Code:    "INVALID_FORM",
				Message: "invalid form body",
			}
		}
		for key, items := range c.Request.PostForm {
			values[key] = append(values[key], items...)
		}
	}
	return values, nil
}

func bindFormFields(t reflect.Type, v reflect.Value, values url.Values, formBody bool) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			if err := bindFormFields(deref(field.Type), derefValue(fv), values, formBody); err != nil {
				return err
			}
			continue
		}
		if !fv.CanSet() {
			continue
		}
		formTag := field.Tag.Get("form")
		if formTag == "" || formTag == "-" {
			continue
		}
		raw, ok := values[formTag]
		if !ok || len(raw) == 0 {
			continue
		}
		if err := setFieldFromStrings(fv, raw); err != nil {
			code := "INVALID_QUERY"
			if formBody {
				code = "INVALID_FORM"
			}
			return &Error{
				Status:  http.StatusBadRequest,
				Code:    code,
				Message: fmt.Sprintf("form field '%s': %s", formTag, err.Error()),
			}
		}
	}
	return nil
}

func applyDefaults(c *gin.Context, t reflect.Type, v reflect.Value) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			if err := applyDefaults(c, deref(field.Type), derefValue(fv)); err != nil {
				return err
			}
			continue
		}

		rawDefault := field.Tag.Get("default")
		if rawDefault == "" || !isZeroValue(fv) {
			continue
		}

		switch {
		case field.Tag.Get("header") != "":
			if c.GetHeader(field.Tag.Get("header")) != "" {
				continue
			}
		case field.Tag.Get("cookie") != "":
			if _, err := c.Cookie(field.Tag.Get("cookie")); err == nil {
				continue
			}
		case field.Tag.Get("form") != "":
			if hasFormValue(c, field.Tag.Get("form")) {
				continue
			}
		default:
			continue
		}

		if err := setFieldFromString(fv, rawDefault); err != nil {
			return &Error{
				Status:  http.StatusBadRequest,
				Code:    "BAD_DEFAULT_VALUE",
				Message: fmt.Sprintf("default for field '%s': %s", field.Name, err.Error()),
			}
		}
	}
	return nil
}

func bindMultipartFields(c *gin.Context, t reflect.Type, v reflect.Value) error {
	form, err := c.MultipartForm()
	if err != nil {
		return &Error{
			Status:  http.StatusBadRequest,
			Code:    "INVALID_MULTIPART",
			Message: "invalid multipart form",
		}
	}
	return bindMultipartValue(t, v, form)
}

func bindMultipartValue(t reflect.Type, v reflect.Value, form *multipart.Form) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			if err := bindMultipartValue(deref(field.Type), derefValue(fv), form); err != nil {
				return err
			}
			continue
		}

		if formTag := field.Tag.Get("form"); formTag != "" {
			values := form.Value[formTag]
			if len(values) == 0 {
				continue
			}
			if err := setFieldFromStrings(fv, values); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "BAD_FORM_VALUE",
					Message: fmt.Sprintf("form field '%s': %s", formTag, err.Error()),
				}
			}
			continue
		}

		if fileTag := field.Tag.Get("file"); fileTag != "" {
			files := form.File[fileTag]
			if len(files) == 0 {
				continue
			}
			if err := setFileField(fv, files); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "BAD_FILE_FIELD",
					Message: fmt.Sprintf("file field '%s': %s", fileTag, err.Error()),
				}
			}
		}
	}
	return nil
}

// bindSpecialFields walks the struct fields and binds path, header, and cookie params.
func bindSpecialFields(c *gin.Context, t reflect.Type, v reflect.Value) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		// Handle embedded / anonymous structs recursively.
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			if err := bindSpecialFields(c, deref(field.Type), derefValue(fv)); err != nil {
				return err
			}
			continue
		}

		// Path parameters.
		if pathTag := field.Tag.Get("path"); pathTag != "" {
			raw := c.Param(pathTag)
			if raw == "" {
				continue
			}
			if err := setFieldFromStrings(fv, valuesForStringField(fv, raw)); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "BAD_PATH_PARAM",
					Message: fmt.Sprintf("path param '%s': %s", pathTag, err.Error()),
				}
			}
			continue
		}

		// Header parameters.
		if headerTag := field.Tag.Get("header"); headerTag != "" {
			raw := c.Request.Header.Values(headerTag)
			if len(raw) == 0 {
				continue
			}
			if err := setFieldFromStrings(fv, valuesForStringField(fv, raw...)); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "BAD_HEADER",
					Message: fmt.Sprintf("header '%s': %s", headerTag, err.Error()),
				}
			}
			continue
		}

		// Cookie parameters.
		if cookieTag := field.Tag.Get("cookie"); cookieTag != "" {
			raw, err := c.Cookie(cookieTag)
			if err != nil || raw == "" {
				continue
			}
			if err := setFieldFromStrings(fv, valuesForStringField(fv, raw)); err != nil {
				return &Error{
					Status:  http.StatusBadRequest,
					Code:    "BAD_COOKIE",
					Message: fmt.Sprintf("cookie '%s': %s", cookieTag, err.Error()),
				}
			}
		}
	}
	return nil
}

func valuesForStringField(fv reflect.Value, raw ...string) []string {
	if len(raw) == 0 {
		return nil
	}
	if fv.Kind() != reflect.Slice || implementsTextUnmarshalerValue(fv) {
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
	if fv.Kind() == reflect.Slice && !implementsTextUnmarshalerValue(fv) {
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

func hasFormValue(c *gin.Context, name string) bool {
	if c.Request.URL.Query().Has(name) {
		return true
	}
	if _, ok := c.GetPostForm(name); ok {
		return true
	}
	return false
}

type fieldValueSnapshot struct {
	index []int
	value reflect.Value
}

func collectNonBodyFieldValues(t reflect.Type, v reflect.Value) []fieldValueSnapshot {
	var snapshots []fieldValueSnapshot
	collectNonBodyFieldValuesInto(t, v, nil, &snapshots)
	return snapshots
}

func collectNonBodyFieldValuesInto(t reflect.Type, v reflect.Value, prefix []int, snapshots *[]fieldValueSnapshot) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		index := append(append([]int(nil), prefix...), i)
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			collectNonBodyFieldValuesInto(deref(field.Type), derefValue(fv), index, snapshots)
			continue
		}
		if isNonBodyField(field) {
			copyValue := reflect.New(fv.Type()).Elem()
			copyValue.Set(fv)
			*snapshots = append(*snapshots, fieldValueSnapshot{index: index, value: copyValue})
		}
	}
}

func isNonBodyField(field reflect.StructField) bool {
	return field.Tag.Get("path") != "" ||
		field.Tag.Get("form") != "" ||
		field.Tag.Get("header") != "" ||
		field.Tag.Get("cookie") != "" ||
		field.Tag.Get("file") != ""
}

func restoreFieldValues(root reflect.Value, snapshots []fieldValueSnapshot) {
	for _, snapshot := range snapshots {
		field := root.FieldByIndex(snapshot.index)
		if field.CanSet() {
			field.Set(snapshot.value)
		}
	}
}

func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
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
