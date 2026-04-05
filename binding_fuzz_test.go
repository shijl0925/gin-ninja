package ninja

import (
	"reflect"
	"testing"
)

func FuzzSetFieldFromString(f *testing.F) {
	for _, seed := range []string{"42", "true", "3.14", "", "中文", " \n "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		values := []reflect.Value{
			reflect.New(reflect.TypeOf("")).Elem(),
			reflect.New(reflect.TypeOf(int(0))).Elem(),
			reflect.New(reflect.TypeOf(bool(false))).Elem(),
			reflect.New(reflect.TypeOf(float64(0))).Elem(),
		}
		for _, value := range values {
			_ = setFieldFromString(value, raw)
		}
	})
}
