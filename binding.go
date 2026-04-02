package ninja

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindQuery(input); err != nil {
		// Ignore – struct might not have any form-tagged fields.
		_ = err
	}

	// Bind JSON body for mutating methods.
	if isBodyMethod(method) {
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

// bindSpecialFields walks the struct fields and binds path and header params.
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
		}
	}
	return nil
}

// setFieldFromString converts a raw string value into the target reflect.Value.
func setFieldFromString(fv reflect.Value, raw string) error {
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

// isBodyMethod returns true for methods that carry a request body.
func isBodyMethod(method string) bool {
	m := strings.ToUpper(method)
	return m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch
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
