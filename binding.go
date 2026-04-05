package ninja

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var validate = func() *validator.Validate {
	v := validator.New()
	// Use the "binding" tag (gin convention) instead of the default "validate" tag.
	v.SetTagName("binding")
	return v
}()

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

	// Always bind query-string / form params (uses `form` tags).
	if hasFormFields(t) {
		if err := binding.MapFormWithTag(input, c.Request.URL.Query(), "form"); err != nil {
			return &Error{
				Status:  http.StatusBadRequest,
				Code:    "INVALID_QUERY",
				Message: err.Error(),
			}
		}
	}

	if isMultipartRequest(c) {
		if err := bindMultipartFields(c, t, v); err != nil {
			return err
		}
	}

	// Bind JSON body for mutating methods.
	if isBodyMethod(method) && !isMultipartRequest(c) {
		body, err := io.ReadAll(c.Request.Body)
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
					Message: err.Error(),
				}
			}
		}
	}

	if err := applyDefaults(c, t, v); err != nil {
		return err
	}

	// Run validation.
	if err := validate.Struct(input); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return buildValidationError(ve)
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

func applyDefaults(c *gin.Context, t reflect.Type, v reflect.Value) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := applyDefaults(c, field.Type, fv); err != nil {
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
			Message: err.Error(),
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

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := bindMultipartValue(field.Type, fv, form); err != nil {
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
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := bindSpecialFields(c, field.Type, fv); err != nil {
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
			if err := setFieldFromString(fv, raw); err != nil {
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
			raw := c.GetHeader(headerTag)
			if raw == "" {
				continue
			}
			if err := setFieldFromString(fv, raw); err != nil {
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
			if err := setFieldFromString(fv, raw); err != nil {
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

func setFieldFromStrings(fv reflect.Value, raw []string) error {
	if fv.Kind() == reflect.Slice {
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

func hasFormValue(c *gin.Context, name string) bool {
	if c.Request.URL.Query().Has(name) {
		return true
	}
	if _, ok := c.GetPostForm(name); ok {
		return true
	}
	return false
}

// buildValidationError converts validator.ValidationErrors into our ValidationError type.
func buildValidationError(ve validator.ValidationErrors) *ValidationError {
	errs := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		errs = append(errs, FieldError{
			Field:   strings.ToLower(fe.Field()),
			Message: humanizeValidationError(fe),
		})
	}
	return &ValidationError{Errors: errs}
}

// humanizeValidationError turns a single validator.FieldError into a readable message.
func humanizeValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "len":
		return fmt.Sprintf("length must be exactly %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	default:
		return fmt.Sprintf("failed validation '%s'", fe.Tag())
	}
}
