package settings

import (
	"os"
	"reflect"
	"regexp"
	"strings"
)

// placeholderRe matches ${VAR} and ${VAR:default} tokens.
var placeholderRe = regexp.MustCompile(`\$\{[^}]+\}`)

// expandPlaceholders replaces every ${VAR} and ${VAR:default} token in s with
// the value of the named environment variable.  If the variable is unset or
// empty, the text after the first ':' is used as the default; if no default is
// provided the token is replaced with an empty string.
//
//	expandPlaceholders("${DB_HOST:localhost}")  // → "localhost" when DB_HOST unset
//	expandPlaceholders("${DB_PASS}")            // → "" when DB_PASS unset
//	expandPlaceholders("root@${DB_HOST:127.0.0.1}:3306") // mixed literal + placeholder
func expandPlaceholders(s string) string {
	return placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-1] // strip ${ and }
		name, def, _ := strings.Cut(inner, ":")
		if val := os.Getenv(name); val != "" {
			return val
		}
		return def
	})
}

// expandConfigStrings walks every string field (and map[string]string value)
// in v – which must be a pointer to a struct – and expands
// ${ENV_VAR} / ${ENV_VAR:default} placeholders in place.
func expandConfigStrings(v interface{}) {
	expandValue(reflect.ValueOf(v))
}

func expandValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Ptr:
		if !v.IsNil() {
			expandValue(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			expandValue(v.Field(i))
		}
	case reflect.String:
		if v.CanSet() {
			if expanded := expandPlaceholders(v.String()); expanded != v.String() {
				v.SetString(expanded)
			}
		}
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String && v.Type().Elem().Kind() == reflect.String {
			for _, k := range v.MapKeys() {
				old := v.MapIndex(k).String()
				if expanded := expandPlaceholders(old); expanded != old {
					v.SetMapIndex(k, reflect.ValueOf(expanded))
				}
			}
		}
	}
}
